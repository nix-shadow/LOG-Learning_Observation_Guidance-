package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"log-backend/internal/domain"
)

var jwtSecret []byte

func init() {
	secret := os.Getenv("JWT_SECRET")
	// The main function will abort if this is empty.
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

// GenerateInviteCode returns a short, human-friendly class join code
// (WP-1.5): unambiguous uppercase letters+digits, easy to read aloud on a
// school LAN. Collisions are handled by the DB unique index at create time.
func GenerateInviteCode() string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
	}
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b)
}

// GenerateTempPassword produces the one-time credential handed to an
// imported student's teacher (WP-1.5 roster import). 10 chars, unambiguous
// alphabet — never logged, returned exactly once in the import report.
func GenerateTempPassword() string {
	const alphabet = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ23456789"
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("LOG-%d", time.Now().UnixNano())
	}
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b)
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
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
		"jti":  jti,
	})
	return token.SignedString(jwtSecret)
}
