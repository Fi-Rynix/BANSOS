package helper

import (
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	"bansos-backend/app/model"
)

const (
	DefaultPageLimit = 20
	MaxPageLimit     = 100
)

func EncodeCursor(c model.Cursor) string {
	raw := c.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + strconv.Itoa(c.ID)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func DecodeCursor(s string) (*model.Cursor, error) {
	if s == "" {
		return nil, nil
	}

	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, BadRequest("cursor tidak valid")
	}

	createdPart, idPart, ok := strings.Cut(string(raw), "|")
	if !ok {
		return nil, BadRequest("cursor tidak valid")
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdPart)
	if err != nil {
		return nil, BadRequest("cursor tidak valid")
	}

	id, err := strconv.Atoi(idPart)
	if err != nil || id < 1 {
		return nil, BadRequest("cursor tidak valid")
	}

	return &model.Cursor{CreatedAt: createdAt, ID: id}, nil
}

func NormalizeLimit(limit int) int {
	if limit < 1 {
		return DefaultPageLimit
	}
	if limit > MaxPageLimit {
		return MaxPageLimit
	}
	return limit
}

func Paginate[T any](rows []T, limit int, cursorOf func(T) model.Cursor) ([]T, model.CursorMeta) {
	meta := model.CursorMeta{Limit: limit}
	if len(rows) <= limit {
		return rows, meta
	}

	page := rows[:limit]
	meta.HasMore = true
	meta.NextCursor = EncodeCursor(cursorOf(page[len(page)-1]))
	return page, meta
}
