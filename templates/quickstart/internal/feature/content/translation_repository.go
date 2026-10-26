package content

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/brizenchi/quickstart-template/internal/audit"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrTranslationInProgress = errors.New("content: translation already in progress")
	ErrTranslationLease      = errors.New("content: translation job lease lost")
)

// TranslationJob persists one queued or running AI translation. ActiveMarker
// holds the article id only while the job is active; its unique index gives
// every article at most one active job across application instances.
type TranslationJob struct {
	ID             string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	ArticleID      string     `gorm:"type:varchar(36);not null;index" json:"article_id"`
	ActorID        string     `gorm:"type:varchar(36);not null" json:"actor_id"`
	SourceLocale   string     `gorm:"type:varchar(8);not null" json:"source_locale"`
	TargetLocale   string     `gorm:"type:varchar(8);not null" json:"target_locale"`
	ArticleVersion int64      `gorm:"not null" json:"article_version"`
	SourceSnapshot string     `gorm:"type:text;not null" json:"-"`
	SourceHash     string     `gorm:"type:varchar(64);not null" json:"-"`
	IdempotencyKey string     `gorm:"type:varchar(128);not null" json:"-"`
	RequestHash    string     `gorm:"type:varchar(64);not null" json:"-"`
	Status         string     `gorm:"type:varchar(16);not null;index" json:"status"`
	Phase          string     `gorm:"type:varchar(16);not null" json:"phase"`
	ErrorReason    string     `gorm:"type:varchar(64)" json:"reason"`
	ResultMeta     string     `gorm:"type:text" json:"-"`
	ActiveMarker   *string    `gorm:"type:varchar(36);uniqueIndex:uniq_content_translation_jobs_active"`
	LeaseToken     string     `gorm:"type:varchar(36)" json:"-"`
	Deadline       *time.Time `json:"-"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (TranslationJob) TableName() string { return "content_translation_jobs" }

// SourceTranslation decodes the temporary source snapshot carried by the job.
func (j TranslationJob) SourceTranslation() Translation {
	var source Translation
	_ = json.Unmarshal([]byte(j.SourceSnapshot), &source)
	return source
}

// TranslationJobRequest carries server-read source content; clients never
// upload arbitrary article text to the job entry point.
type TranslationJobRequest struct {
	ArticleID      string
	ActorID        string
	IdempotencyKey string
	Reason         string
	SourceLocale   string
	TargetLocale   string
	ArticleVersion int64
	Source         Translation
}

type translationRequestFingerprint struct {
	ArticleID      string `json:"article_id"`
	ArticleVersion int64  `json:"article_version"`
	SourceLocale   string `json:"source_locale"`
	TargetLocale   string `json:"target_locale"`
	SourceHash     string `json:"source_hash"`
}

func readTranslationJob(ctx context.Context, db *gorm.DB, id string) (TranslationJob, error) {
	var job TranslationJob
	err := db.WithContext(ctx).Where("id = ?", id).Take(&job).Error
	if err == nil {
		return job, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return TranslationJob{}, ErrNotFound
	}
	return TranslationJob{}, err
}

// TranslationJob loads one job by id.
func (s *Service) TranslationJob(ctx context.Context, id string) (TranslationJob, error) {
	return readTranslationJob(ctx, s.db, id)
}

// ActiveTranslationJobID returns the queued or running job for an article,
// or an empty string. It feeds the admin article detail view only.
func (s *Service) ActiveTranslationJobID(ctx context.Context, articleID string) (string, error) {
	var job TranslationJob
	err := s.db.WithContext(ctx).Where("article_id = ? AND status IN ('queued','running')", articleID).Order("created_at, id").Take(&job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return job.ID, nil
}

// CreateTranslationJob validates the saved article, enforces one active job
// per article and commits the job together with its audit claim. The same
// idempotency key with the same request returns the original job; a different
// request under the same key surfaces audit.ErrConflict.
func (s *Service) CreateTranslationJob(ctx context.Context, in TranslationJobRequest) (TranslationJob, error) {
	if !validTranslationLocales(in.SourceLocale, in.TargetLocale) {
		return TranslationJob{}, ErrInvalid
	}
	fingerprint := translationRequestFingerprint{
		ArticleID:      in.ArticleID,
		ArticleVersion: in.ArticleVersion,
		SourceLocale:   in.SourceLocale,
		TargetLocale:   in.TargetLocale,
		SourceHash:     HashTranslation(in.Source),
	}
	requestHash := audit.Hash(fingerprint)
	operation := audit.Operation{
		ActorID:        in.ActorID,
		IdempotencyKey: in.IdempotencyKey,
		Action:         "content.translation.create",
		TargetID:       in.ArticleID,
		Reason:         in.Reason,
		RequestHash:    requestHash,
		Status:         "pending",
	}
	var job TranslationJob
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		created, err := s.audit.Claim(ctx, tx, &operation)
		if err != nil {
			return err
		}
		if !created {
			var saved struct {
				JobID string `json:"job_id"`
			}
			if err := json.Unmarshal([]byte(operation.Details), &saved); err != nil {
				return err
			}
			replayed, err := readTranslationJob(ctx, tx, saved.JobID)
			if err != nil {
				return err
			}
			job = replayed
			return nil
		}
		article, err := read(ctx, tx, in.ArticleID)
		if err != nil {
			return err
		}
		if article.Version != in.ArticleVersion {
			return ErrVersion
		}
		if article.SourceLocale != in.SourceLocale || HashTranslation(article.Translations[in.SourceLocale]) != fingerprint.SourceHash {
			return ErrInvalid
		}
		var active int64
		if err := tx.Model(&TranslationJob{}).Where("article_id = ? AND status IN ('queued','running')", in.ArticleID).Count(&active).Error; err != nil {
			return err
		}
		if active > 0 {
			return ErrTranslationInProgress
		}
		snapshot, err := json.Marshal(in.Source)
		if err != nil {
			return err
		}
		job = TranslationJob{
			ID:             uuid.NewString(),
			ArticleID:      in.ArticleID,
			ActorID:        in.ActorID,
			SourceLocale:   in.SourceLocale,
			TargetLocale:   in.TargetLocale,
			ArticleVersion: in.ArticleVersion,
			SourceSnapshot: string(snapshot),
			SourceHash:     fingerprint.SourceHash,
			IdempotencyKey: in.IdempotencyKey,
			RequestHash:    requestHash,
			Status:         "queued",
			Phase:          "queued",
			ActiveMarker:   &in.ArticleID,
		}
		if err := tx.Create(&job).Error; err != nil {
			return err
		}
		details, _ := json.Marshal(map[string]any{"job_id": job.ID, "article_id": in.ArticleID, "source_locale": in.SourceLocale, "target_locale": in.TargetLocale, "article_version": in.ArticleVersion})
		return s.audit.Finish(ctx, tx, &operation, "succeeded", string(details))
	})
	if err != nil {
		return TranslationJob{}, err
	}
	return job, nil
}

// ClaimNextTranslationJob conditionally claims the oldest queued job. The
// returned lease token authorizes exactly one worker until the deadline.
func (s *Service) ClaimNextTranslationJob(ctx context.Context, leaseToken string, deadline time.Time) (TranslationJob, bool, error) {
	var claimed TranslationJob
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var ids []string
		if err := tx.Model(&TranslationJob{}).Where("status = 'queued'").Order("created_at, id").Limit(1).Pluck("id", &ids).Error; err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}
		result := tx.Model(&TranslationJob{}).Where("id = ? AND status = 'queued'", ids[0]).Updates(map[string]any{
			"status": "running", "phase": "translating", "lease_token": leaseToken, "deadline": deadline, "updated_at": time.Now().UTC(),
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		job, err := readTranslationJob(ctx, tx, ids[0])
		if err != nil {
			return err
		}
		claimed = job
		return nil
	})
	if err != nil {
		return TranslationJob{}, false, err
	}
	return claimed, claimed.ID != "", nil
}

// UpdateTranslationJobPhase reports worker progress and requires the lease.
func (s *Service) UpdateTranslationJobPhase(ctx context.Context, job TranslationJob, phase string) error {
	result := s.db.WithContext(ctx).Model(&TranslationJob{}).Where("id = ? AND lease_token = ? AND status = 'running'", job.ID, job.LeaseToken).Updates(map[string]any{"phase": phase, "updated_at": time.Now().UTC()})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTranslationLease
	}
	return nil
}

func (s *Service) failTranslationJobInTx(ctx context.Context, tx *gorm.DB, job TranslationJob, reason string) error {
	result := tx.Model(&TranslationJob{}).Where("id = ? AND lease_token = ? AND status = 'running'", job.ID, job.LeaseToken).Updates(map[string]any{
		"status": "failed", "error_reason": reason, "source_snapshot": "", "active_marker": nil, "lease_token": "", "updated_at": time.Now().UTC(),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTranslationLease
	}
	return s.audit.Record(ctx, tx, audit.Operation{
		ActorID:        job.ActorID,
		IdempotencyKey: "translation-fail:" + job.ID,
		Action:         "content.translation.fail",
		TargetID:       job.ArticleID,
		Reason:         reason,
		RequestHash:    job.RequestHash,
	}, map[string]any{"job_id": job.ID, "reason": reason})
}

// FailTranslationJob terminates the running job owned by this lease. Drafts
// and published snapshots stay untouched.
func (s *Service) FailTranslationJob(ctx context.Context, job TranslationJob, reason string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.failTranslationJobInTx(ctx, tx, job, reason)
	})
}

// CompleteTranslationJob atomically replaces only the target draft, records
// translation metadata, bumps the article version and finishes the job. Any
// concurrent article write turns into a committed job failure with the
// candidate discarded.
func (s *Service) CompleteTranslationJob(ctx context.Context, job TranslationJob, candidate Translation, checkStatus string, issues []ReviewIssue, model string) (Article, error) {
	if !validGeneratedTranslation(candidate) || (checkStatus != "passed" && checkStatus != "warning") {
		return Article{}, ErrInvalid
	}
	meta := TranslationMeta{
		SourceLocale: job.SourceLocale,
		SourceHash:   job.SourceHash,
		DraftHash:    HashTranslation(candidate),
		CheckStatus:  checkStatus,
		Issues:       issues,
		Model:        model,
		CompletedAt:  time.Now().UTC().Format(time.RFC3339),
		JobID:        job.ID,
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return Article{}, err
	}
	var article Article
	var conflict error
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		current, err := read(ctx, tx, job.ArticleID)
		if err != nil {
			return err
		}
		if current.Version != job.ArticleVersion || HashTranslation(current.Translations[job.SourceLocale]) != job.SourceHash {
			if err := s.failTranslationJobInTx(ctx, tx, job, ReasonTranslationVersionConflict); err != nil {
				return err
			}
			conflict = ErrVersion
			return nil
		}
		result := tx.Model(&TranslationJob{}).Where("id = ? AND lease_token = ? AND status = 'running'", job.ID, job.LeaseToken).Updates(map[string]any{
			"status": "succeeded", "phase": "done", "source_snapshot": "", "active_marker": nil, "lease_token": "", "result_meta": string(metaJSON), "updated_at": time.Now().UTC(),
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrTranslationLease
		}
		current.Translations[job.TargetLocale] = candidate
		if current.TranslationMeta == nil {
			current.TranslationMeta = map[string]TranslationMeta{}
		}
		current.TranslationMeta[job.TargetLocale] = meta
		current.Version++
		current.UpdatedAt = time.Now().UTC()
		if err := persist(tx, current, job.ArticleVersion); err != nil {
			return err
		}
		article = current
		return s.audit.Record(ctx, tx, audit.Operation{
			ActorID:        job.ActorID,
			IdempotencyKey: "translation-apply:" + job.ID,
			Action:         "content.translation.apply",
			TargetID:       job.ArticleID,
			Reason:         "AI 翻译：" + job.SourceLocale + " → " + job.TargetLocale,
			RequestHash:    job.RequestHash,
		}, map[string]any{"job_id": job.ID, "article_id": job.ArticleID, "target_locale": job.TargetLocale, "check_status": checkStatus, "before": job.ArticleVersion, "after": current.Version})
	})
	if err != nil {
		return Article{}, err
	}
	if conflict != nil {
		return Article{}, conflict
	}
	return article, nil
}

// ReclaimExpiredTranslationJobs fails running jobs whose lease deadline has
// passed. Interrupted paid calls are never replayed automatically.
func (s *Service) ReclaimExpiredTranslationJobs(ctx context.Context, now time.Time) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var expired []TranslationJob
		if err := tx.Where("status = 'running' AND deadline IS NOT NULL AND deadline < ?", now).Find(&expired).Error; err != nil {
			return err
		}
		for _, job := range expired {
			result := tx.Model(&TranslationJob{}).Where("id = ? AND status = 'running'", job.ID).Updates(map[string]any{
				"status": "failed", "error_reason": ReasonTranslationInterrupted, "source_snapshot": "", "active_marker": nil, "lease_token": "", "updated_at": time.Now().UTC(),
			})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				continue
			}
			if err := s.audit.Record(ctx, tx, audit.Operation{
				ActorID:        job.ActorID,
				IdempotencyKey: "translation-reclaim:" + job.ID,
				Action:         "content.translation.reclaim",
				TargetID:       job.ArticleID,
				Reason:         ReasonTranslationInterrupted,
				RequestHash:    job.RequestHash,
			}, map[string]any{"job_id": job.ID, "reason": ReasonTranslationInterrupted}); err != nil {
				return err
			}
		}
		return nil
	})
}
