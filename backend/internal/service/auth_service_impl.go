package service

import (
	"context"
	"fmt"
	"time"

	"log-backend/internal/domain"
	"google.golang.org/api/idtoken"
)

type authService struct {
	userRepo domain.UserRepository
	authRepo domain.AuthRepository
}

func NewAuthService(userRepo domain.UserRepository, authRepo domain.AuthRepository) AuthService {
	return &authService{
		userRepo: userRepo,
		authRepo: authRepo,
	}
}

func (s *authService) RequestOTP(ctx context.Context, phone string) error {
	s.authRepo.DeleteOTP(ctx, phone)
	s.authRepo.DeleteExpiredOTPs(ctx)

	otpInt, err := generateSecureOTP()
	if err != nil {
		return err
	}
	otp := fmt.Sprintf("%06d", otpInt)

	otpHash, err := HashPassword(otp)
	if err != nil {
		return err
	}

	record := domain.OTPRecord{
		Phone:     phone,
		OTP:       otpHash,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	return s.authRepo.SaveOTP(ctx, &record)
}

func (s *authService) VerifyOTP(ctx context.Context, phone, otp string) (string, error) {
	record, err := s.authRepo.FindOTPByPhone(ctx, phone)
	if err != nil {
		return "", fmt.Errorf("Invalid OTP")
	}

	if record.ExpiresAt.Before(time.Now()) {
		s.authRepo.DeleteOTP(ctx, phone)
		return "", fmt.Errorf("OTP has expired")
	}

	if !CheckPasswordHash(otp, record.OTP) {
		return "", fmt.Errorf("Invalid OTP")
	}

	s.authRepo.DeleteOTP(ctx, phone)

	user, err := s.userRepo.FindByPhone(ctx, phone)
	if err != nil {
		user = &domain.User{
			ID:         GenerateSecureID("user"),
			Phone:      phone,
			Role:       domain.RoleStudent,
			IsVerified: true,
			CreatedAt:  time.Now(),
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			return "", err
		}
	}

	return GenerateJWT(user.ID, user.Role)
}

func (s *authService) Login(ctx context.Context, email, password string) (*domain.User, string, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, "", fmt.Errorf("Invalid email or password")
	}

	if !CheckPasswordHash(password, user.PasswordHash) {
		return nil, "", fmt.Errorf("Invalid email or password")
	}

	token, err := GenerateJWT(user.ID, user.Role)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

func (s *authService) Register(ctx context.Context, user *domain.User, password string) (*domain.User, error) {
	_, err := s.userRepo.FindByEmail(ctx, user.Email)
	if err == nil {
		return nil, fmt.Errorf("User with this email already exists")
	}

	hashedPassword, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	user.ID = GenerateSecureID("usr")
	user.PasswordHash = hashedPassword
	user.Role = domain.RoleStudent
	user.IsVerified = true
	user.CreatedAt = time.Now()

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *authService) GoogleAuth(ctx context.Context, token string) (*domain.User, string, error) {
	// Skip validation if demo token
	if token == "mock-demo-token-123" {
		return &domain.User{
			ID:         "user-123",
			Name:       "Aisha Student",
			Email:      "aisha@example.com",
			Role:       domain.RoleStudent,
			IsVerified: true,
		}, "mock-jwt", nil
	}

	payload, err := idtoken.Validate(ctx, token, "")
	if err != nil {
		return nil, "", fmt.Errorf("Invalid Google token")
	}

	email, ok := payload.Claims["email"].(string)
	if !ok {
		return nil, "", fmt.Errorf("Email not found in Google token")
	}
	name, _ := payload.Claims["name"].(string)

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		user = &domain.User{
			ID:         GenerateSecureID("user-g"),
			Name:       name,
			Email:      email,
			Role:       domain.RoleStudent,
			IsVerified: true,
			CreatedAt:  time.Now(),
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, "", err
		}
	}

	jwtStr, err := GenerateJWT(user.ID, user.Role)
	if err != nil {
		return nil, "", err
	}
	return user, jwtStr, nil
}

func (s *authService) Logout(ctx context.Context, jti, userID string, exp float64) error {
	expTime := time.Unix(int64(exp), 0)
	revocation := domain.TokenBlocklist{
		JTI:       jti,
		UserID:    userID,
		ExpiresAt: expTime,
		RevokedAt: time.Now(),
	}
	return s.authRepo.BlockToken(ctx, &revocation)
}
