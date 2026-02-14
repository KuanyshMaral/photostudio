package config

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

const (
	defaultJWTAccessTTL       = "15m"
	defaultRefreshTTL         = "168h"
	defaultVerifyCodeTTL      = "5m"
	defaultVerifyResend       = "60s"
	defaultCookieSecure       = "false"
	defaultCookieSameSite     = "Lax"
	defaultCookiePath         = "/api/v1/auth"
	defaultJWTAllowLegacy     = "true"
	defaultJWTSecret          = "change-me-jwt-secret"
	defaultRefreshTokenPepper = "change-me-refresh-pepper"
	defaultVerifyCodePepper   = "change-me-verification-pepper"
)

type AuthRuntimeConfig struct {
	AppEnv                 string
	JWTAllowLegacyClaims   bool
	JWTSecret              string
	JWTAccessTTL           time.Duration
	RefreshTTL             time.Duration
	RefreshTokenPepper     string
	VerificationCodePepper string
	VerifyCodeTTL          time.Duration
	VerifyResendCooldown   time.Duration
	CookieSecure           bool
	CookieSameSite         string
	CookiePath             string
}

func LoadAuthRuntimeConfig() (*AuthRuntimeConfig, error) {
	cfg := &AuthRuntimeConfig{}
	appEnv := strings.TrimSpace(os.Getenv("APP_ENV"))
	if appEnv == "" {
		appEnv = strings.TrimSpace(os.Getenv("ENV"))
	}
	if appEnv == "" {
		appEnv = "dev"
	}
	cfg.AppEnv = strings.ToLower(appEnv)

	cfg.JWTSecret = strings.TrimSpace(getEnv("JWT_SECRET", defaultJWTSecret))
	cfg.RefreshTokenPepper = strings.TrimSpace(getEnv("REFRESH_TOKEN_PEPPER", defaultRefreshTokenPepper))
	cfg.VerificationCodePepper = strings.TrimSpace(getEnv("VERIFICATION_CODE_PEPPER", defaultVerifyCodePepper))

	var err error
	cfg.JWTAccessTTL, err = parseDurationEnv("JWT_ACCESS_TTL", defaultJWTAccessTTL)
	if err != nil {
		return nil, err
	}

	cfg.RefreshTTL, err = parseDurationEnv("REFRESH_TTL", defaultRefreshTTL)
	if err != nil {
		return nil, err
	}

	cfg.VerifyCodeTTL, err = parseDurationEnv("VERIFY_CODE_TTL", defaultVerifyCodeTTL)
	if err != nil {
		return nil, err
	}

	cfg.VerifyResendCooldown, err = parseDurationEnv("VERIFY_RESEND_COOLDOWN", defaultVerifyResend)
	if err != nil {
		return nil, err
	}

	cfg.CookieSecure = parseBoolEnv("COOKIE_SECURE", defaultCookieSecure)
	cfg.CookieSameSite = strings.TrimSpace(getEnv("COOKIE_SAMESITE", defaultCookieSameSite))
	cfg.CookiePath = strings.TrimSpace(getEnv("COOKIE_PATH", defaultCookiePath))
	cfg.JWTAllowLegacyClaims = parseBoolEnv("JWT_ALLOW_LEGACY_CLAIMS", defaultJWTAllowLegacy)

	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	log.Printf("auth cookie config: secure=%t, sameSite=%s, path=%s", cfg.CookieSecure, cfg.CookieSameSite, cfg.CookiePath)

	return cfg, nil
}

func validateConfig(cfg *AuthRuntimeConfig) error {
	if cfg.JWTAccessTTL <= 0 {
		return fmt.Errorf("JWT_ACCESS_TTL must be > 0")
	}
	if cfg.RefreshTTL <= 0 {
		return fmt.Errorf("REFRESH_TTL must be > 0")
	}
	if cfg.VerifyCodeTTL <= 0 {
		return fmt.Errorf("VERIFY_CODE_TTL must be > 0")
	}
	if cfg.VerifyResendCooldown <= 0 {
		return fmt.Errorf("VERIFY_RESEND_COOLDOWN must be > 0")
	}
	if cfg.CookiePath == "" {
		return fmt.Errorf("COOKIE_PATH must not be empty")
	}
	if cfg.CookieSameSite == "" {
		return fmt.Errorf("COOKIE_SAMESITE must not be empty")
	}
	sameSite := strings.ToLower(strings.TrimSpace(cfg.CookieSameSite))
	if sameSite != "lax" && sameSite != "none" && sameSite != "strict" {
		return fmt.Errorf("COOKIE_SAMESITE must be one of: Lax, None, Strict")
	}
	if sameSite == "none" && !cfg.CookieSecure {
		return fmt.Errorf("COOKIE_SECURE must be true when COOKIE_SAMESITE=None")
	}

	if isProdLike(cfg.AppEnv) {
		if isEmptyOrDefault(cfg.JWTSecret, defaultJWTSecret) {
			return fmt.Errorf("in prod/release JWT_SECRET must be set and not default")
		}
		if isEmptyOrDefault(cfg.RefreshTokenPepper, defaultRefreshTokenPepper) {
			return fmt.Errorf("in prod/release REFRESH_TOKEN_PEPPER must be set and not default")
		}
		if isEmptyOrDefault(cfg.VerificationCodePepper, defaultVerifyCodePepper) {
			return fmt.Errorf("in prod/release VERIFICATION_CODE_PEPPER must be set and not default")
		}
		if !cfg.CookieSecure {
			return fmt.Errorf("in prod/release COOKIE_SECURE must be true")
		}
	}

	return nil
}

func isProdLike(env string) bool {
	env = strings.ToLower(strings.TrimSpace(env))
	return env == "prod" || env == "production" || env == "release"
}

func isEmptyOrDefault(v, def string) bool {
	trimmed := strings.TrimSpace(v)
	return trimmed == "" || trimmed == def
}

func parseDurationEnv(name, fallback string) (time.Duration, error) {
	value := strings.TrimSpace(getEnv(name, fallback))
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s value %q: %w", name, value, err)
	}
	return d, nil
}

func parseBoolEnv(name, fallback string) bool {
	value := strings.ToLower(strings.TrimSpace(getEnv(name, fallback)))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

func getEnv(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}