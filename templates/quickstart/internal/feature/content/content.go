// Package content owns localized editorial drafts and published snapshots.
package content

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/brizenchi/quickstart-template/internal/audit"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNotFound = errors.New("content: not found")
	ErrInvalid  = errors.New("content: invalid content")
	ErrVersion  = errors.New("content: revision conflict")
)

type Translation struct {
	Title          string `json:"title"`
	Summary        string `json:"summary"`
	Body           string `json:"body"`
	SEOTitle       string `json:"seo_title"`
	SEODescription string `json:"seo_description"`
}
type Publication struct {
	Translation
	PublishedAt time.Time `json:"published_at"`
}
type Article struct {
	ID           string                 `json:"id"`
	Path         string                 `json:"path"`
	Kind         string                 `json:"kind"`
	Position     int                    `json:"position"`
	Version      int64                  `json:"version"`
	Translations map[string]Translation `json:"translations"`
	Published    map[string]Publication `json:"published"`
	UpdatedAt    time.Time              `json:"updated_at"`
}
type Record struct {
	ID        string `gorm:"type:varchar(36);primaryKey"`
	Path      string `gorm:"type:varchar(200);not null;uniqueIndex"`
	Kind      string `gorm:"type:varchar(20);not null;index"`
	Position  int
	Version   int64  `gorm:"not null"`
	DraftJSON string `gorm:"type:text;not null"`
	LiveJSON  string `gorm:"type:text;not null"`
	UpdatedAt time.Time
}

func (Record) TableName() string { return "content_articles" }
func Models() []any              { return []any{&Record{}} }

//go:embed seed.json
var seedJSON []byte

// Seed returns importable drafts. It never contributes records to public reads.
func Seed() []Article {
	var rows []struct {
		Path         string                 `json:"path"`
		Kind         string                 `json:"kind"`
		Translations map[string]Translation `json:"translations"`
	}
	_ = json.Unmarshal(seedJSON, &rows)
	result := make([]Article, 0, len(rows))
	for i, row := range rows {
		sum := sha256.Sum256([]byte(row.Path))
		result = append(result, Article{ID: hex.EncodeToString(sum[:16]), Path: row.Path, Kind: row.Kind, Position: i, Translations: row.Translations, Published: map[string]Publication{}})
	}
	return result
}

type Service struct {
	db    *gorm.DB
	audit *audit.Service
}

