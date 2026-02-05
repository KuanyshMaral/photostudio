package repository

import (
	"context"
	"photostudio/internal/modules/auth"
	"strings"
	"time"

	"photostudio/internal/domain"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

type userModel struct {
	ID            int64     `gorm:"column:id;primaryKey"`
	Email         string    `gorm:"column:email"`
	PasswordHash  string    `gorm:"column:password_hash"`
	Role          string    `gorm:"column:role"`
	Name          string    `gorm:"column:name"`
	Phone         *string   `gorm:"column:phone"`
	AvatarURL     *string   `gorm:"column:avatar_url"`
	EmailVerified bool      `gorm:"column:email_verified"`
	StudioStatus  *string   `gorm:"column:studio_status"`
	MworkUserID   *string   `gorm:"column:mwork_user_id"`
	MworkRole     *string   `gorm:"column:mwork_role"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

func (userModel) TableName() string { return "users" }

func toDomainUser(m userModel) *domain.User {
	var phone, avatar, status, mworkUserID, mworkRole string
	if m.Phone != nil {
		phone = *m.Phone
	}
	if m.AvatarURL != nil {
		avatar = *m.AvatarURL
	}
	if m.StudioStatus != nil {
		status = *m.StudioStatus
	}
	if m.MworkUserID != nil {
		mworkUserID = *m.MworkUserID
	}
	if m.MworkRole != nil {
		mworkRole = *m.MworkRole
	}

	return &domain.User{
		ID:            m.ID,
		Email:         m.Email,
		PasswordHash:  m.PasswordHash,
		Role:          domain.UserRole(m.Role),
		Name:          m.Name,
		Phone:         phone,
		AvatarURL:     avatar,
		EmailVerified: m.EmailVerified,
		StudioStatus:  domain.StudioStatus(status),
		MworkUserID:   mworkUserID,
		MworkRole:     mworkRole,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

func toUserModel(u *domain.User) userModel {
	email := strings.TrimSpace(strings.ToLower(u.Email))

	var phone, avatar, status, mworkUserID, mworkRole *string
	if u.Phone != "" {
		v := u.Phone
		phone = &v
	}
	if u.AvatarURL != "" {
		v := u.AvatarURL
		avatar = &v
	}
	if u.StudioStatus != "" {
		v := string(u.StudioStatus)
		status = &v
	}
	if u.MworkUserID != "" {
		v := u.MworkUserID
		mworkUserID = &v
	}
	if u.MworkRole != "" {
		v := u.MworkRole
		mworkRole = &v
	}

	return userModel{
		ID:            u.ID,
		Email:         email,
		PasswordHash:  u.PasswordHash,
		Role:          string(u.Role),
		Name:          u.Name,
		Phone:         phone,
		AvatarURL:     avatar,
		EmailVerified: u.EmailVerified,
		StudioStatus:  status,
		MworkUserID:   mworkUserID,
		MworkRole:     mworkRole,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}

func (r *UserRepository) DB() *gorm.DB {
	return r.db
}

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	m := toUserModel(u)
	tx := r.db.WithContext(ctx).Create(&m)
	if tx.Error != nil {
		return tx.Error
	}
	*u = *toDomainUser(m)
	return nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var m userModel
	tx := r.db.WithContext(ctx).
		Where("LOWER(email) = ?", strings.ToLower(strings.TrimSpace(email))).
		First(&m)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return toDomainUser(m), nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	var m userModel
	tx := r.db.WithContext(ctx).First(&m, id)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return toDomainUser(m), nil
}

func (r *UserRepository) GetByMworkUserID(ctx context.Context, mworkUserID string) (*domain.User, error) {
	var m userModel
	tx := r.db.WithContext(ctx).
		Where("mwork_user_id = ?", mworkUserID).
		First(&m)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return toDomainUser(m), nil
}

// ExistsByEmail checks if email already exists (case-insensitive)
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&userModel{}).
		Where("LOWER(email) = ?", strings.ToLower(strings.TrimSpace(email))).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *UserRepository) Update(ctx context.Context, u *domain.User) error {
	m := toUserModel(u)
	// We use Save because it updates all fields (including timestamps)
	// and works whether ID is set or not
	tx := r.db.WithContext(ctx).Save(&m)
	if tx.Error != nil {
		return tx.Error
	}
	// Optional: refresh domain object with updated timestamps
	*u = *toDomainUser(m)
	return nil
}

var _ auth.UserRepositoryInterface = (*UserRepository)(nil)
