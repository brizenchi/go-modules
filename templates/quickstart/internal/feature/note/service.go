package note

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"
)

// ErrEmptyTitle is returned when a note has no usable title.
var ErrEmptyTitle = errors.New("note: title is required")
var ErrInvalidContent = errors.New("note: title must be at most 200 characters and content at most 50000 characters")

const defaultListLimit = 50

// service holds business rules. No gin, no SQL.
type service struct {
	repo *repository
}

func newService(repo *repository) *service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, userID, title, body string) (*Note, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrEmptyTitle
	}
	if !utf8.ValidString(title) || !utf8.ValidString(body) || utf8.RuneCountInString(title) > 200 || utf8.RuneCountInString(body) > 50000 {
		return nil, ErrInvalidContent
	}
	n := &Note{UserID: userID, Title: title, Body: body}
	if err := s.repo.Create(ctx, n); err != nil {
		return nil, err
	}
	return n, nil
}

func (s *service) ListMine(ctx context.Context, userID string) ([]Note, error) {
	return s.repo.ListByUser(ctx, userID, defaultListLimit)
}

func (s *service) CountAll(ctx context.Context) (int64, error) {
	return s.repo.Count(ctx)
}
