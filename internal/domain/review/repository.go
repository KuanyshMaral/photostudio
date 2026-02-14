package review

import (
	"context"
	"gorm.io/gorm"
	"time"
)

type ReviewRepository struct {
	db *gorm.DB
}

func NewReviewRepository(db *gorm.DB) *ReviewRepository {
	return &ReviewRepository{db: db}
}

type reviewModel struct {
	ID            int64      `gorm:"column:id;primaryKey"`
	StudioID      int64      `gorm:"column:studio_id"`
	UserID        int64      `gorm:"column:user_id"`
	BookingID     *int64     `gorm:"column:booking_id"`
	Rating        int        `gorm:"column:rating"`
	Comment       *string    `gorm:"column:comment"`
	Photos        []string   `gorm:"column:photos;type:text[]"`
	OwnerResponse *string    `gorm:"column:owner_response"`
	RespondedAt   *time.Time `gorm:"column:responded_at"`
	IsVerified    bool       `gorm:"column:is_verified"`
	IsHidden      bool       `gorm:"column:is_hidden"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
}

func (reviewModel) TableName() string { return "reviews" }

func toDomainReview(m reviewModel) Review {
	comment := ""
	if m.Comment != nil {
		comment = *m.Comment
	}
	return Review{
		ID:            m.ID,
		StudioID:      m.StudioID,
		UserID:        m.UserID,
		BookingID:     m.BookingID,
		Rating:        m.Rating,
		Comment:       comment,
		Photos:        m.Photos,
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
		StudioID:      r.StudioID,
		UserID:        r.UserID,
		BookingID:     r.BookingID,
		Rating:        r.Rating,
		Comment:       comment,
		Photos:        r.Photos,
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

func (r *ReviewRepository) GetByStudio(ctx context.Context, studioID int64, limit, offset int) ([]Review, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var rows []reviewModel
	tx := r.db.WithContext(ctx).
		Where("studio_id = ? AND is_hidden = false", studioID).
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

func (r *ReviewRepository) DB() *gorm.DB {
	return r.db
}

func (r *ReviewRepository) Update(ctx context.Context, rv *Review) error {
	m := toReviewModel(rv)
	tx := r.db.WithContext(ctx).
		Table("reviews").
		Where("id = ?", rv.ID).
		Updates(&m)
	return tx.Error
}

func (r *ReviewRepository) ExistsByUserAndStudio(ctx context.Context, userID, studioID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&reviewModel{}).
		Where("user_id = ? AND studio_id = ?", userID, studioID).
		Count(&count).Error
	return count > 0, err
}