package storage

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	_ "golang.org/x/image/webp"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
)

const MaxImageBytes = 5 * 1024 * 1024
const MaxImagePixels = 16 * 1024 * 1024
const MaxImageDimension = 8192

// Bounds JSON base64 overhead as well as decompressed Connect request bodies.
const ImageRPCReadMaxBytes = 8 * 1024 * 1024

var imageDecodeSlots = make(chan struct{}, 2)

// NormalizeImage accepts only a single bounded raster and returns metadata-free PNG bytes.
func NormalizeImage(ctx context.Context, data []byte, contentType string) ([]byte, error) {
	if len(data) == 0 || len(data) > MaxImageBytes {
		return nil, fmt.Errorf("image must contain between 1 byte and 5 MiB")
	}
	formats := map[string]string{"image/jpeg": "jpeg", "image/png": "png", "image/gif": "gif", "image/webp": "webp"}
	expected, ok := formats[contentType]
	if !ok {
		return nil, fmt.Errorf("image type must be JPEG, PNG, WebP, or a single-frame GIF")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || format != expected {
		return nil, fmt.Errorf("image contents do not match the declared type")
	}
	if !boundedDimensions(config.Width, config.Height) {
		return nil, fmt.Errorf("image exceeds dimension or pixel limits")
	}
	if err := requireSingleFrame(data, format); err != nil {
		return nil, err
	}
	select {
	case imageDecodeSlots <- struct{}{}:
		defer func() { <-imageDecodeSlots }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	bounds := decoded.Bounds()
	if !boundedDimensions(bounds.Dx(), bounds.Dy()) {
		return nil, fmt.Errorf("decoded image exceeds pixel limits")
	}
	out := &boundedImageBuffer{}
	if err := png.Encode(out, decoded); err != nil {
		return nil, fmt.Errorf("encode bounded image: %w", err)
	}
	return out.Bytes(), nil
}
func boundedDimensions(width, height int) bool {
	return width > 0 && height > 0 && width <= MaxImageDimension && height <= MaxImageDimension && int64(width)*int64(height) <= MaxImagePixels
}

type boundedImageBuffer struct{ bytes.Buffer }

func (b *boundedImageBuffer) Write(data []byte) (int, error) {
	if b.Len()+len(data) > MaxImageBytes {
		return 0, fmt.Errorf("normalized image exceeds 5 MiB")
	}
	return b.Buffer.Write(data)
}

// Inspect container blocks before allocating decoded frames. Decoding only frame one
// silently accepts animations, while DecodeAll can allocate an unbounded frame count.
func requireSingleFrame(data []byte, format string) error {
	invalid := fmt.Errorf("invalid or animated images are not supported")
	switch format {
	case "png":
		for offset := 8; offset+12 <= len(data); {
			size := uint64(binary.BigEndian.Uint32(data[offset : offset+4]))
			end := uint64(offset) + 12 + size
			if end > uint64(len(data)) {
				return invalid
			}
			kind := string(data[offset+4 : offset+8])
			if kind == "acTL" {
				return invalid
			}
			if kind == "IEND" {
				return nil
			}
			offset = int(end)
		}
		return invalid
	case "webp":
		if len(data) < 12 {
			return invalid
		}
		end := uint64(binary.LittleEndian.Uint32(data[4:8])) + 8
		if end > uint64(len(data)) || end < 12 {
			return invalid
		}
		for offset := uint64(12); offset+8 <= end; {
			size := uint64(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
			next := offset + 8 + size
			if next > end {
				return invalid
			}
			kind := string(data[offset : offset+4])
			if kind == "ANIM" || kind == "ANMF" {
				return invalid
			}
			if kind == "VP8X" {
				if size != 10 || data[offset+8]&2 != 0 {
					return invalid
				}
				chunk := data[offset+8 : next]
				w := 1 + int(chunk[4]) + (int(chunk[5]) << 8) + (int(chunk[6]) << 16)
				h := 1 + int(chunk[7]) + (int(chunk[8]) << 8) + (int(chunk[9]) << 16)
				if !boundedDimensions(w, h) {
					return invalid
				}
			}
			offset = next + (size & 1)
		}
		return nil
	case "gif":
		if len(data) < 13 {
			return invalid
		}
		offset := 13
		if data[10]&0x80 != 0 {
			offset += 3 * (1 << ((data[10] & 7) + 1))
		}
		frames := 0
		skipBlocks := func() error {
			for {
				if offset >= len(data) {
					return io.ErrUnexpectedEOF
				}
				size := int(data[offset])
				offset++
				if size == 0 {
					return nil
				}
				offset += size
				if offset > len(data) {
					return io.ErrUnexpectedEOF
				}
			}
		}
		for offset < len(data) {
			kind := data[offset]
			offset++
			switch kind {
			case 0x3b:
				if frames == 1 {
					return nil
				}
				return invalid
			case 0x21:
				offset++
				if err := skipBlocks(); err != nil {
					return invalid
				}
			case 0x2c:
				frames++
				if frames > 1 || offset+9 > len(data) {
					return invalid
				}
				if !boundedDimensions(int(binary.LittleEndian.Uint16(data[offset+4:offset+6])), int(binary.LittleEndian.Uint16(data[offset+6:offset+8]))) {
					return invalid
				}
				packed := data[offset+8]
				offset += 9
				if packed&0x80 != 0 {
					offset += 3 * (1 << ((packed & 7) + 1))
				}
				offset++
				if err := skipBlocks(); err != nil {
					return invalid
				}
			default:
				return invalid
			}
		}
		return invalid
	}
	return nil
}
