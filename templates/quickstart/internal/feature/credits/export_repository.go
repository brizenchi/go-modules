package credits

import (
	"context"
	"fmt"

	"github.com/brizenchi/quickstart-template/internal/credit"
	"github.com/brizenchi/quickstart-template/internal/feature/note"
	"gorm.io/gorm"
)

type exportRepository struct{ db *gorm.DB }

func (r exportRepository) find(ctx context.Context, userID, key string, noteID uint64) (*NoteExport, error) {
	var saved NoteExport
	err := r.db.WithContext(ctx).Where("user_id = ? AND idempotency_key = ?", userID, key).Take(&saved).Error
	if err == nil && saved.NoteID != noteID {
		return nil, credit.ErrConflict
	}
	return &saved, err
}
func (r exportRepository) checkOwner(ctx context.Context, userID string, noteID uint64) error {
	var owned note.Note
	return r.db.WithContext(ctx).Where("id = ? AND user_id = ?", noteID, userID).Take(&owned).Error
}

// saveInTransaction rechecks ownership and saves the result using the credit
// service's transaction, so a failed write also rolls back the charge.
func (r exportRepository) saveInTransaction(ctx context.Context, tx *gorm.DB, userID, key string, noteID uint64, entry *credit.Transaction) error {
	var current note.Note
	if err := tx.WithContext(ctx).Where("id = ? AND user_id = ?", noteID, userID).Take(&current).Error; err != nil {
		return err
	}
	saved := NoteExport{UserID: userID, IdempotencyKey: key, NoteID: noteID, TransactionID: entry.ID, Filename: fmt.Sprintf("note-%d.md", noteID), Content: "# " + current.Title + "\n\n" + current.Body + "\n"}
	return tx.WithContext(ctx).Create(&saved).Error
}
