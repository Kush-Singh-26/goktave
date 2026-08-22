package thumbnail

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"os"
	"os/exec"

	lipgloss "charm.land/lipgloss/v2"
)

// cellAspect is the approximate height:width ratio of a terminal cell.
// Two stacked pixels are shown per cell via half-blocks.
const cellAspect = 2.0

// Render downloads the artwork at url and renders it as a truecolor
// half-block grid exactly `cols` cells wide (rows derived from the
// aspect ratio). The output always has a deterministic shape and size,
// independent of any external converter.
func Render(url string, cols int) (string, error) {
	if url == "" {
		return "", fmt.Errorf("empty url")
	}
	if cols < 4 {
		cols = 4
	}
	rows := int(float64(cols) / cellAspect)
	if rows < 2 {
		rows = 2
	}

	pngData, err := fetchProcessedPNG(url)
	if err != nil {
		return "", err
	}

	img, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		return "", fmt.Errorf("failed to decode artwork: %v", err)
	}

	return renderHalfBlocks(img, cols, rows), nil
}

// fetchProcessedPNG downloads the image and runs it through ffmpeg to
// produce a square-cropped, contrast-boosted PNG of fixed resolution.
func fetchProcessedPNG(url string) ([]byte, error) {
	rawTmp, err := os.CreateTemp("", "goktave-raw-*.img")
	if err != nil {
		return nil, fmt.Errorf("failed to create raw temp file: %v", err)
	}
	rawPath := rawTmp.Name()
	defer os.Remove(rawPath)

	// User-Agent avoids 400 errors from YT Music CDN
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		rawTmp.Close()
		return nil, fmt.Errorf("failed to download: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		rawTmp.Close()
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	if _, err := io.Copy(rawTmp, resp.Body); err != nil {
		rawTmp.Close()
		return nil, fmt.Errorf("failed to save download: %v", err)
	}
	rawTmp.Close()

	pngTmp, err := os.CreateTemp("", "goktave-thumb-*.png")
	if err != nil {
		return nil, fmt.Errorf("failed to create png temp file: %v", err)
	}
	pngPath := pngTmp.Name()
	pngTmp.Close()
	defer os.Remove(pngPath)

	filters := "crop=min(iw\\,ih):min(iw\\,ih),scale=512:512,eq=contrast=1.08:saturation=1.15"
	convCmd := exec.Command("ffmpeg", "-y", "-i", rawPath, "-vf", filters, "-vframes", "1", pngPath)
	var stderr bytes.Buffer
	convCmd.Stderr = &stderr
	if err := convCmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg pre-processing failed: %v (stderr: %s)", err, stderr.String())
	}

	return os.ReadFile(pngPath)
}

// renderHalfBlocks maps the image onto a cols x rows cell grid using
// bilinear sampling. Each cell shows two vertically stacked pixels via
// the '▀' glyph: upper half as foreground, lower half as background.
func renderHalfBlocks(img image.Image, cols, rows int) string {
	b := img.Bounds()
	iw, ih := b.Dx(), b.Dy()
	if iw <= 0 || ih <= 0 {
		return ""
	}

	pw := cols     // pixel columns == cell columns
	ph := rows * 2 // pixel rows: two per cell

	var out bytes.Buffer
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			tr, tg, tb := sampleBilinear(img, b, iw, ih, pw, ph, x*2+1, y*2+1)
			br_, bg_, bb_ := sampleBilinear(img, b, iw, ih, pw, ph, x*2+1, y*2+2)
			out.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", tr, tg, tb))).
				Background(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", br_, bg_, bb_))).
				Render("▀"))
		}
		if y < rows-1 {
			out.WriteByte('\n')
		}
	}
	return out.String()
}

// sampleBilinear samples the source image at logical pixel coordinates
// (px, py) within a pw x ph target grid, mapping into the source bounds
// with bilinear interpolation.
func sampleBilinear(img image.Image, b image.Rectangle, iw, ih, pw, ph, px, py int) (uint8, uint8, uint8) {
	fx := (float64(px) - 0.5) * float64(iw) / float64(pw)
	fy := (float64(py) - 0.5) * float64(ih) / float64(ph)

	x0 := int(fx)
	y0 := int(fy)
	xd := fx - float64(x0)
	yd := fy - float64(y0)

	x1, y1 := x0+1, y0+1
	if x1 >= iw {
		x1 = iw - 1
	}
	if y1 >= ih {
		y1 = ih - 1
	}
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}

	c00 := rgbaAt(img, b, x0, y0)
	c10 := rgbaAt(img, b, x1, y0)
	c01 := rgbaAt(img, b, x0, y1)
	c11 := rgbaAt(img, b, x1, y1)

	r := lerp(lerp(c00[0], c10[0], xd), lerp(c01[0], c11[0], xd), yd)
	g := lerp(lerp(c00[1], c10[1], xd), lerp(c01[1], c11[1], xd), yd)
	bl := lerp(lerp(c00[2], c10[2], xd), lerp(c01[2], c11[2], xd), yd)

	return uint8(r), uint8(g), uint8(bl)
}

// rgbaAt returns the 8-bit RGB at logical source pixel (x, y). Go's
// RGBA() values are alpha-premultiplied, so transparent pixels blend
// onto black naturally.
func rgbaAt(img image.Image, b image.Rectangle, x, y int) [3]float64 {
	r, g, bb, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
	return [3]float64{
		float64(r >> 8),
		float64(g >> 8),
		float64(bb >> 8),
	}
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}
