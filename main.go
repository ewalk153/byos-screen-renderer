package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/osteele/liquid"
)

func main() {
	// Load Liquid template
	templateBytes, err := os.ReadFile("template.liquid")
	if err != nil {
		log.Fatalf("Failed to read Liquid template: %v", err)
	}

	// Compile and render the Liquid template with sample data
	engine := liquid.NewEngine()
	tmpl, err := engine.ParseString(string(templateBytes))
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}

	data := map[string]interface{}{
		"title":   "Liquid Screenshot",
		"date": "Jul 26, 2025",
	}

	renderedHTML, err := tmpl.Render(data)
	if err != nil {
		log.Fatalf("Failed to render template: %v", err)
	}

	// Create a context for headless Chrome
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Write the rendered HTML to a temp file
	tmpFile, err := os.CreateTemp("", "*.html")
	if err != nil {
		log.Fatalf("Failed to create temp HTML file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(renderedHTML)); err != nil {
		log.Fatalf("Failed to write HTML to temp file: %v", err)
	}
	tmpFile.Close()

	// Capture screenshot
	var buf []byte
	url := "file://" + tmpFile.Name()

	err = chromedp.Run(ctx,
		chromedp.EmulateViewport(800, 480),
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

	// Convert to 1-bit BMP using ImageMagick
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
