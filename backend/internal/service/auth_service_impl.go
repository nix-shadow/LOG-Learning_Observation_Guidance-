package service

import (
	"context"
	"fmt"
	"os"
	"time"

	"google.golang.org/api/idtoken"
	"gorm.io/gorm"
	"log-backend/internal/domain"
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
	existing, err := s.authRepo.FindOTPByPhone(ctx, phone)
	if err == nil && existing.ExpiresAt.After(time.Now().Add(4*time.Minute)) {
		return fmt.Errorf("Please wait 1 minute before requesting another OTP")
	}

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

func (s *authService) VerifyOTP(ctx context.Context, phone, otp string) (*domain.User, string, error) {
	record, err := s.authRepo.FindOTPByPhone(ctx, phone)
	if err != nil {
		return nil, "", fmt.Errorf("Invalid OTP")
	}

	if record.ExpiresAt.Before(time.Now()) {
		s.authRepo.DeleteOTP(ctx, phone)
		return nil, "", fmt.Errorf("OTP has expired")
	}

	if !CheckPasswordHash(otp, record.OTP) {
		err := s.authRepo.IncrementOTPAttempts(ctx, phone)
		if err != nil {
			return nil, "", fmt.Errorf("Invalid OTP")
		}

		updatedRecord, err := s.authRepo.FindOTPByPhone(ctx, phone)
		if err == nil && updatedRecord.Attempts >= 5 {
			s.authRepo.DeleteOTP(ctx, phone)
			return nil, "", fmt.Errorf("Too many incorrect attempts. Please request a new OTP")
		}
		return nil, "", fmt.Errorf("Invalid OTP")
	}

	s.authRepo.DeleteOTP(ctx, phone)

	user, err := s.userRepo.FindByPhoneUnscoped(ctx, phone)
	if err != nil {
		p := phone
		user = &domain.User{
			ID:         GenerateSecureID("user"),
			Phone:      &p,
			Role:       domain.RoleStudent,
			IsVerified: true,
			CreatedAt:  time.Now(),
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, "", err
		}
	} else if user.DeletedAt.Valid {
		user.DeletedAt = gorm.DeletedAt{Valid: false}
		if err := s.userRepo.Update(ctx, user); err != nil {
			return nil, "", err
		}
	}

	token, err := GenerateJWT(user.ID, user.Role)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
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

func (s *authService) GoogleAuth(ctx context.Context, token string) (*domain.User, string, error) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		return nil, "", fmt.Errorf("Google Auth is not configured on the server")
	}

	payload, err := idtoken.Validate(ctx, token, clientID)
	if err != nil {
		return nil, "", fmt.Errorf("Invalid Google token: %v", err)
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

func (s *authService) UpdatePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	if user.PasswordHash != "" && !CheckPasswordHash(oldPassword, user.PasswordHash) {
		return fmt.Errorf("incorrect old password")
	}

	hashedPassword, err := HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password")
	}

	user.PasswordHash = hashedPassword
	return s.userRepo.Update(ctx, user)
}
