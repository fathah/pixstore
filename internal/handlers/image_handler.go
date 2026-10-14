package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"pixstore/internal/config"
	"pixstore/internal/models"
	"pixstore/internal/repository"
	"pixstore/internal/services"
)

type ImageHandler struct {
	repo      *repository.ImageRepository
	processor *services.ImageProcessor
	cfg       *config.Config
}

func NewImageHandler(repo *repository.ImageRepository, processor *services.ImageProcessor, cfg *config.Config) *ImageHandler {
	return &ImageHandler{
		repo:      repo,
		processor: processor,
		cfg:       cfg,
	}
}

type UploadResponse struct {
	ID           string `json:"id"`
	ThumbnailURL string `json:"thumbnail_url"`
	ImageURL     string `json:"image_url"`
}

func (h *ImageHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Auth Check
	// Check header X-Admin-Secret
	secret := r.Header.Get("X-Admin-Secret")
	if secret == "" {
		// Fallback to query param? Or just stick to header.
		// "bound with a secret code" - simplest is header.
		secret = r.URL.Query().Get("code")
	}
	
	if secret != h.cfg.AdminSecret {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 10MB limit for now
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "File too large or invalid form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Image file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if err := h.processor.ValidateType(header); err != nil {
		http.Error(w, fmt.Sprintf("Invalid image: %v", err), http.StatusBadRequest)
		return
	}

	// Extract metadata
	productName := strings.TrimSpace(r.FormValue("product_name"))
	category := strings.TrimSpace(r.FormValue("category"))
	tagsStr := r.FormValue("tags")
	var tags []string
	if tagsStr != "" {
		parts := strings.Split(tagsStr, ",")
		for _, p := range parts {
			tag := strings.TrimSpace(p)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	if productName == "" {
		http.Error(w, "product_name is required", http.StatusBadRequest)
		return
	}

	// Generate Hash
	hash, err := h.processor.GenerateHash(file)
	if err != nil {
		http.Error(w, "Failed to process image", http.StatusInternalServerError)
		return
	}

	// Check duplication
	existingID, err := h.repo.ExistsByHash(r.Context(), hash)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if existingID != "" {
		// Return existing
		existingFunc, err := h.repo.GetByHash(r.Context(), hash)
		if err == nil {
			resp := UploadResponse{
				ID:           existingFunc.ID,
				ThumbnailURL: existingFunc.ThumbPath,
				ImageURL:     existingFunc.Image600Path,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
	}

	// Process and Save
	thumbPath, imgPath, err := h.processor.ProcessAndSave(file, hash)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to save image: %v", err), http.StatusInternalServerError)
		return
	}

	normalizedName := h.processor.NormalizeName(productName)

	img := &models.Image{
		Hash:           hash,
		OriginalName:   header.Filename, // Or productName? Requirement lists original_name in DB.
		NormalizedName: normalizedName,
		Category:       category,
		Tags:           tags,
		ThumbPath:      thumbPath,
		Image600Path:   imgPath,
	}

	if err := h.repo.Create(r.Context(), img); err != nil {
		// Cleanup files?
		http.Error(w, "Failed to save metadata", http.StatusInternalServerError)
		return
	}

	resp := UploadResponse{
		ID:           img.ID,
		ThumbnailURL: img.ThumbPath,
		ImageURL:     img.Image600Path,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *ImageHandler) Search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Auth Check
	apiKey := r.Header.Get("x-api-key")
	if apiKey != h.cfg.ApiKey {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	query := r.URL.Query().Get("q")
	limitStr := r.URL.Query().Get("limit")

	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	results, err := h.repo.Search(r.Context(), query, limit)
	if err != nil {
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// If empty, return empty array []
	if results == nil {
		w.Write([]byte("[]"))
		return
	}
	json.NewEncoder(w).Encode(results)
}