func New(db *gorm.DB, recorder *audit.Service) *Service {
	if recorder == nil {
		recorder = audit.New()
	}
	return &Service{db: db, audit: recorder}
}
func decode(row Record) (Article, error) {
	result := Article{ID: row.ID, Path: row.Path, Kind: row.Kind, Position: row.Position, Version: row.Version, UpdatedAt: row.UpdatedAt, Translations: map[string]Translation{}, Published: map[string]Publication{}}
	if err := json.Unmarshal([]byte(row.DraftJSON), &result.Translations); err != nil {
		return Article{}, err
	}
	if err := json.Unmarshal([]byte(row.LiveJSON), &result.Published); err != nil {
		return Article{}, err
	}
	return result, nil
}
func (s *Service) List(ctx context.Context) ([]Article, error) {
	byID := map[string]Article{}
	cursor := ""
	for {
		var rows []Record
		if err := s.db.WithContext(ctx).Where("id > ?", cursor).Order("id ASC").Limit(100).Find(&rows).Error; err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			break
		}
		for _, row := range rows {
			item, err := decode(row)
			if err != nil {
				return nil, err
			}
			byID[item.ID] = item
			cursor = row.ID
		}
	}
	result := make([]Article, 0, len(byID))
	for _, row := range byID {
		result = append(result, row)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Position == result[j].Position {
			return result[i].Path < result[j].Path
		}
		return result[i].Position < result[j].Position
	})
	return result, nil
}
func (s *Service) Read(ctx context.Context, id string) (Article, error) { return read(ctx, s.db, id) }
func read(ctx context.Context, db *gorm.DB, id string) (Article, error) {
	var row Record
	err := db.WithContext(ctx).Where("id = ?", id).Take(&row).Error
	if err == nil {
		return decode(row)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return Article{}, err
	}
	return Article{}, ErrNotFound
}
func (s *Service) Resolve(ctx context.Context, path, locale string) (*Article, error) {
	if !ValidPath(path) || (locale != "en" && locale != "zh") {
		return nil, ErrInvalid
	}
	var record Record
	err := s.db.WithContext(ctx).Where("path = ?", path).Take(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	row, err := decode(record)
	if err != nil {
		return nil, err
	}
	if _, ok := row.Published[locale]; !ok {
		return nil, nil
	}
	return &row, nil
}

// pagePaths is deliberately bounded: a CMS editor cannot shadow auth, billing,
// API, legal or administrator routes.
var pagePaths = []string{"/", "/features", "/use-cases", "/compare", "/resources", "/docs", "/blog", "/about", "/updates"}

func ValidPath(path string) bool {
	if validPath.MatchString(path) {
		return true
	}
	for _, allowed := range pagePaths {
		if path == allowed {
			return true
		}
	}
	return false
}
func validKind(path, kind string) bool {
	if kind == "page" {
		return ValidPath(path) && !validPath.MatchString(path)
	}
	return validPath.MatchString(path) && strings.HasPrefix(path, "/"+kind+"/")
}

var validPath = regexp.MustCompile(`^/(blog|docs|features|use-cases|compare)/[a-z0-9]+(?:-[a-z0-9]+)*$`)

// ValidArticlePath reports whether path is a publishable article such as /docs/setup.
func ValidArticlePath(path string) bool { return len(path) <= 200 && validPath.MatchString(path) }

type Save struct {
	Version      int64                  `json:"version"`
	Path         string                 `json:"path"`
	Kind         string                 `json:"kind"`
	Position     int                    `json:"position"`
	Translations map[string]Translation `json:"translations"`
	Reason       string                 `json:"reason"`
}

func Valid(in Save) bool {
	if !validKind(in.Path, in.Kind) || len(in.Path) > 200 || in.Position < 0 || in.Position > 100000 || len(in.Translations) > 2 {
		return false
	}
	for locale, t := range in.Translations {
		if locale != "en" && locale != "zh" {
			return false
		}
		if !utf8.ValidString(t.Body) || len(t.Body) > 150000 || utf8.RuneCountInString(t.Title) > 160 || utf8.RuneCountInString(t.Summary) > 500 || utf8.RuneCountInString(t.SEOTitle) > 160 || utf8.RuneCountInString(t.SEODescription) > 500 {
			return false
		}
	}
	return true
}
func (s *Service) Save(ctx context.Context, id, actor, key string, in Save) (Article, error) {
	if !Valid(in) {
		return Article{}, ErrInvalid
	}
	if id == "" {
		id = uuid.NewString()
	}
	operation := audit.Operation{ActorID: actor, IdempotencyKey: key, Action: "content.save", TargetID: id, Reason: in.Reason, RequestHash: audit.Hash(in), Status: "pending"}
	// Create retries use the same actor/key even though the client did not know the generated ID.
	if in.Version == 0 {
		operation.TargetID = in.Path
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		created, err := s.audit.Claim(ctx, tx, &operation)
		if err != nil {
			return err
		}
		if !created {
			var saved struct {
				ID string `json:"article_id"`
			}
			if err := json.Unmarshal([]byte(operation.Details), &saved); err != nil {
				return err
			}
			id = saved.ID
			return nil
		}
		previous, err := read(ctx, tx, id)
		if errors.Is(err, ErrNotFound) {
			previous = Article{ID: id, Path: in.Path, Kind: in.Kind, Published: map[string]Publication{}}
		} else if err != nil {
			return err
		}
		if previous.Version != in.Version {
			return ErrVersion
		}
		if previous.Path != in.Path || previous.Kind != in.Kind {
			return ErrInvalid
		}
		var duplicate int64
		if err := tx.Model(&Record{}).Where("path = ? AND id <> ?", in.Path, id).Count(&duplicate).Error; err != nil {
			return err
		}
		if duplicate > 0 {
			return ErrVersion
		}
		next := Article{ID: id, Path: in.Path, Kind: in.Kind, Position: in.Position, Version: in.Version + 1, Translations: in.Translations, Published: previous.Published, UpdatedAt: time.Now().UTC()}
		if err := persist(tx, next, in.Version); err != nil {
			return err
		}
		details, _ := json.Marshal(map[string]any{"article_id": id, "before": in.Version, "after": next.Version, "path": in.Path, "locales": translationLocales(in.Translations)})
		return s.audit.Finish(ctx, tx, &operation, "succeeded", string(details))
	})
	if err != nil {
		return Article{}, err
	}
	return s.Read(ctx, id)
}
func translationLocales(t map[string]Translation) []string {
	result := []string{}
	for locale := range t {
		result = append(result, locale)
	}
	sort.Strings(result)
	return result
}
func persist(tx *gorm.DB, a Article, expected int64) error {
	draft, _ := json.Marshal(a.Translations)
	live, _ := json.Marshal(a.Published)
	row := Record{ID: a.ID, Path: a.Path, Kind: a.Kind, Position: a.Position, Version: a.Version, DraftJSON: string(draft), LiveJSON: string(live), UpdatedAt: a.UpdatedAt}
	if expected == 0 {
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrVersion
		}
		return nil
	}
	result := tx.Model(&Record{}).Where("id = ? AND version = ?", a.ID, expected).Select("position", "version", "draft_json", "live_json", "updated_at").Updates(&row)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrVersion
	}
	return nil
}

