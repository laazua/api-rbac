package service

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"api-rbac/internal/model"
	"api-rbac/internal/repository"
	jwtpkg "api-rbac/pkg/jwt"
)

type AuthService struct {
	userRepo *repository.UserRepo
}

func NewAuthService(userRepo *repository.UserRepo) *AuthService {
	return &AuthService{userRepo: userRepo}
}

func (s *AuthService) Login(req *model.LoginRequest) (string, *model.User, error) {
	// 优先按用户名查找，若未找到则尝试按邮箱查找
	user, err := s.userRepo.FindByUsername(req.Account)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 尝试邮箱登录
			user, err = s.userRepo.FindByEmail(req.Account)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return "", nil, errors.New("用户名或密码错误")
				}
				return "", nil, err
			}
		} else {
			return "", nil, err
		}
	}

	if user.Status == 0 {
		return "", nil, errors.New("用户已被禁用")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return "", nil, errors.New("用户名或密码错误")
	}

	token, err := jwtpkg.Generate(user.ID, user.Username)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil
}

func (s *AuthService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}
