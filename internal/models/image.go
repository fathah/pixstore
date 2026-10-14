package models

import (
	"time"
)

type Image struct {
	ID             string    `json:"id"`
	Hash           string    `json:"-"`
	OriginalName   string    `json:"-"`
	NormalizedName string    `json:"-"`
	Category       string    `json:"category,omitempty"`
	Tags           []string  `json:"tags,omitempty"`
	ThumbPath      string    `json:"thumbnail_url"` // This will be the URL path
	Image600Path   string    `json:"image_url"`    // This will be the URL path
	CreatedAt      time.Time `json:"created_at"`
}

type SearchResult struct {
	ID        string `json:"id"`
	ThumbURL  string `json:"thumbnail_url"`
	ImageURL  string `json:"image_url"`
}
