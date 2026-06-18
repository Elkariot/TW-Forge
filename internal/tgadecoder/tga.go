package tgadecoder

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
)

// Decode декодирует TGA файл. Поддерживает тип 2 (uncompressed) и 10 (RLE),
// глубину 24 и 32 бита (BGR/BGRA) — стандарт для RTW.
func Decode(data []byte) (image.Image, error) {
	if len(data) < 18 {
		return nil, fmt.Errorf("tga: file too short")
	}

	idLen := int(data[0])
	imgType := data[2]
	width := int(binary.LittleEndian.Uint16(data[12:14]))
	height := int(binary.LittleEndian.Uint16(data[14:16]))
	bpp := int(data[16])
	descriptor := data[17]

	if width == 0 || height == 0 {
		return nil, fmt.Errorf("tga: invalid dimensions %dx%d", width, height)
	}
	if imgType != 2 && imgType != 10 {
		return nil, fmt.Errorf("tga: unsupported image type %d", imgType)
	}
	if bpp != 24 && bpp != 32 {
		return nil, fmt.Errorf("tga: unsupported bpp %d", bpp)
	}

	pixelBytes := bpp / 8
	payload := data[18+idLen:]

	pixels := make([]byte, width*height*pixelBytes)
	if imgType == 2 {
		if len(payload) < len(pixels) {
			return nil, fmt.Errorf("tga: not enough pixel data")
		}
		copy(pixels, payload[:len(pixels)])
	} else {
		// RLE
		out := 0
		in := 0
		for out < len(pixels) {
			if in >= len(payload) {
				return nil, fmt.Errorf("tga: unexpected end of RLE data")
			}
			pkt := payload[in]
			in++
			count := int(pkt&0x7F) + 1
			if pkt&0x80 != 0 {
				// run-length packet
				if in+pixelBytes > len(payload) {
					return nil, fmt.Errorf("tga: RLE run overflows data")
				}
				px := payload[in : in+pixelBytes]
				in += pixelBytes
				for i := 0; i < count; i++ {
					copy(pixels[out:], px)
					out += pixelBytes
				}
			} else {
				// raw packet
				n := count * pixelBytes
				if in+n > len(payload) {
					return nil, fmt.Errorf("tga: raw packet overflows data")
				}
				copy(pixels[out:], payload[in:in+n])
				in += n
				out += n
			}
		}
	}

	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	flipV := descriptor&0x20 == 0 // бит 5 = origin: 0 = bottom-left

	for y := 0; y < height; y++ {
		srcY := y
		if flipV {
			srcY = height - 1 - y
		}
		for x := 0; x < width; x++ {
			i := (srcY*width + x) * pixelBytes
			b := pixels[i]
			g := pixels[i+1]
			r := pixels[i+2]
			a := uint8(255)
			if pixelBytes == 4 {
				a = pixels[i+3]
			}
			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: a})
		}
	}

	return img, nil
}
