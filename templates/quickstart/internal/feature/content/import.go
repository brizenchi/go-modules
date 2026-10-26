package content

import (
	"context"
	"fmt"

	"github.com/brizenchi/quickstart-template/internal/audit"
	"gorm.io/gorm"
)

type ImportItem struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Title  string `json:"title"`
	Status string `json:"status"`
}
type ImportResult struct {
	Applied bool         `json:"applied"`
	Items   []ImportItem `json:"items"`
	Created int          `json:"created"`
	Skipped int          `json:"skipped"`
}

// ImportStarter explicitly materializes the reviewed starter pack. Reads never
// invoke this method. Existing paths (including unpublished drafts) are skipped,
// so an upgrade or repeated click cannot replace an editor's work.
func (s *Service) ImportStarter(ctx context.Context, actor, key, reason string, apply bool) (ImportResult, error) {
	result := ImportResult{Applied: apply, Items: []ImportItem{}}
	run := func(db *gorm.DB) error {
		service := New(db, s.audit)
		rows, err := service.List(ctx)
		if err != nil {
			return err
		}
		existing := map[string]bool{}
		for _, row := range rows {
			existing[row.Path] = true
		}
		for _, draft := range Seed() {
			item := ImportItem{Path: draft.Path, Kind: draft.Kind, Title: draft.Translations["en"].Title, Status: "create"}
			if existing[draft.Path] {
				item.Status = "skip"
				result.Skipped++
			} else {
				if apply {
					intent := audit.Hash(struct{ Key, Path string }{key, draft.Path})
					saved, err := service.Save(ctx, draft.ID, actor, intent+"-save", Save{Path: draft.Path, Kind: draft.Kind, Position: &draft.Position, Translations: draft.Translations, Reason: reason})
					if err != nil {
						return fmt.Errorf("import content draft: %w", err)
					}
					for _, locale := range Locales {
						if _, ok := draft.Translations[locale]; !ok {
							continue
						}
						saved, err = service.Publish(ctx, saved.ID, actor, intent+"-"+locale, Publish{Version: saved.Version, Locale: locale, Action: "publish", Reason: reason})
						if err != nil {
							return fmt.Errorf("import content publication: %w", err)
						}
					}
				}
				result.Created++
			}
			result.Items = append(result.Items, item)
		}
		return nil
	}
	var err error
	if apply {
		err = s.db.WithContext(ctx).Transaction(run)
	} else {
		err = run(s.db.WithContext(ctx))
	}
	if err != nil {
		return ImportResult{}, err
	}
	return result, nil
}
