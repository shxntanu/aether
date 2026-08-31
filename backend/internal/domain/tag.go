package domain

import (
	"fmt"
	"strings"
)

// TagID uniquely identifies a reusable tag.
type TagID string

// Tag retains display spelling while providing a case-insensitive key.
type Tag struct {
	// ID uniquely identifies the reusable tag.
	ID TagID `json:"id"`
	// DisplayName retains the first accepted spelling.
	DisplayName string `json:"displayName"`
	// NormalizedName is the case-insensitive identity used for matching.
	NormalizedName string `json:"normalizedName"`
}

// NewTag validates and normalizes a reusable tag name.
func NewTag(id TagID, name string) (Tag, error) {
	displayName := strings.Join(strings.Fields(name), " ")
	if displayName == "" {
		return Tag{}, fmt.Errorf("tag name must not be empty")
	}

	return Tag{
		ID:             id,
		DisplayName:    displayName,
		NormalizedName: strings.ToLower(displayName),
	}, nil
}
