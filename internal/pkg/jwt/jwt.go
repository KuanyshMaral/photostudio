package jwt

import (
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	jwtlib "github.com/golang-jwt/jwt/v5"
)

type Service struct {
	secret      []byte
	ttl         time.Duration
	allowLegacy bool
}

type Claims struct {
	// Legacy fallback during migration.
	UserID int64 `json:"user_id,omitempty"`
	Role   string `json:"role"`

	TokenType string `json:"type,omitempty"`

	jwtlib.RegisteredClaims
}

func New(secret string, ttl time.Duration) *Service {
	return NewWithLegacy(secret, ttl, true)
}

func NewWithLegacy(secret string, ttl time.Duration, allowLegacy bool) *Service {
	return &Service{
		secret:      []byte(secret),
		ttl:         ttl,
		allowLegacy: allowLegacy,
	}
}

func (s *Service) GenerateToken(userID int64, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		Role:      role,
		TokenType: "access",
		RegisteredClaims: jwtlib.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			ID:        uuid.NewString(),
			ExpiresAt: jwtlib.NewNumericDate(now.Add(s.ttl)),
			IssuedAt:  jwtlib.NewNumericDate(now),
		},
	}

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *Service) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwtlib.ParseWithClaims(tokenStr, &Claims{}, func(t *jwtlib.Token) (any, error) {
		return s.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	if claims.TokenType == "" {
		if s.allowLegacy {
			claims.TokenType = "access"
		} else {
			return nil, errors.New("invalid token type")
		}
	}
	if claims.TokenType != "access" {
		return nil, errors.New("invalid token type")
	}

	if claims.Subject != "" {
		userID, convErr := strconv.ParseInt(claims.Subject, 10, 64)
		if convErr != nil || userID <= 0 {
			return nil, errors.New("invalid sub claim")
		}
		claims.UserID = userID
		return claims, nil
	}

	if s.allowLegacy && claims.UserID > 0 {
		return claims, nil
	}

	return nil, errors.New("missing subject claim")
}
