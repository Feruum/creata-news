package registrationfiles

import (
	"archive/zip"
	"bytes"
	"image"
	"image/png"
	"testing"
	"time"

	pb "github.com/studyverse/ems-backend/gen/registrationforms/v1"
)

func TestValidateBytesRejectsSpoofedAndOversizedFiles(t *testing.T) {
	var imageBytes bytes.Buffer
	_ = png.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	for _, c := range []struct {
		name, mime string
		data       []byte
	}{
		{"fake.pdf", "application/pdf", []byte("<html>hello</html>")},
		{"fake.png", "image/png", []byte("MZ executable")},
		{"wrong.jpg", "image/jpeg", imageBytes.Bytes()},
		{"broken.png", "image/png", imageBytes.Bytes()[:24]},
		{"big.pdf", "application/pdf", make([]byte, MaxBytes+1)},
		{"fake.docx", docxMIME, officeFixture(t, "xl/workbook.xml")},
	} {
		if err := validateBytes(c.name, c.mime, c.data); err == nil {
			t.Fatalf("accepted %s", c.name)
		}
	}
	if err := validateBytes("ok.png", "image/png", imageBytes.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := validateBytes("ok.pdf", "application/pdf", []byte("%PDF-1.7\n1 0 obj\n<< /Type /Catalog >>\nendobj\ntrailer\n<< /Root 1 0 R >>\n%%EOF")); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ name, mime, part string }{{"ok.docx", docxMIME, "word/document.xml"}, {"ok.xlsx", xlsxMIME, "xl/workbook.xml"}} {
		if err := validateBytes(c.name, c.mime, officeFixture(t, c.part)); err != nil {
			t.Fatal(err)
		}
	}
}

func officeFixture(t *testing.T, part string) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for name, data := range map[string]string{"[Content_Types].xml": "<Types xmlns=\"http://schemas.openxmlformats.org/package/2006/content-types\"><Override PartName=\"/" + part + "\" ContentType=\"" + map[string]string{"word/document.xml": "application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml", "xl/workbook.xml": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"}[part] + "\"/></Types>", "_rels/.rels": "<Relationships xmlns=\"http://schemas.openxmlformats.org/package/2006/relationships\"><Relationship Id=\"rId1\" Type=\"http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument\" Target=\"" + part + "\"/></Relationships>", part: "<document/>"} {
		w, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(data))
	}
	_ = z.Close()
	return b.Bytes()
}

func TestOwnerAndAttachmentIdentityChecks(t *testing.T) {
	valid := Owner{UserID: 7}
	now := time.Now()
	for _, o := range []Owner{{}, {UserID: -1}, {UserID: 7, AccountID: "f2a91935-5766-43a4-9252-80027295d170"}, {AccountID: "bad"}} {
		if o.valid() {
			t.Fatalf("accepted bogus owner %#v", o)
		}
	}
	u := &upload{ID: "f2a91935-5766-43a4-9252-80027295d170", EventID: 2, FormVersionID: "v", FieldID: "file", OwnerUserID: 7, State: "ready", ExpiresAt: now.Add(time.Hour), FileName: "ok.pdf", ContentType: "application/pdf", SizeBytes: 80, ObjectKey: "final/unique"}
	f := &pb.FormField{Id: "file", Type: pb.FormFieldType_FORM_FIELD_TYPE_FILE}
	if err := validateAttachment(u, 2, "v", valid, f, now); err != nil {
		t.Fatal(err)
	}
	mutations := []func(*upload){func(u *upload) { u.OwnerUserID = 8 }, func(u *upload) { u.EventID = 3 }, func(u *upload) { u.FormVersionID = "old" }, func(u *upload) { u.FieldID = "other" }, func(u *upload) { u.State = "uploading" }, func(u *upload) { u.State = "attached" }, func(u *upload) { u.ExpiresAt = now.Add(-time.Second) }, func(u *upload) { u.SizeBytes = MaxBytes + 1 }}
	for i, mutate := range mutations {
		copy := *u
		mutate(&copy)
		if validateAttachment(&copy, 2, "v", valid, f, now) == nil {
			t.Fatalf("accepted invalid attachment %d", i)
		}
	}
	if validateAttachment(u, 2, "v", Owner{AccountID: "f2a91935-5766-43a4-9252-80027295d170"}, f, now) == nil {
		t.Fatal("cross-namespace access")
	}
}

func TestFileRulesOnlyNarrow(t *testing.T) {
	for _, rules := range []*pb.FormFileRules{{MaxBytes: MaxBytes + 1}, {MaxFiles: 4}, {Extensions: []string{"exe"}}, {MaxBytes: -1}, {MaxFiles: -1}} {
		if _, _, _, err := limits(rules); err == nil {
			t.Fatalf("invalid rules accepted %v", rules)
		}
	}
	if _, _, _, err := limits(&pb.FormFileRules{MaxBytes: 1000, MaxFiles: 1, Extensions: []string{"pdf"}}); err != nil {
		t.Fatal(err)
	}
	if got := safeFilename("../../evil\r\n.pdf"); got != "evil.pdf" {
		t.Fatal(got)
	}
}
