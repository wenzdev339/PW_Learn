package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"sync"
	"time"

	"backend/internal/apperror"
	"backend/internal/config"
	"backend/internal/models"
	"backend/internal/password"
	"backend/internal/token"

	"gorm.io/gorm"
)

func RegisterUser(db *gorm.DB, email, plainPassword, name string) (*models.User, error) {
	var existing models.User
	err := db.Where("email = ?", email).First(&existing).Error
	if err == nil {
		return nil, apperror.New(409, "EMAIL_TAKEN", "Email is already registered")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hash, err := password.Hash(plainPassword)
	if err != nil {
		return nil, err
	}

	user := models.User{Email: email, PasswordHash: hash, Name: name, Role: models.RoleCustomer}
	if err := db.Create(&user).Error; err != nil {
		return nil, err
	}

	cart := models.Cart{UserID: user.ID}
	if err := db.Create(&cart).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func LoginUser(db *gorm.DB, email, plainPassword string) (*models.User, error) {
	var user models.User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, apperror.New(401, "INVALID_CREDENTIALS", "Invalid email or password")
	}
	if !password.Verify(plainPassword, user.PasswordHash) {
		return nil, apperror.New(401, "INVALID_CREDENTIALS", "Invalid email or password")
	}
	return &user, nil
}

type tokenPair struct {
	AccessToken  string
	RefreshToken string
}

func issueTokens(cfg config.Config, userID, role string) (tokenPair, error) {
	access, err := token.SignAccessToken(userID, role, cfg.JWTAccessSecret)
	if err != nil {
		return tokenPair{}, err
	}
	refresh, err := token.SignRefreshToken(userID, role, cfg.JWTRefreshSecret)
	if err != nil {
		return tokenPair{}, err
	}
	return tokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

func RotateRefreshToken(db *gorm.DB, cfg config.Config, refreshToken string) (tokenPair, error) {
	claims, err := token.Verify(refreshToken, cfg.JWTRefreshSecret)
	if err != nil {
		return tokenPair{}, apperror.New(401, "INVALID_REFRESH_TOKEN", "Invalid or expired refresh token")
	}
	var user models.User
	if err := db.First(&user, "id = ?", claims.Subject).Error; err != nil {
		return tokenPair{}, apperror.New(401, "INVALID_REFRESH_TOKEN", "User no longer exists")
	}
	return issueTokens(cfg, user.ID, string(user.Role))
}

type resetEntry struct {
	userID    string
	expiresAt time.Time
}

var (
	resetTokensMu sync.Mutex
	resetTokens   = map[string]resetEntry{}
)

// RequestPasswordReset never reveals whether the email exists: it always
// returns nil, logging the reset token instead of sending an email.
func RequestPasswordReset(db *gorm.DB, email string) error {
	var user models.User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil
	}

	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}
	resetToken := hex.EncodeToString(tokenBytes)

	resetTokensMu.Lock()
	resetTokens[resetToken] = resetEntry{userID: user.ID, expiresAt: time.Now().Add(15 * time.Minute)}
	resetTokensMu.Unlock()

	log.Printf("[password-reset] token for %s: %s", email, resetToken)
	return nil
}

func ResetPassword(db *gorm.DB, resetToken, newPassword string) error {
	resetTokensMu.Lock()
	entry, ok := resetTokens[resetToken]
	if ok {
		delete(resetTokens, resetToken)
	}
	resetTokensMu.Unlock()

	if !ok || time.Now().After(entry.expiresAt) {
		return apperror.New(400, "INVALID_RESET_TOKEN", "Invalid or expired reset token")
	}

	hash, err := password.Hash(newPassword)
	if err != nil {
		return err
	}
	return db.Model(&models.User{}).Where("id = ?", entry.userID).Update("password_hash", hash).Error
}
