package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"pixstore/internal/models"
)

type ImageRepository struct {
	db *sql.DB
}

func NewImageRepository(db *sql.DB) *ImageRepository {
	return &ImageRepository{db: db}
}

func (r *ImageRepository) Create(ctx context.Context, img *models.Image) error {
	query := `
		INSERT INTO images (
			hash, original_name, normalized_name, category, tags, thumb_path, image_600_path
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`

	err := r.db.QueryRowContext(ctx, query,
		img.Hash,
		img.OriginalName,
		img.NormalizedName,
		img.Category,
		pq.Array(img.Tags),
		img.ThumbPath,
		img.Image600Path,
	).Scan(&img.ID, &img.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to insert image: %w", err)
	}
	return nil
}

func (r *ImageRepository) ExistsByHash(ctx context.Context, hash string) (string, error) {
	var id string
	query := `SELECT id FROM images WHERE hash = $1`
	err := r.db.QueryRowContext(ctx, query, hash).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *ImageRepository) GetByHash(ctx context.Context, hash string) (*models.Image, error) {
	img := &models.Image{}
	query := `
		SELECT id, hash, original_name, normalized_name, category, tags, thumb_path, image_600_path, created_at
		FROM images WHERE hash = $1
	`
	var tags []string
	err := r.db.QueryRowContext(ctx, query, hash).Scan(
		&img.ID, &img.Hash, &img.OriginalName, &img.NormalizedName, &img.Category, pq.Array(&tags),
		&img.ThumbPath, &img.Image600Path, &img.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	img.Tags = tags
	return img, nil
}

func (r *ImageRepository) Search(ctx context.Context, query string, limit int) ([]models.SearchResult, error) {
	// Normalize query for unaccent search
	searchQuery := strings.TrimSpace(query)
	
	// Complex query using pg_trgm similarity on normalized_name and array overlap on tags
	// We'll give higher precedence to name matches
	sqlQuery := `
		SELECT id, thumb_path, image_600_path,
		       similarity(normalized_name, unaccent($1)) as score
		FROM images
		WHERE 
			normalized_name % unaccent($1) OR
			unaccent($1) = ANY(tags)
		ORDER BY score DESC, created_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, sqlQuery, searchQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("search query failed: %w", err)
	}
	defer rows.Close()

	var results []models.SearchResult
	for rows.Next() {
		var img models.SearchResult
		var score float64
		if err := rows.Scan(&img.ID, &img.ThumbURL, &img.ImageURL, &score); err != nil {
			return nil, err
		}
		// Convert stored paths to public URLs if needed here, 
		// but we stored them as relative paths that match the requirement
		results = append(results, img)
	}

	return results, nil
}
