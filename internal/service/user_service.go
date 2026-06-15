package service

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/laazua/api-rbac/internal/cache"
	"github.com/laazua/api-rbac/internal/model"
	"github.com/laazua/api-rbac/internal/repository"
)

type UserService struct {
	userRepo *repository.UserRepo
	roleRepo *repository.RoleRepo
	cache    *cache.PermissionCache
}

func NewUserService(userRepo *repository.UserRepo, roleRepo *repository.RoleRepo, cache *cache.PermissionCache) *UserService {
	return &UserService{userRepo: userRepo, roleRepo: roleRepo, cache: cache}
}

func (s *UserService) Create(req *model.CreateUserRequest) (*model.User, error) {
	// 检查是否存在同名用户（包括已软删除的）
	existing, err := s.userRepo.FindByUsernameIncludingDeleted(req.Username)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing != nil {
		if existing.DeletedAt.Valid {
			// 存在同名已软删除的用户，恢复并更新该记录
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			if err != nil {
				return nil, err
			}
			if err := s.userRepo.Restore(existing.ID); err != nil {
				return nil, err
			}
			if err := s.userRepo.UpdatePassword(existing.ID, string(hashedPassword)); err != nil {
				return nil, err
			}
			existing.Email = req.Email
			existing.Status = 1
			if err := s.userRepo.Update(existing); err != nil {
				return nil, err
			}
			// 重新加载完整数据返回
			return s.userRepo.FindByID(existing.ID)
		}
		return nil, errors.New("用户名已存在")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username: req.Username,
		Password: string(hashedPassword),
		Email:    req.Email,
		Status:   1,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) GetByID(id uint) (*model.User, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}
	return user, nil
}

func (s *UserService) List(req *model.ListUserRequest) ([]model.User, int64, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	return s.userRepo.List(req.Page, req.PageSize, req.Keyword)
}

func (s *UserService) Update(id uint, req *model.UpdateUserRequest) error {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("用户不存在")
		}
		return err
	}

	user.Email = req.Email
	if req.Status != nil {
		user.Status = *req.Status
	}

	return s.userRepo.Update(user)
}

func (s *UserService) Delete(id uint) error {
	_, err := s.userRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("用户不存在")
		}
		return err
	}
	return s.userRepo.Delete(id)
}

func (s *UserService) ChangePassword(id uint, req *model.ChangePasswordRequest) error {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("用户不存在")
		}
		return err
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		return errors.New("旧密码错误")
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.userRepo.UpdatePassword(id, string(hashedPassword))
}

func (s *UserService) AssignRoles(id uint, roleIDs []uint) error {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("用户不存在")
		}
		return err
	}

	roles, err := s.roleRepo.FindByIDs(roleIDs)
	if err != nil {
		return err
	}

	if err := s.userRepo.AssignRoles(user, roles); err != nil {
		return err
	}

	// 角色变更后失效权限缓存
	s.invalidateCache(id)
	return nil
}

func (s *UserService) RemoveRole(id, roleID uint) error {
	_, err := s.userRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("用户不存在")
		}
		return err
	}

	if err := s.userRepo.RemoveRole(id, roleID); err != nil {
		return err
	}

	// 角色变更后失效权限缓存
	s.invalidateCache(id)
	return nil
}

func (s *UserService) invalidateCache(userID uint) {
	if s.cache != nil {
		_ = s.cache.InvalidateUser(context.Background(), userID)
	}
}
