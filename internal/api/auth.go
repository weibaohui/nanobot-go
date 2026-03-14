package api

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/weibaohui/nanobot-go/internal/models"
)

// JWT 密钥从环境变量读取
func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// 开发环境使用默认值，但输出警告
		return []byte("nanobot-dev-secret-key-change-in-production")
	}
	return []byte(secret)
}

// JWT 令牌有效期从环境变量读取，默认 7 天
func getTokenDuration() time.Duration {
	duration := os.Getenv("JWT_TOKEN_DURATION")
	if duration == "" {
		return 7 * 24 * time.Hour
	}
	d, err := time.ParseDuration(duration)
	if err != nil {
		return 7 * 24 * time.Hour
	}
	return d
}

// ValidateJWTConfig 检查 JWT 配置是否安全
func ValidateJWTConfig() error {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return fmt.Errorf("JWT_SECRET environment variable is required. Please set a secure secret key")
	}
	if len(secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters long for security")
	}
	return nil
}

// Claims JWT 声明
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken 生成 JWT 令牌
func GenerateToken(user *models.User) (string, error) {
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(getTokenDuration())),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.Username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

// ParseToken 解析 JWT 令牌
func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return getJWTSecret(), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// GetUserIDFromToken 从令牌中获取用户ID
func GetUserIDFromToken(tokenString string) (uint, error) {
	claims, err := ParseToken(tokenString)
	if err != nil {
		return 0, err
	}
	return claims.UserID, nil
}
