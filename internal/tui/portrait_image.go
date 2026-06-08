package tui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"os"
	"strings"
)

// truecolorSupported reports whether the terminal supports 24-bit color.
func truecolorSupported() bool {
	ct := os.Getenv("COLORTERM")
	return ct == "truecolor" || ct == "24bit"
}

// scaleArea scales src to dstW×dstH using box-filter (area-average) downsampling.
// Produces much cleaner results than nearest-neighbor when shrinking images.
func scaleArea(src image.Image, dstW, dstH int) *image.RGBA {
	sb := src.Bounds()
	srcW, srcH := sb.Dx(), sb.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	for y := 0; y < dstH; y++ {
		y0 := y * srcH / dstH
		y1 := (y+1)*srcH/dstH + 1
		if y1 > srcH {
			y1 = srcH
		}
		for x := 0; x < dstW; x++ {
			x0 := x * srcW / dstW
			x1 := (x+1)*srcW/dstW + 1
			if x1 > srcW {
				x1 = srcW
			}
			var r, g, b, a uint64
			n := uint64((x1 - x0) * (y1 - y0))
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					pr, pg, pb, pa := src.At(sb.Min.X+sx, sb.Min.Y+sy).RGBA()
					r += uint64(pr)
					g += uint64(pg)
					b += uint64(pb)
					a += uint64(pa)
				}
			}
			dst.Set(x, y, color.RGBA64{
				R: uint16(r / n),
				G: uint16(g / n),
				B: uint16(b / n),
				A: uint16(a / n),
			})
		}
	}
	return dst
}

// pngToHalfBlocks converts PNG data to a string of colored Unicode half-block
// characters (▀) scaled to cols×rows terminal cells. Each cell encodes two
// vertical pixels: top pixel as fg, bottom pixel as bg. Returns "" if truecolor
// is not supported or the image cannot be decoded.
func pngToHalfBlocks(data []byte, cols, rows int) string {
	if !truecolorSupported() {
		return ""
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return ""
	}

	// scale to cols wide × (rows*2) tall so each pair of pixel rows = one terminal row
	scaled := scaleArea(img, cols, rows*2)

	var sb strings.Builder
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			// RGBA() returns pre-multiplied 16-bit values; >>8 gives 8-bit composite on black
			tr, tg, tb, _ := scaled.At(col, row*2).RGBA()
			br, bg8, bb, _ := scaled.At(col, row*2+1).RGBA()
			fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀",
				tr>>8, tg>>8, tb>>8, br>>8, bg8>>8, bb>>8)
		}
		sb.WriteString("\x1b[0m\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}