type Publish struct {
	Version int64  `json:"version"`
	Locale  string `json:"locale"`
	Action  string `json:"action"`
	Reason  string `json:"reason"`
}

func (s *Service) Publish(ctx context.Context, id, actor, key string, in Publish) (Article, error) {
	if (in.Locale != "en" && in.Locale != "zh") || (in.Action != "publish" && in.Action != "unpublish") {
		return Article{}, ErrInvalid
	}
	operation := audit.Operation{ActorID: actor, IdempotencyKey: key, Action: "content." + in.Action, TargetID: id, Reason: in.Reason, RequestHash: audit.Hash(in), Status: "pending"}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		created, err := s.audit.Claim(ctx, tx, &operation)
		if err != nil {
			return err
		}
		if !created {
			return nil
		}
		row, err := read(ctx, tx, id)
		if err != nil {
			return err
		}
		if row.Version != in.Version {
			return ErrVersion
		}
		if in.Action == "publish" {
			t, ok := row.Translations[in.Locale]
			if !ok || strings.TrimSpace(t.Title) == "" || strings.TrimSpace(t.Summary) == "" || strings.TrimSpace(t.Body) == "" {
				return ErrInvalid
			}
			if row.Published == nil {
				row.Published = map[string]Publication{}
			}
			row.Published[in.Locale] = Publication{Translation: t, PublishedAt: time.Now().UTC()}
		} else {
			delete(row.Published, in.Locale)
		}
		row.Version++
		row.UpdatedAt = time.Now().UTC()
		if err := persist(tx, row, in.Version); err != nil {
			return err
		}
		details, _ := json.Marshal(map[string]any{"article_id": id, "path": row.Path, "locale": in.Locale, "before": in.Version, "after": row.Version, "action": in.Action})
		return s.audit.Finish(ctx, tx, &operation, "succeeded", string(details))
	})
	if err != nil {
		return Article{}, err
	}
	return s.Read(ctx, id)
}
