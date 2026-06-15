package service

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/laazua/api-rbac/internal/model"
	"github.com/laazua/api-rbac/internal/repository"
	jwtpkg "github.com/laazua/api-rbac/pkg/jwt"
)

type AuthService struct {
	userRepo *repository.UserRepo
}

func NewAuthService(userRepo *repository.UserRepo) *AuthService {
	return &AuthService{userRepo: userRepo}
}

// GetByID 根据 ID 获取用户信息
func (s *AuthService) GetByID(id uint) (*model.User, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// LoginResult 登录返回结果
type LoginResult struct {
	Token        string
	RefreshToken string
	ExpiresIn    int64 // Access Token 过期时间（秒）
	User         *model.User
}

func (s *AuthService) Login(req *model.LoginRequest) (*LoginResult, error) {
	// 优先按用户名查找，若未找到则尝试按邮箱查找
	user, err := s.userRepo.FindByUsername(req.Account)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 尝试邮箱登录
			user, err = s.userRepo.FindByEmail(req.Account)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, errors.New("用户名或密码错误")
				}
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	if user.Status == 0 {
		return nil, errors.New("用户名或密码错误")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	token, err := jwtpkg.Generate(user.ID, user.Username)
	if err != nil {
		return nil, err
	}

	refreshToken, err := jwtpkg.GenerateRefreshToken(user.ID, user.Username)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(jwtpkg.GetAccessTokenExpireHour() * 3600),
		User:         user,
	}, nil
}

func (s *AuthService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	return string(bytes), err
}
