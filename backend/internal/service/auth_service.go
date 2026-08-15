package service

import (
	"context"
	"log-backend/internal/domain"
)

type AuthService interface {
	RequestOTP(ctx context.Context, phone string) error
	VerifyOTP(ctx context.Context, phone, otp string) (string, error) // Returns JWT
	Login(ctx context.Context, email, password string) (*domain.User, string, error) // Returns User, JWT
	Register(ctx context.Context, user *domain.User, password string) (*domain.User, error)
	GoogleAuth(ctx context.Context, token string) (*domain.User, string, error)
	Logout(ctx context.Context, jti, userID string, exp float64) error
}
