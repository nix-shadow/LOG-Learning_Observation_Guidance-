package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"google.golang.org/api/idtoken"
	"gorm.io/gorm"
	"log-backend/internal/domain"
)

// ErrOTPCooldown lets handlers map the re-request window to a real 429
// instead of a misleading 500.
var ErrOTPCooldown = errors.New("please wait 1 minute before requesting another OTP")

// ErrEmailTaken lets the register handler answer 409 with the honest state
// instead of leaking a raw unique-constraint error.
var ErrEmailTaken = errors.New("a user with this email already exists")

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
		return ErrOTPCooldown
	}

	// Only delete when the previous OTP is dead (expired or near-expiry) —
	// refreshing a live OTP would let anyone who knows a phone number keep the
	// victim permanently locked out by re-requesting every minute.
	if err := s.authRepo.DeleteExpiredOTPs(ctx); err != nil {
		slog.Warn("cleanup of expired OTPs failed", "error", err)
	}

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
	// Replace any prior (near-expiry) record for the phone.
	if err := s.authRepo.DeleteOTP(ctx, phone); err != nil {
		return err
	}
	return s.authRepo.SaveOTP(ctx, &record)
}

func (s *authService) VerifyOTP(ctx context.Context, phone, otp string) (*domain.User, string, error) {
	record, err := s.authRepo.FindOTPByPhone(ctx, phone)
	if err != nil {
		return nil, "", fmt.Errorf("invalid OTP")
	}

	if record.ExpiresAt.Before(time.Now()) {
		_ = s.authRepo.DeleteOTP(ctx, phone)
		return nil, "", fmt.Errorf("otp has expired")
	}

	if !CheckPasswordHash(otp, record.OTP) {
		err := s.authRepo.IncrementOTPAttempts(ctx, phone)
		if err != nil {
			return nil, "", fmt.Errorf("invalid OTP")
		}

		updatedRecord, err := s.authRepo.FindOTPByPhone(ctx, phone)
		if err == nil && updatedRecord.Attempts >= 5 {
			_ = s.authRepo.DeleteOTP(ctx, phone)
			return nil, "", fmt.Errorf("too many incorrect attempts, please request a new OTP")
		}
		return nil, "", fmt.Errorf("invalid OTP")
	}

	_ = s.authRepo.DeleteOTP(ctx, phone)

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
		// Soft-deleted accounts come back as plain STUDENTs — never with their
		// old role. A removed admin must not regain ADMIN by dialing their
		// phone number; re-provisioning is the only path back to privilege.
		user.DeletedAt = gorm.DeletedAt{Valid: false}
		user.Role = domain.RoleStudent
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

func (s *authService) Register(ctx context.Context, name, email, password string) (*domain.User, string, error) {
	if existing, err := s.userRepo.FindByEmail(ctx, email); err == nil && existing != nil {
		return nil, "", ErrEmailTaken
	}

	hashed, err := HashPassword(password)
	if err != nil {
		return nil, "", err
	}

	user := &domain.User{
		ID:           GenerateSecureID("user"),
		Name:         name,
		Email:        email,
		PasswordHash: hashed,
		Role:         domain.RoleStudent,
		IsVerified:   true,
		CreatedAt:    time.Now(),
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, "", err
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
		return nil, "", fmt.Errorf("invalid email or password")
	}

	if !CheckPasswordHash(password, user.PasswordHash) {
		return nil, "", fmt.Errorf("invalid email or password")
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
		return nil, "", fmt.Errorf("google auth is not configured on the server")
	}

	payload, err := idtoken.Validate(ctx, token, clientID)
	if err != nil {
		// Log the provider detail server-side; the client gets one generic
		// message (validation internals are not user-facing information).
		slog.Warn("google token validation failed", "error", err)
		return nil, "", fmt.Errorf("invalid google token")
	}

	email, ok := payload.Claims["email"].(string)
	if !ok {
		return nil, "", fmt.Errorf("email not found in google token")
	}
	if verified, ok := payload.Claims["email_verified"].(bool); !ok || !verified {
		return nil, "", fmt.Errorf("google email is not verified")
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
