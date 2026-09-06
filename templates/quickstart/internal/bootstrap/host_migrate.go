package bootstrap

import (
	"github.com/brizenchi/quickstart-template/internal/feature/credits"
	"github.com/brizenchi/quickstart-template/internal/feature/note"
	"github.com/brizenchi/quickstart-template/internal/feature/operations"
)

// YOURS — edit freely.
//
// hostModels lists your own feature models. They are auto-migrated at boot,
// right after the host User and enabled modules have migrated their tables, so you can
// safely reference users(id) from here.
//
// Keep the list flat and explicit; one line per model:
//
//	func hostModels() []any {
//		return []any{
//			&note.Note{},
//			&invoice.Invoice{},
//		}
//	}
func hostModels() []any {
	models := []any{
		&note.Note{},
	}
	models = append(models, credits.Models()...)
	return append(models, operations.Models()...)
}
