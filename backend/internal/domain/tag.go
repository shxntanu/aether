package domain

import (
	"fmt"
	"strings"
)

type TagID string

type Tag struct {
	ID             TagID
	DisplayName    string
	NormalizedName string
}

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
