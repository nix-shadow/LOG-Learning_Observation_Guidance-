package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"log-backend/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret []byte

func init() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-only-change-me-in-production-min-32-chars!"
	}
	jwtSecret = []byte(secret)
}

func GetJWTSecret() []byte {
	return jwtSecret
}

func GenerateSecureID(prefix string) string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b))
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func generateSecureOTP() (int, error) {
	b := make([]byte, 4)
	_, err := rand.Read(b)
	if err != nil {
		return 0, err
	}
	val := uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
	return int(100000 + (val % 900000)), nil
}

func GenerateJWT(userID string, role domain.Role) (string, error) {
	jti := GenerateSecureID("jti") // Unique JWT ID for revocation
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  userID,
		"role": role,
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
		"jti":  jti,
	})
	return token.SignedString(jwtSecret)
}
