package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/jpeg" // Register JPEG decoder
	_ "image/png"  // Register PNG decoder
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	"pixstore/internal/config"
	// _ "golang.org/x/image/webp" // If we need to read webp inputs too.
)

type ImageProcessor struct {
	cfg *config.Config
}

func NewImageProcessor(cfg *config.Config) *ImageProcessor {
	return &ImageProcessor{cfg: cfg}
}

// NormalizedName normalizes the product name
func (s *ImageProcessor) NormalizeName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	// Simple removal of special characters, keep alphanumeric and spaces
	// Or use a regex replacement if stricter
	// user req: remove special characters
	var sb strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func (s *ImageProcessor) GenerateHash(file multipart.File) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}
	// Reset file pointer
	if _, err := file.Seek(0, 0); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (s *ImageProcessor) ProcessAndSave(file multipart.File, hash string) (thumbPath, img600Path string, err error) {
	// Decode image
	img, _, err := image.Decode(file)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Paths
	storagePath := s.cfg.StoragePath
	originalDir := filepath.Join(storagePath, "original")
	thumbDir := filepath.Join(storagePath, "thumb")
	img600Dir := filepath.Join(storagePath, "600")

	// Create directories if not exist (ensure they exist at startup is better, but safe here)
	for _, dir := range []string{originalDir, thumbDir, img600Dir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", "", err
		}
	}

	// 1. Save Original (private)
	// We'll save it as .webp or original format? 
	// Req: "Save original image in a PRIVATE folder"
	// Req: "Convert image to WebP" (implies serving layers)
	// Let's save the original format as is for backup, or convert it?
	// Users usually want the RAW original to remain in original format.
	// But for this pipeline, we just need to store it. Storage path depends on content.
	// Let's save the decoded image objects.
	// Actually for "Save original", we should copy the input stream directly to disk to preserve it exactly.
	// But we already decoded it. Let's Seek(0, 0) and Copy.

	if _, err := file.Seek(0, 0); err != nil {
		return "", "", err
	}
	
	originalPath := filepath.Join(originalDir, hash+".original") // unknown ext? or just binary
	dest, err := os.Create(originalPath)
	if err != nil {
		return "", "", err
	}
	defer dest.Close()
	if _, err := io.Copy(dest, file); err != nil {
		return "", "", err
	}

	// 2. Generate 600x600 WebP
	// "Product image: 600x600 WebP"
	// "High-quality resize (no stretching)" -> Fit
	img600 := imaging.Fit(img, 600, 600, imaging.Lanczos)
	img600PathRel := fmt.Sprintf("/600/%s.webp", hash)
	img600OutPath := filepath.Join(img600Dir, hash+".webp")
	
	if err := saveWebP(img600, img600OutPath); err != nil {
		return "", "", err
	}

	// 3. Generate Thumbnail 200x200 WebP
	// "High-quality resize (no stretching)" allows Fit (maintain aspect ratio, one dim < 200).
	// Let's stick to Fit.
	thumbResized := imaging.Fit(img, 200, 200, imaging.Lanczos)
	// Or maybe Fill (crop to center) is better for strict 200x200 grid?
	// "High-quality resize (no stretching)" allows Fit (maintain aspect ratio, one dim < 200).
	// Let's stick to Fit.
	
	thumbPathRel := fmt.Sprintf("/thumb/%s.webp", hash)
	thumbOutPath := filepath.Join(thumbDir, hash+".webp")

	if err := saveWebP(thumbResized, thumbOutPath); err != nil {
		return "", "", err
	}

	return thumbPathRel, img600PathRel, nil
}

func saveWebP(img image.Image, path string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	// Quality 80 seems reasonable for products
	if err := webp.Encode(out, img, &webp.Options{Lossless: false, Quality: 80}); err != nil {
		os.Remove(path) // cleanup
		return fmt.Errorf("failed to encode webp: %w", err)
	}
	return nil
}

func (s *ImageProcessor) ValidateType(fileHeader *multipart.FileHeader) error {
	// Simple validation by extension or content type
	// implementation requirement: "Validate image (jpg, png, webp only)"
	// Better to use http.DetectContentType for security?
	// We'll rely on extension first for speed, then Decode will fail if invalid.
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return fmt.Errorf("unsupported file type: %s", ext)
	}
	return nil
}
