package admin

import (
	"context"
	"errors"
	"strings"
	"time"

	"photostudio/internal/domain"

	"gorm.io/gorm"
)

type Service struct {
	userRepo        UserRepository
	studioRepo      StudioRepository
	bookingRepo     BookingRepository
	reviewRepo      ReviewRepository
	studioOwnerRepo StudioOwnerRepository
	notifs          NotificationSender
}

func NewService(
	userRepo UserRepository,
	studioRepo StudioRepository,
	bookingRepo BookingRepository,
	reviewRepo ReviewRepository,
	studioOwnerRepo StudioOwnerRepository,
	notifs NotificationSender,
) *Service {
	return &Service{
		userRepo:        userRepo,
		studioRepo:      studioRepo,
		bookingRepo:     bookingRepo,
		reviewRepo:      reviewRepo,
		studioOwnerRepo: studioOwnerRepo,
		notifs:          notifs,
	}
}

// -------------------- Studios --------------------

func (s *Service) GetPendingStudioOwners(ctx context.Context, page, limit int) ([]PendingStudioDTO, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	rows, total, err := s.studioOwnerRepo.FindPendingPaginated(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	out := make([]PendingStudioDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, PendingStudioDTO{
			ID:          r.ID,
			UserID:      r.UserID,
			BIN:         r.BIN,
			CompanyName: r.CompanyName,
			Status:      r.Status,
			CreatedAt:   r.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	return out, total, nil
}

func (s *Service) ApproveStudioOwner(ctx context.Context, studioOwnerID, adminID int64) error {
	owner, err := s.studioOwnerRepo.FindByID(ctx, studioOwnerID)
	if err != nil {
		return errors.New("studio owner not found")
	}

	u, err := s.userRepo.GetByID(ctx, owner.UserID)
	if err != nil {
		return errors.New("user not found")
	}

	if u.StudioStatus != domain.StatusPending {
		return errors.New("can only approve pending applications")
	}

	now := time.Now()
	u.StudioStatus = domain.StatusVerified
	if err := s.userRepo.Update(ctx, u); err != nil {
		return err
	}

	owner.VerifiedAt = &now
	owner.VerifiedBy = &adminID
	owner.RejectedReason = ""
	if err := s.studioOwnerRepo.Update(ctx, owner); err != nil {
		return err
	}

	// уведомления (если реализованы)
	if s.notifs != nil {
		_ = s.notifs.NotifyVerificationApproved(ctx, owner.UserID, 0)
	}

	return nil
}

func (s *Service) RejectStudioOwner(ctx context.Context, studioOwnerID, adminID int64, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return errors.New("reason is required")
	}

	owner, err := s.studioOwnerRepo.FindByID(ctx, studioOwnerID)
	if err != nil {
		return errors.New("studio owner not found")
	}

	u, err := s.userRepo.GetByID(ctx, owner.UserID)
	if err != nil {
		return errors.New("user not found")
	}

	if u.StudioStatus != domain.StatusPending {
		return errors.New("can only reject pending applications")
	}

	u.StudioStatus = domain.StatusRejected
	if err := s.userRepo.Update(ctx, u); err != nil {
		return err
	}

	owner.VerifiedAt = nil
	owner.VerifiedBy = &adminID
	owner.RejectedReason = reason
	if err := s.studioOwnerRepo.Update(ctx, owner); err != nil {
		return err
	}

	if s.notifs != nil {
		_ = s.notifs.NotifyVerificationRejected(ctx, owner.UserID, 0, reason)
	}

	return nil
}

// GetPendingStudios returns studios with status "pending"
func (s *Service) GetPendingStudios(ctx context.Context, page, limit int) ([]domain.Studio, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	// pending = owner(user).studio_status = 'pending'
	// считаем total корректно
	var total int64
	if err := s.studioRepo.DB().WithContext(ctx).
		Table("studios").
		Joins("JOIN users u ON u.id = studios.owner_id").
		Where("u.studio_status = ? AND studios.deleted_at IS NULL", domain.StatusPending).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var studios []domain.Studio
	if err := s.studioRepo.DB().WithContext(ctx).
		Table("studios").
		Joins("JOIN users u ON u.id = studios.owner_id").
		Where("u.studio_status = ? AND studios.deleted_at IS NULL", domain.StatusPending).
		Order("studios.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&studios).Error; err != nil {
		return nil, 0, err
	}

	return studios, int(total), nil
}

// VerifyStudio changes status to "verified"
func (s *Service) VerifyStudio(ctx context.Context, studioID, adminID int64, notes string) (*domain.Studio, error) {
	studio, err := s.studioRepo.GetByID(ctx, studioID)
	if err != nil {
		return nil, err
	}

	owner, err := s.userRepo.GetByID(ctx, studio.OwnerID)
	if err != nil {
		return nil, err
	}

	owner.StudioStatus = domain.StatusVerified
	if err := s.userRepo.Update(ctx, owner); err != nil {
		return nil, err
	}

	if s.notifs != nil {
		_ = s.notifs.NotifyVerificationApproved(ctx, owner.ID, studio.ID)
	}

	// TODO later: studio_owner.AdminNotes/VerifiedBy/VerifiedAt
	_ = adminID
	_ = notes

	return studio, nil
}

func (s *Service) RejectStudio(ctx context.Context, studioID, adminID int64, reason string) (*domain.Studio, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, errors.New("reason is required")
	}

	studio, err := s.studioRepo.GetByID(ctx, studioID)
	if err != nil {
		return nil, err
	}

	owner, err := s.userRepo.GetByID(ctx, studio.OwnerID)
	if err != nil {
		return nil, err
	}

	owner.StudioStatus = domain.StatusRejected
	if err := s.userRepo.Update(ctx, owner); err != nil {
		return nil, err
	}

	if s.notifs != nil {
		_ = s.notifs.NotifyVerificationRejected(ctx, studio.OwnerID, studio.ID, reason)
	}

	// TODO later: studio_owner.RejectedReason/VerifiedBy
	_ = adminID

	return studio, nil
}

// -------------------- Statistics --------------------

func (s *Service) GetPlatformStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalUsers int64
	if err := s.userRepo.DB().WithContext(ctx).Table("users").Count(&totalUsers).Error; err != nil {
		return nil, err
	}
	stats["total_users"] = totalUsers

	var totalStudios int64
	if err := s.studioRepo.DB().WithContext(ctx).Table("studios").Where("deleted_at IS NULL").Count(&totalStudios).Error; err != nil {
		return nil, err
	}
	stats["total_studios"] = totalStudios

	var totalBookings int64
	if err := s.bookingRepo.DB().WithContext(ctx).Table("bookings").Count(&totalBookings).Error; err != nil {
		return nil, err
	}
	stats["total_bookings"] = totalBookings

	var pendingStudios int64
	if err := s.userRepo.DB().WithContext(ctx).Table("users").Where("studio_status = ?", domain.StatusPending).Count(&pendingStudios).Error; err != nil {
		return nil, err
	}
	stats["pending_studios"] = pendingStudios

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthEnd := monthStart.AddDate(0, 1, 0)

	var completedThisMonth int64
	if err := s.bookingRepo.DB().WithContext(ctx).
		Table("bookings").
		Where("status = ? AND updated_at >= ? AND updated_at < ?", domain.BookingCompleted, monthStart, monthEnd).
		Count(&completedThisMonth).Error; err != nil {
		return nil, err
	}
	stats["completed_bookings_this_month"] = completedThisMonth

	return stats, nil
}

