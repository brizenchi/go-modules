package bootstrap

import "github.com/brizenchi/quickstart-template/internal/feature/note"

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
	return []any{
		&note.Note{},
	}
}
