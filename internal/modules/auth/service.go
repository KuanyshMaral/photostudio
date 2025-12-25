package auth

import (
	"context"
	"strings"

	"photostudio/internal/domain"
	"photostudio/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

type jwtService interface {
	GenerateToken(userID int64, role string) (string, error)
}

type Service struct {
	users *repository.UserRepository
	jwt   jwtService
}

func NewService(users *repository.UserRepository, jwt jwtService) *Service {
	return &Service{users: users, jwt: jwt}
}

func (s *Service) RegisterClient(ctx context.Context, req RegisterClientRequest) (*domain.User, string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	u := &domain.User{
		Email:         strings.TrimSpace(req.Email),
		PasswordHash:  string(hash),
		Role:          domain.RoleClient,
		Name:          req.Name,
		Phone:         req.Phone,
		EmailVerified: false,
	}

	if err := s.users.Create(ctx, u); err != nil {
		return nil, "", err
	}

	token, err := s.jwt.GenerateToken(u.ID, string(u.Role))
	if err != nil {
		return nil, "", err
	}

	u.PasswordHash = ""
	return u, token, nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*domain.User, string, error) {
	u, err := s.users.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, "", err
	}
	
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	token, err := s.jwt.GenerateToken(u.ID, string(u.Role))
	if err != nil {
		return nil, "", err
	}

	u.PasswordHash = ""
	return u, token, nil
}