func (s *Service) GetStatistics(ctx context.Context) (*StatisticsResponse, error) {
	var totalUsers int64
	if err := s.userRepo.DB().WithContext(ctx).Table("users").Count(&totalUsers).Error; err != nil {
		return nil, err
	}

	var totalStudios int64
	if err := s.studioRepo.DB().WithContext(ctx).Table("studios").Where("deleted_at IS NULL").Count(&totalStudios).Error; err != nil {
		return nil, err
	}

	var totalBookings int64
	if err := s.bookingRepo.DB().WithContext(ctx).Table("bookings").Count(&totalBookings).Error; err != nil {
		return nil, err
	}

	var pendingStudios int64
	if err := s.studioRepo.DB().WithContext(ctx).
		Table("studios").
		Joins("JOIN users u ON u.id = studios.owner_id").
		Where("u.studio_status = ? AND studios.deleted_at IS NULL", domain.StatusPending).
		Count(&pendingStudios).Error; err != nil {
		return nil, err
	}

	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.Add(24 * time.Hour)

	var completedBookingsThisMonth int64
	if err := s.bookingRepo.DB().WithContext(ctx).
		Table("bookings").
		Where("created_at >= ? AND created_at < ?", start, end).
		Count(&completedBookingsThisMonth).Error; err != nil {
		return nil, err
	}

	return &StatisticsResponse{
		TotalUsers:                 int(totalUsers),
		TotalStudios:               int(totalStudios),
		TotalBookings:              int(totalBookings),
		PendingStudios:             int(pendingStudios),
		CompletedBookingsThisMonth: int(completedBookingsThisMonth),
	}, nil
}

