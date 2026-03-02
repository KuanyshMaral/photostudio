package review

import (
	"context"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type ReviewRepository struct {
	db *gorm.DB
}

func NewReviewRepository(db *gorm.DB) *ReviewRepository {
	return &ReviewRepository{db: db}
}

type reviewModel struct {
	ID            int64          `gorm:"column:id;primaryKey"`
	AuthorID      int64          `gorm:"column:author_id"`
	TargetType    TargetType     `gorm:"column:target_type"`
	TargetID      int64          `gorm:"column:target_id"`
	ContextType   *string        `gorm:"column:context_type"`
	ContextID     *int64         `gorm:"column:context_id"`
	Rating        int            `gorm:"column:rating"`
	Comment       *string        `gorm:"column:comment"`
	Photos        pq.StringArray `gorm:"column:photos_deprecated;type:text[]"`
	Criteria      string         `gorm:"column:criteria;type:jsonb;default:'{}'"`
	OwnerResponse *string        `gorm:"column:owner_response"`
	RespondedAt   *time.Time     `gorm:"column:responded_at"`
	IsVerified    bool           `gorm:"column:is_verified"`
	IsHidden      bool           `gorm:"column:is_hidden"`
	CreatedAt     time.Time      `gorm:"column:created_at"`
	UpdatedAt     time.Time      `gorm:"column:updated_at"`
}

func (reviewModel) TableName() string { return "reviews" }

func toDomainReview(m reviewModel) Review {
	comment := ""
	if m.Comment != nil {
		comment = *m.Comment
	}
	return Review{
		ID:            m.ID,
		AuthorID:      m.AuthorID,
		TargetType:    m.TargetType,
		TargetID:      m.TargetID,
		ContextType:   m.ContextType,
		ContextID:     m.ContextID,
		Rating:        m.Rating,
		Comment:       comment,
		Photos:        []string(m.Photos),
		Criteria:      []byte(m.Criteria),
		OwnerResponse: m.OwnerResponse,
		RespondedAt:   m.RespondedAt,
		IsVerified:    m.IsVerified,
		IsHidden:      m.IsHidden,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

func toReviewModel(r *Review) reviewModel {
	var comment *string
	if r.Comment != "" {
		v := r.Comment
		comment = &v
	}
	return reviewModel{
		ID:            r.ID,
		AuthorID:      r.AuthorID,
		TargetType:    r.TargetType,
		TargetID:      r.TargetID,
		ContextType:   r.ContextType,
		ContextID:     r.ContextID,
		Rating:        r.Rating,
		Comment:       comment,
		Photos:        pq.StringArray(r.Photos),
		Criteria:      string(r.Criteria),
		OwnerResponse: r.OwnerResponse,
		RespondedAt:   r.RespondedAt,
		IsVerified:    r.IsVerified,
		IsHidden:      r.IsHidden,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}

func (r *ReviewRepository) Create(ctx context.Context, rv *Review) error {
	m := toReviewModel(rv)
	if len(m.Criteria) == 0 {
		m.Criteria = "{}"
	}
	tx := r.db.WithContext(ctx).Create(&m)
	if tx.Error != nil {
		return tx.Error
	}
	*rv = toDomainReview(m)
	return nil
}

func (r *ReviewRepository) GetByID(ctx context.Context, id int64) (*Review, error) {
	var m reviewModel
	tx := r.db.WithContext(ctx).First(&m, id)
	if tx.Error != nil {
		return nil, tx.Error
	}
	d := toDomainReview(m)
	return &d, nil
}

func (r *ReviewRepository) GetByTarget(ctx context.Context, targetType TargetType, targetID int64, limit, offset int) ([]Review, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var rows []reviewModel
	tx := r.db.WithContext(ctx).
		Where("target_type = ? AND target_id = ? AND is_hidden = false", targetType, targetID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&rows)
	if tx.Error != nil {
		return nil, tx.Error
	}

	out := make([]Review, 0, len(rows))
	for _, m := range rows {
		out = append(out, toDomainReview(m))
	}
	return out, nil
}

func (r *ReviewRepository) SetOwnerResponse(ctx context.Context, reviewID int64, response string) (*Review, error) {
	tx := r.db.WithContext(ctx).
		Table("reviews").
		Where("id = ?", reviewID).
		Updates(map[string]any{
			"owner_response": response,
			"responded_at":   time.Now().UTC(),
			"updated_at":     time.Now().UTC(),
		})
	if tx.Error != nil {
		return nil, tx.Error
	}
	if tx.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return r.GetByID(ctx, reviewID)
}

func (r *ReviewRepository) Update(ctx context.Context, rv *Review) error {
	m := toReviewModel(rv)
	tx := r.db.WithContext(ctx).
		Table("reviews").
		Where("id = ?", rv.ID).
		Updates(&m)
	return tx.Error
}

func (r *ReviewRepository) HasReviewed(ctx context.Context, authorID int64, targetType TargetType, targetID int64, contextType *string, contextID *int64) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).
		Model(&reviewModel{}).
		Where("author_id = ? AND target_type = ? AND target_id = ?", authorID, targetType, targetID)

	if contextType != nil && contextID != nil {
		query = query.Where("context_type = ? AND context_id = ?", contextType, contextID)
	} else {
		query = query.Where("context_type IS NULL AND context_id IS NULL")
	}

	err := query.Count(&count).Error
	return count > 0, err
}

func (r *ReviewRepository) DB() *gorm.DB {
	return r.db
}
