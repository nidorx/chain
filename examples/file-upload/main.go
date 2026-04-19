// Package main demonstrates file upload handling in Chain.
//
// Run with: go run main.go
// Then try:
//
//	# Upload a single file
//	curl -X POST http://localhost:8080/upload \
//	  -F "file=@./router.go"
//
//	# Upload with metadata
//	curl -X POST http://localhost:8080/upload/metadata \
//	  -F "file=@./router.go" \
//	  -F "description=Source code" \
//	  -F "tags=golang,example"
//
//	# Upload multiple files
//	curl -X POST http://localhost:8080/upload/multiple \
//	  -F "files=@./router.go" \
//	  -F "files=@./pubsub/pubsub.go"
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/nidorx/chain"
)

const (
	maxUploadSize = 10 << 20 // 10 MB
	uploadDir     = "./uploads"
)

func main() {
	// Create upload directory
	os.MkdirAll(uploadDir, 0755)

	router := chain.New()

	// Limit request body size globally
	router.Use(chain.MaxBytesMiddleware(maxUploadSize))

	// ── Single file upload ─────────────────────────────────────────────
	router.POST("/upload", func(ctx *chain.Context) error {
		// Parse multipart form (max 10 MB)
		if err := ctx.Request.ParseMultipartForm(maxUploadSize); err != nil {
			ctx.Json(map[string]string{"error": "failed to parse form"})
			ctx.Status(http.StatusBadRequest)
			return nil
		}

		file, header, err := ctx.Request.FormFile("file")
		if err != nil {
			ctx.Json(map[string]string{"error": "no file uploaded"})
			ctx.Status(http.StatusBadRequest)
			return nil
		}
		defer file.Close()

		// Validate file type
		ext := strings.ToLower(filepath.Ext(header.Filename))
		allowedExts := map[string]bool{
			".txt": true, ".go": true, ".md": true, ".json": true, ".png": true, ".jpg": true,
		}
		if !allowedExts[ext] {
			ctx.Json(map[string]string{"error": "file type not allowed"})
			ctx.Status(http.StatusBadRequest)
			return nil
		}

		// Save file
		destPath := filepath.Join(uploadDir, header.Filename)
		dest, err := os.Create(destPath)
		if err != nil {
			ctx.Json(map[string]string{"error": "failed to save file"})
			ctx.InternalServerError()
			return nil
		}
		defer dest.Close()

		written, err := io.Copy(dest, file)
		if err != nil {
			ctx.Json(map[string]string{"error": "failed to write file"})
			ctx.InternalServerError()
			return nil
		}

		ctx.Json(map[string]any{
			"message":  "file uploaded successfully",
			"filename": header.Filename,
			"size":     written,
			"path":     destPath,
		})
		return nil
	})

	// ── File upload with metadata ──────────────────────────────────────
	router.POST("/upload/metadata", func(ctx *chain.Context) error {
		if err := ctx.Request.ParseMultipartForm(maxUploadSize); err != nil {
			ctx.Json(map[string]string{"error": "failed to parse form"})
			ctx.Status(http.StatusBadRequest)
			return nil
		}

		// Get file
		file, header, err := ctx.Request.FormFile("file")
		if err != nil {
			ctx.Json(map[string]string{"error": "no file uploaded"})
			ctx.Status(http.StatusBadRequest)
			return nil
		}
		defer file.Close()

		// Get form fields
		description := ctx.Request.FormValue("description")
		tags := ctx.Request.Form["tags"]

		// Save file
		destPath := filepath.Join(uploadDir, header.Filename)
		dest, err := os.Create(destPath)
		if err != nil {
			ctx.Json(map[string]string{"error": "failed to save file"})
			ctx.InternalServerError()
			return nil
		}
		defer dest.Close()

		written, _ := io.Copy(dest, file)

		ctx.Json(map[string]any{
			"message":     "file uploaded with metadata",
			"filename":    header.Filename,
			"size":        written,
			"description": description,
			"tags":        tags,
		})
		return nil
	})

	// ── Multiple file upload ───────────────────────────────────────────
	router.POST("/upload/multiple", func(ctx *chain.Context) error {
		if err := ctx.Request.ParseMultipartForm(maxUploadSize); err != nil {
			ctx.Json(map[string]string{"error": "failed to parse form"})
			ctx.Status(http.StatusBadRequest)
			return nil
		}

		form := ctx.Request.MultipartForm
		files := form.File["files"]

		if len(files) == 0 {
			ctx.Json(map[string]string{"error": "no files uploaded"})
			ctx.Status(http.StatusBadRequest)
			return nil
		}

		if len(files) > 5 {
			ctx.Json(map[string]string{"error": "maximum 5 files allowed"})
			ctx.Status(http.StatusBadRequest)
			return nil
		}

		uploaded := make([]map[string]any, 0, len(files))
		for _, fh := range files {
			file, err := fh.Open()
			if err != nil {
				continue
			}

			destPath := filepath.Join(uploadDir, fh.Filename)
			dest, err := os.Create(destPath)
			if err != nil {
				file.Close()
				continue
			}

			written, _ := io.Copy(dest, file)
			file.Close()
			dest.Close()

			uploaded = append(uploaded, map[string]any{
				"filename": fh.Filename,
				"size":     written,
				"type":     fh.Header.Get("Content-Type"),
			})
		}

		ctx.Json(map[string]any{
			"message": fmt.Sprintf("%d files uploaded", len(uploaded)),
			"files":   uploaded,
		})
		return nil
	})

	// ── Upload status ──────────────────────────────────────────────────
	router.GET("/uploads", func(ctx *chain.Context) error {
		entries, err := os.ReadDir(uploadDir)
		if err != nil {
			ctx.Json(map[string]string{"error": "failed to read uploads"})
			ctx.InternalServerError()
			return nil
		}

		files := make([]map[string]any, 0, len(entries))
		for _, entry := range entries {
			info, _ := entry.Info()
			files = append(files, map[string]any{
				"name": entry.Name(),
				"size": info.Size(),
			})
		}

		ctx.Json(map[string]any{
			"count": len(files),
			"files": files,
		})
		return nil
	})

	log.Println("File upload demo listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
