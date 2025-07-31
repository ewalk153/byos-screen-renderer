package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/osteele/liquid"
)

var (
	engine        = liquid.NewEngine()
	templatePath  = "template.liquid"
	screenshotPNG = "screenshot.png"
	outputBasePNG = "output.png"
	mutex         sync.Mutex // protects access to output.png
)

func main() {
	if envOutputPNG := os.Getenv("OUTPUT_PATH"); envOutputPNG != "" {
		outputBasePNG = envOutputPNG
	}
	http.HandleFunc("/render/{slug}", handleRender)
	http.HandleFunc("/up", healthCheck)
	http.HandleFunc("/screenshot.png/{slug}", serveScreenshot) // still mounted here for backward compat

	fmt.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK")
}

func handleRender(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST supported", http.StatusMethodNotAllowed)
		return
	}
	slug := r.PathValue("slug")

	defer r.Body.Close()
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	var templateBytes []byte
	var err error
	// if data["template"].(string) != "" {
	// 	templateBytes = []byte(data["template"].(string))
	// } else {
	// }
	templateBytes, err = os.ReadFile(templatePath)
	if err != nil {
		http.Error(w, "Failed to read template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := engine.ParseString(string(templateBytes))
	if err != nil {
		http.Error(w, "Failed to parse template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	renderedBytes, err := tmpl.Render(data)
	if err != nil {
		http.Error(w, "Failed to render template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	renderedHTML := string(renderedBytes)

	go func() {
		if err := generateScreenshot(renderedHTML, slug); err != nil {
			log.Println("Screenshot generation failed:", err)
		}
	}()

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Rendering started. Visit /screenshot.png to retrieve the result."))
}

func serveScreenshot(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	outputPNG := outputBasePNG
	if slug != "" {
		outputPNG = slug + "-" + outputPNG
	}
	mutex.Lock()
	defer mutex.Unlock()

	file, err := os.Open(outputPNG)
	if err != nil {
		log.Println("Screenshot serve failed:", err)
		http.Error(w, "Screenshot not ready", http.StatusNotFound)
		return
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		http.Error(w, "Failed to get file size", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
	io.Copy(w, file)
}

func generateScreenshot(html string, slug string) error {
	outputPNG := outputBasePNG
	if slug != "" {
		outputPNG = slug + "-" + outputPNG
	}
	opts := chromedp.DefaultExecAllocatorOptions[:]
	if path := os.Getenv("CHROMIUM_PATH"); path != "" {
		opts = append(opts, chromedp.ExecPath(path))
	}
	opts = append(opts,
		chromedp.NoSandbox,
		chromedp.Headless,
		chromedp.DisableGPU,
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	tmpFile, err := os.CreateTemp("", "*.html")
	if err != nil {
		log.Println("Failed create temp file:", err)
		return err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(html)); err != nil {
		log.Println("Failed write temp file:", err)
		return err
	}
	tmpFile.Close()

	var buf []byte
	url := "file://" + tmpFile.Name()

	if err := chromedp.Run(ctx,
		chromedp.EmulateViewport(800, 480),
		chromedp.Navigate(url),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.CaptureScreenshot(&buf),
	); err != nil {
		log.Println("Failed chrome session", err)
		return err
	}

	if err := os.WriteFile(screenshotPNG, buf, 0644); err != nil {
		log.Println("Failed write final screenshot", err)
		return err
	}
	defer os.Remove(screenshotPNG)

	mutex.Lock()
	defer mutex.Unlock()

	convertArgs := []string{
		screenshotPNG,
		"-dither", "FloydSteinberg",
		"-remap", "pattern:gray50",
		"-depth", "1",
		"-strip",
		"png:" + outputPNG,
	}
	cmd := exec.Command("convert", convertArgs...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Println("ImageMagick convert failed:", err)
		return fmt.Errorf("ImageMagick convert failed: %v\n%s", err, stderr.String())
	}

	return nil
}
