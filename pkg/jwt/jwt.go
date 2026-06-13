package jwt

import (
	"errors"
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
	jwt.RegisteredClaims
}

var (
	secret          string
	expireHour      int
	refreshExpireDay int

	ErrTokenExpired = errors.New("token已过期")
	ErrTokenInvalid = errors.New("token无效")
)

func Init(sec string, expire int, refreshExpire int) {
	secret = sec
	expireHour = expire
	refreshExpireDay = refreshExpire
}

// Generate 生成 Access Token（短时效）
func Generate(userID uint, username string) (string, error) {
	claims := Claims{
		UserID:    userID,
		Username:  username,
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireHour) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateRefreshToken 生成 Refresh Token（长时效），用于获取新的 Access Token
func GenerateRefreshToken(userID uint, username string) (string, error) {
	claims := Claims{
		UserID:    userID,
		Username:  username,
		TokenType: TokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
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
