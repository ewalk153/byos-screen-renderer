package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	// Read HTML content
	// htmlBytes, err := ioutil.ReadFile("input.html")
	htmlBytes, err := ioutil.ReadFile("template2.html")
	if err != nil {
		log.Fatalf("Failed to read input.html: %v", err)
	}
	html := string(htmlBytes)

	// Create context with timeout
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Write temp HTML file
	tmpFile, err := os.CreateTemp("", "*.html")
	if err != nil {
		log.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(html)); err != nil {
		log.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Screenshot buffer
	var buf []byte
	url := "file://" + tmpFile.Name()

	// Run headless Chrome to capture PNG
	err = chromedp.Run(ctx,
		chromedp.EmulateViewport(840, 400),
		chromedp.Navigate(url),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.CaptureScreenshot(&buf),
	)
	if err != nil {
		log.Fatalf("Failed to capture screenshot: %v", err)
	}

	// Save PNG
	pngFile := "screenshot.png"
	err = os.WriteFile(pngFile, buf, 0644)
	if err != nil {
		log.Fatalf("Failed to write PNG file: %v", err)
	}
	fmt.Println("PNG screenshot saved as", pngFile)

	// Convert PNG to 1-bit BMP using ImageMagick
	bmpFile := "screenshot.bmp"
	convertArgs := []string{
		pngFile,
		"-dither", "FloydSteinberg",
		"-remap", "pattern:gray50",
		"-depth", "1",
		"-colors", "2",
		"-strip",
		bmpFile,
	}
	cmd := exec.Command("convert", convertArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("ImageMagick convert failed: %v\n%s", err, output)
	}

	fmt.Println("1-bit BMP saved as", bmpFile)
}
