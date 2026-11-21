package registrationfiles

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"path"
	"strings"
	"unicode"

	pb "github.com/studyverse/ems-backend/gen/registrationforms/v1"
	_ "golang.org/x/image/webp"
)

const MaxBytes int64 = 10 * 1024 * 1024
const docxMIME = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
const xlsxMIME = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

var allowedTypes = map[string]string{"pdf": "application/pdf", "jpg": "image/jpeg", "jpeg": "image/jpeg", "png": "image/png", "webp": "image/webp", "docx": docxMIME, "xlsx": xlsxMIME}

func safeFilename(name string) string {
	name = path.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == '"' || r == ';' {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if len(name) > 200 {
		ext := path.Ext(name)
		name = name[:180] + ext
	}
	if name == "" || name == "." || name == ".." {
		return "file"
	}
	return name
}

func limits(rules *pb.FormFileRules) (map[string]bool, int64, int, error) {
	exts := map[string]bool{}
	for ext := range allowedTypes {
		exts[ext] = true
	}
	size := MaxBytes
	count := 3
	if rules == nil {
		return exts, size, count, nil
	}
	if rules.MaxBytes < 0 || rules.MaxBytes > MaxBytes || rules.MaxFiles < 0 || rules.MaxFiles > 3 {
		return nil, 0, 0, fmt.Errorf("file rules exceed supported limits")
	}
	if rules.MaxBytes > 0 {
		size = rules.MaxBytes
	}
	if rules.MaxFiles > 0 {
		count = int(rules.MaxFiles)
	}
	if len(rules.Extensions) > 0 {
		exts = map[string]bool{}
		for _, ext := range rules.Extensions {
			ext = strings.TrimPrefix(strings.ToLower(ext), ".")
			if _, ok := allowedTypes[ext]; !ok {
				return nil, 0, 0, fmt.Errorf("unsupported file extension")
			}
			exts[ext] = true
		}
	}
	return exts, size, count, nil
}

func validateMetadata(field *pb.FormField, name, mime string, size int64) error {
	if field == nil || field.Type != pb.FormFieldType_FORM_FIELD_TYPE_FILE {
		return fmt.Errorf("unknown file question")
	}
	exts, max, _, err := limits(field.FileRules)
	if err != nil {
		return err
	}
	ext := strings.TrimPrefix(strings.ToLower(path.Ext(name)), ".")
	if !exts[ext] || allowedTypes[ext] != mime {
		return fmt.Errorf("file format does not match allowed extension and content type")
	}
	if size <= 0 || size > max {
		return fmt.Errorf("file exceeds question size limit")
	}
	return nil
}

func validateBytes(name, mime string, data []byte) error {
	field := &pb.FormField{Type: pb.FormFieldType_FORM_FIELD_TYPE_FILE}
	if err := validateMetadata(field, name, mime, int64(len(data))); err != nil {
		return err
	}
	switch mime {
	case "application/pdf":
		if !bytes.HasPrefix(data, []byte("%PDF-1.")) && !bytes.HasPrefix(data, []byte("%PDF-2.")) {
			return fmt.Errorf("invalid PDF header")
		}
		tail := bytes.TrimSpace(data)
		if !bytes.HasSuffix(tail, []byte("%%EOF")) || !bytes.Contains(data, []byte(" obj")) || !bytes.Contains(data, []byte("endobj")) || (!bytes.Contains(data, []byte("trailer")) && !bytes.Contains(data, []byte("/Type /XRef"))) {
			return fmt.Errorf("invalid PDF structure")
		}
	case docxMIME, xlsxMIME:
		return validateOffice(data, mime)
	default:
		cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
		expected := map[string]string{"image/png": "png", "image/jpeg": "jpeg", "image/webp": "webp"}[mime]
		if err != nil || format != expected || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 8192 || cfg.Height > 8192 || int64(cfg.Width)*int64(cfg.Height) > 16*1024*1024 {
			return fmt.Errorf("invalid image contents or dimensions")
		}
		if _, _, err = image.Decode(bytes.NewReader(data)); err != nil {
			return fmt.Errorf("invalid image body: %w", err)
		}
	}
	return nil
}

func validateOffice(data []byte, mime string) error {
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("invalid Office archive")
	}
	part, kind := "word/document.xml", "application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"
	if mime == xlsxMIME {
		part = "xl/workbook.xml"
		kind = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"
	}
	entries := map[string]*zip.File{}
	var total uint64
	if len(z.File) > 2048 {
		return fmt.Errorf("too many Office archive entries")
	}
	for _, f := range z.File {
		name := f.Name
		lower := strings.ToLower(name)
		total += f.UncompressedSize64
		if total > 50*1024*1024 || f.UncompressedSize64 > 20*1024*1024 || strings.HasPrefix(name, "/") || strings.Contains(name, "\\") || path.Clean(name) != strings.TrimSuffix(name, "/") || entries[name] != nil || strings.Contains(lower, "vbaproject") || strings.HasSuffix(lower, ".exe") {
			return fmt.Errorf("unsafe Office archive")
		}
		entries[name] = f
	}
	if entries[part] == nil || entries["[Content_Types].xml"] == nil || entries["_rels/.rels"] == nil {
		return fmt.Errorf("Office document layout does not match extension")
	}
	foundType, foundRel := false, false
	for _, name := range []string{"[Content_Types].xml", "_rels/.rels", part} {
		r, err := entries[name].Open()
		if err != nil {
			return err
		}
		decoder := xml.NewDecoder(io.LimitReader(r, 20*1024*1024+1))
		hasRoot := false
		for {
			token, e := decoder.Token()
			if e == io.EOF {
				break
			}
			if e != nil {
				_ = r.Close()
				return fmt.Errorf("invalid Office XML")
			}
			start, ok := token.(xml.StartElement)
			if !ok {
				continue
			}
			hasRoot = true
			attrs := map[string]string{}
			for _, a := range start.Attr {
				attrs[a.Name.Local] = a.Value
			}
			if name == "[Content_Types].xml" && start.Name.Local == "Override" && attrs["PartName"] == "/"+part && attrs["ContentType"] == kind {
				foundType = true
			}
			if name == "_rels/.rels" && start.Name.Local == "Relationship" && attrs["Type"] == "http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" && strings.TrimPrefix(attrs["Target"], "/") == part && attrs["TargetMode"] != "External" {
				foundRel = true
			}
		}
		_ = r.Close()
		if !hasRoot {
			return fmt.Errorf("empty Office XML")
		}
	}
	if !foundType || !foundRel {
		return fmt.Errorf("Office package type or relationship does not match extension")
	}
	return nil
}
