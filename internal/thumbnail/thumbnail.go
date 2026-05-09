package thumbnail

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

func GetASCII(url string, width int) (string, error) {
	if url == "" {
		return "", fmt.Errorf("empty url")
	}

	bin := "ascii-image-converter"
	if _, err := exec.LookPath(bin); err != nil {
		home, _ := os.UserHomeDir()
		goBin := filepath.Join(home, "go", "bin", "ascii-image-converter")
		if _, err := os.Stat(goBin); err == nil {
			bin = goBin
		}
	}

	// Create a temp file for the raw download
	rawTmp, err := os.CreateTemp("", "goktave-raw-*.img")
	if err != nil {
		return "", fmt.Errorf("failed to create raw temp file: %v", err)
	}
	rawPath := rawTmp.Name()
	defer os.Remove(rawPath)

	// Download with User-Agent to avoid 400 errors from YT Music
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		rawTmp.Close()
		return "", fmt.Errorf("failed to download: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		rawTmp.Close()
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	if _, err := io.Copy(rawTmp, resp.Body); err != nil {
		rawTmp.Close()
		return "", fmt.Errorf("failed to save download: %v", err)
	}
	rawTmp.Close()

	// Create a temp file for the PNG output
	pngTmp, err := os.CreateTemp("", "goktave-thumb-*.png")
	if err != nil {
		return "", fmt.Errorf("failed to create png temp file: %v", err)
	}
	pngPath := pngTmp.Name()
	pngTmp.Close()
	defer os.Remove(pngPath)

	// Pre-process with ffmpeg:
	filters := "crop=min(iw\\,ih):min(iw\\,ih),scale=200:200,boxblur=1:1,eq=contrast=1.2:brightness=0.02"
	convCmd := exec.Command("ffmpeg", "-y", "-i", rawPath, "-vf", filters, "-vframes", "1", pngPath)
	if err := convCmd.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg pre-processing failed: %v", err)
	}

	// Flags: 
	cmd := exec.Command(bin, pngPath, "-C", "--color-bg", "--width", fmt.Sprint(width))
	
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ascii-image-converter failed: %v (stderr: %s)", err, stderr.String())
	}
	
	return out.String(), nil
}
