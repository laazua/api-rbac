package jwt

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

type Claims struct {
	UserID    uint   `json:"user_id"`
	Username  string `json:"username"`
	TokenType string `json:"token_type"`
	JTI       string `json:"jti"`
	jwt.RegisteredClaims
}

var (
	secret           string
	expireHour       int
	refreshExpireDay int

	ErrTokenExpired = errors.New("token已过期")
	ErrTokenInvalid = errors.New("token无效")
)

// generateJTI 生成唯一令牌 ID（UUID v4 格式）
func generateJTI() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // UUID v4
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func Init(sec string, expire int, refreshExpire int) {
	secret = sec
	expireHour = expire
	refreshExpireDay = refreshExpire
}

// Generate 生成 Access Token（短时效）
func Generate(userID uint, username string) (string, error) {
	jti := generateJTI()
	claims := Claims{
		UserID:    userID,
		Username:  username,
		TokenType: TokenTypeAccess,
		JTI:       jti,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireHour) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateRefreshToken 生成 Refresh Token（长时效），用于获取新的 Access Token
func GenerateRefreshToken(userID uint, username string) (string, error) {
	jti := generateJTI()
	claims := Claims{
		UserID:    userID,
		Username:  username,
		TokenType: TokenTypeRefresh,
		JTI:       jti,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(refreshExpireDay) * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// Parse 解析并验证 Token
func Parse(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

// ParseRefreshToken 解析并验证 Refresh Token（仅接受 refresh 类型）
func ParseRefreshToken(tokenString string) (*Claims, error) {
	claims, err := Parse(tokenString)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != TokenTypeRefresh {
		return nil, errors.New("非法的刷新令牌类型")
	}
	return claims, nil
}

// GetAccessTokenExpireHour 返回 Access Token 的过期时间（小时）
func GetAccessTokenExpireHour() int {
	return expireHour
}

// GetRefreshExpireDay 返回 Refresh Token 的过期时间（天）
func GetRefreshExpireDay() int {
	return refreshExpireDay
}
