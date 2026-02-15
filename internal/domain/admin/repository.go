package admin

import (
	"context"

	"gorm.io/gorm"
)

type AdminRepository interface {
	Create(ctx context.Context, admin *AdminUser) error
	GetByID(ctx context.Context, id string) (*AdminUser, error)
	GetByEmail(ctx context.Context, email string) (*AdminUser, error)
	Update(ctx context.Context, admin *AdminUser) error
	List(ctx context.Context, limit, offset int) ([]AdminUser, int64, error)
}

type adminRepository struct {
	db *gorm.DB
}

func NewAdminRepository(db *gorm.DB) AdminRepository {
	return &adminRepository{db: db}
}

func (r *adminRepository) Create(ctx context.Context, admin *AdminUser) error {
	return r.db.WithContext(ctx).Create(admin).Error
}

func (r *adminRepository) GetByID(ctx context.Context, id string) (*AdminUser, error) {
	var admin AdminUser
	if err := r.db.WithContext(ctx).First(&admin, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &admin, nil
}

func (r *adminRepository) GetByEmail(ctx context.Context, email string) (*AdminUser, error) {
	var admin AdminUser
	if err := r.db.WithContext(ctx).First(&admin, "email = ?", email).Error; err != nil {
		return nil, err
	}
	return &admin, nil
}

func (r *adminRepository) Update(ctx context.Context, admin *AdminUser) error {
	return r.db.WithContext(ctx).Save(admin).Error
}

func (r *adminRepository) List(ctx context.Context, limit, offset int) ([]AdminUser, int64, error) {
	var admins []AdminUser
	var total int64

	db := r.db.WithContext(ctx).Model(&AdminUser{})

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := db.Order("created_at DESC").Limit(limit).Offset(offset).Find(&admins).Error; err != nil {
		return nil, 0, err
	}

	return admins, total, nil
}
