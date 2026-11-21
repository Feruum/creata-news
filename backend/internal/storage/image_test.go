package storage

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

func raster(t *testing.T, format string) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.NRGBA{R: 190, A: 255})
	var b bytes.Buffer
	var err error
	switch format {
	case "jpeg":
		err = jpeg.Encode(&b, img, nil)
	case "gif":
		err = gif.Encode(&b, img, nil)
	default:
		err = png.Encode(&b, img)
	}
	if err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func TestImageValidationRejectsUnsafeInputs(t *testing.T) {
	pngData := raster(t, "png")
	bomb := bytes.Clone(pngData)
	binary.BigEndian.PutUint32(bomb[16:20], 100000)
	binary.BigEndian.PutUint32(bomb[20:24], 100000)
	binary.BigEndian.PutUint32(bomb[29:33], crc32.ChecksumIEEE(bomb[12:29]))
	for _, c := range []struct {
		name, mime string
		data       []byte
	}{
		{"empty", "image/png", nil}, {"byte bound", "image/png", make([]byte, 5*1024*1024+1)},
		{"active mime", "image/svg+xml", []byte("<svg/>")}, {"lying mime", "image/jpeg", pngData},
		{"corrupt body", "image/png", pngData[:40]}, {"pixel bomb", "image/png", bomb},
		{"HTML disguised as image", "image/png", []byte("<html>unsafe</html>")},
	} {
		t.Run(c.name, func(t *testing.T) {
			if _, err := NormalizeImage(context.Background(), c.data, c.mime); err == nil {
				t.Fatal("unsafe input accepted")
			}
		})
	}
}
func TestImageReencodingPreservesPixelsAndDropsTrailingPayload(t *testing.T) {
	for _, format := range []string{"png", "jpeg", "gif"} {
		t.Run(format, func(t *testing.T) {
			input := append(raster(t, format), []byte("<script>metadata-secret</script>")...)
			output, err := NormalizeImage(context.Background(), input, "image/"+format)
			if err != nil {
				t.Fatal(err)
			}
			original, _, err := image.Decode(bytes.NewReader(input))
			if err != nil {
				t.Fatal(err)
			}
			normalized, err := png.Decode(bytes.NewReader(output))
			if err != nil {
				t.Fatal(err)
			}
			if original.At(0, 0) != normalized.At(0, 0) {
				r1, g1, b1, a1 := original.At(0, 0).RGBA()
				r2, g2, b2, a2 := normalized.At(0, 0).RGBA()
				if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
					t.Fatal("decoded pixel changed")
				}
			}
			if bytes.Contains(output, []byte("metadata-secret")) {
				t.Fatal("original payload retained")
			}
			config, err := png.DecodeConfig(bytes.NewReader(output))
			if err != nil || config.Width != 2 || config.Height != 2 {
				t.Fatalf("not a normalized PNG: %v %v", config, err)
			}
		})
	}
}
func TestAnimatedGIFIsRejectedBeforeFrameDecode(t *testing.T) {
	frame := image.NewPaletted(image.Rect(0, 0, 2, 2), color.Palette{color.Black, color.White})
	var b bytes.Buffer
	if err := gif.EncodeAll(&b, &gif.GIF{Image: []*image.Paletted{frame, frame}, Delay: []int{1, 1}}); err != nil {
		t.Fatal(err)
	}
	if _, err := NormalizeImage(context.Background(), b.Bytes(), "image/gif"); err == nil {
		t.Fatal("animated image accepted")
	}
}

func TestStaticWebPIsNormalizedWithoutLosingSupport(t *testing.T) {
	data, err := base64.StdEncoding.DecodeString("UklGRjwAAABXRUJQVlA4IDAAAADQAQCdASoCAAIAAUAmJaACdLoB+AADsAD++AQ//xZyMR2eb/8Cd+BO/Anf+AAAAAA=")
	if err != nil {
		t.Fatal(err)
	}
	out, err := NormalizeImage(context.Background(), data, "image/webp")
	if err != nil {
		t.Fatal(err)
	}
	if c, err := png.DecodeConfig(bytes.NewReader(out)); err != nil || c.Width != 2 || c.Height != 2 {
		t.Fatalf("invalid normalized WebP: %v %v", c, err)
	}
}
func TestAnimatedPNGIsRejected(t *testing.T) {
	data := raster(t, "png")
	chunk := make([]byte, 20)
	binary.BigEndian.PutUint32(chunk[:4], 8)
	copy(chunk[4:8], "acTL")
	binary.BigEndian.PutUint32(chunk[8:12], 2)
	binary.BigEndian.PutUint32(chunk[16:20], crc32.ChecksumIEEE(chunk[4:16]))
	data = append(append(bytes.Clone(data[:33]), chunk...), data[33:]...)
	if _, err := NormalizeImage(context.Background(), data, "image/png"); err == nil {
		t.Fatal("animated PNG accepted")
	}
}

func TestImageEmbeddedMetadataIsRemoved(t *testing.T) {
	data := raster(t, "png")
	payload := []byte("comment\x00metadata-secret")
	chunk := make([]byte, len(payload)+12)
	binary.BigEndian.PutUint32(chunk[:4], uint32(len(payload)))
	copy(chunk[4:8], "tEXt")
	copy(chunk[8:], payload)
	binary.BigEndian.PutUint32(chunk[len(chunk)-4:], crc32.ChecksumIEEE(chunk[4:len(chunk)-4]))
	data = append(append(bytes.Clone(data[:33]), chunk...), data[33:]...)
	out, err := NormalizeImage(context.Background(), data, "image/png")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out, []byte("metadata-secret")) {
		t.Fatal("embedded metadata retained")
	}
}
func TestNormalizedOutputIsAlsoByteBounded(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2048, 2048))
	state := uint32(1)
	for i := range img.Pix {
		state = 1664525*state + 1013904223
		img.Pix[i] = byte(state >> 24)
		if i%4 == 3 {
			img.Pix[i] = 255
		}
	}
	var input bytes.Buffer
	if err := jpeg.Encode(&input, img, &jpeg.Options{Quality: 50}); err != nil {
		t.Fatal(err)
	}
	if input.Len() > MaxImageBytes {
		t.Fatal("fixture must fit input bound")
	}
	if _, err := NormalizeImage(context.Background(), input.Bytes(), "image/jpeg"); err == nil {
		t.Fatal("normalized output exceeded 5MiB without rejection")
	}
}