// -------------------- Users moderation --------------------
//
// В твоём domain нет RoleBlocked (и это правильно).
// Поэтому блокировку делаем через StudioStatus = blocked.
// Это логично для studio_owner, но для client тоже сработает как "global disable flag".
// Позже можно выделить отдельное поле is_blocked.

func (s *Service) BanUser(ctx context.Context, userID int64, reason string) error {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return errors.New("user not found")
	}

	if u.Role == domain.RoleAdmin {
		return errors.New("cannot ban admin users")
	}

	u.StudioStatus = domain.StatusBlocked
	if err := s.userRepo.Update(ctx, u); err != nil {
		return err
	}

	_ = reason // в БД нет поля для сохранения причины — позже можно добавить
	return nil
}

func (s *Service) UnbanUser(ctx context.Context, userID int64) error {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return errors.New("user not found")
	}

	if u.StudioStatus == domain.StatusBlocked {
		u.StudioStatus = domain.StatusVerified
	}
	return s.userRepo.Update(ctx, u)
}

func (s *Service) BlockUser(ctx context.Context, userID int64, reason string) (*domain.User, error) {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	u.StudioStatus = domain.StatusBlocked
	if err := s.userRepo.Update(ctx, u); err != nil {
		return nil, err
	}

	_ = reason // если добавишь поле BlockReason — сохраним

	return u, nil
}

func (s *Service) UnblockUser(ctx context.Context, userID int64) (*domain.User, error) {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Возвращаем в verified (или pending — если хочешь вернуть в исходный статус)
	u.StudioStatus = domain.StatusVerified
	if err := s.userRepo.Update(ctx, u); err != nil {
		return nil, err
	}

	return u, nil
}

// ListUsers supports simple filters + pagination
func (s *Service) ListUsers(ctx context.Context, filter UserListFilter, page, limit int) ([]domain.User, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	q := s.userRepo.DB().WithContext(ctx).Table("users")

	if strings.TrimSpace(filter.Role) != "" {
		q = q.Where("role = ?", strings.TrimSpace(filter.Role))
	}

	if filter.Blocked != nil {
		if *filter.Blocked {
			q = q.Where("studio_status = ?", domain.StatusBlocked)
		} else {
			q = q.Where("(studio_status IS NULL OR studio_status <> ?)", domain.StatusBlocked)
		}
	}

	if strings.TrimSpace(filter.Query) != "" {
		sv := "%" + strings.ToLower(strings.TrimSpace(filter.Query)) + "%"
		q = q.Where("LOWER(email) LIKE ? OR LOWER(name) LIKE ?", sv, sv)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// грузим в domain.User напрямую (у тебя json теги, а gorm по колонкам совпадает)
	var users []domain.User
	if err := q.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&users).Error; err != nil {
		return nil, 0, err
	}

	// safety: не отдаём hash
	for i := range users {
		users[i].PasswordHash = ""
	}

	return users, int(total), nil
}

// -------------------- Reviews moderation --------------------

// ListReviews supports filters + pagination
func (s *Service) ListReviews(ctx context.Context, filter ReviewListFilter, page, limit int) ([]domain.Review, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	q := s.reviewRepo.DB().WithContext(ctx).Table("reviews")

	if filter.StudioID != nil {
		q = q.Where("studio_id = ?", *filter.StudioID)
	}
	if filter.UserID != nil {
		q = q.Where("user_id = ?", *filter.UserID)
	}
	if filter.Hidden != nil {
		q = q.Where("is_hidden = ?", *filter.Hidden)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var reviews []domain.Review
	if err := q.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&reviews).Error; err != nil {
		return nil, 0, err
	}

	return reviews, int(total), nil
}

func (s *Service) HideReview(ctx context.Context, reviewID int64) (*domain.Review, error) {
	rv, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		return nil, err
	}

	rv.IsHidden = true
	if err := s.reviewRepo.Update(ctx, rv); err != nil {
		return nil, err
	}

	return rv, nil
}

func (s *Service) ShowReview(ctx context.Context, reviewID int64) (*domain.Review, error) {
	rv, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		return nil, err
	}

	rv.IsHidden = false
	if err := s.reviewRepo.Update(ctx, rv); err != nil {
		return nil, err
	}

	return rv, nil
}

// helper for gorm not found
func isNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
