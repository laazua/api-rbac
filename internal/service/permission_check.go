package service

import (
	"errors"

	"gorm.io/gorm"

	"api-rbac/internal/repository"
)

type PermissionCheckService struct {
	userRepo *repository.UserRepo
}

func NewPermissionCheckService(userRepo *repository.UserRepo) *PermissionCheckService {
	return &PermissionCheckService{userRepo: userRepo}
}

// CheckPermission 检查用户是否拥有对指定资源的操作权限
// 遍历用户的所有角色 → 角色的所有权限，匹配 resource 和 action
func (s *PermissionCheckService) CheckPermission(userID uint, resource, action string) (bool, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, errors.New("用户不存在")
		}
		return false, err
	}

	if user.Status == 0 {
		return false, errors.New("用户已被禁用")
	}

	for _, role := range user.Roles {
		for _, perm := range role.Permissions {
			if perm.Resource == resource && perm.Action == action {
				return true, nil
			}
			// 通配符匹配：resource="*" 或 action="*" 表示全部权限
			if perm.Resource == "*" && perm.Action == "*" {
				return true, nil
			}
			if perm.Resource == resource && perm.Action == "*" {
				return true, nil
			}
			if perm.Resource == "*" && perm.Action == action {
				return true, nil
			}
		}
	}

	return false, nil
}
