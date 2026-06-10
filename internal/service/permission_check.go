package service

import (
	"errors"

	"gorm.io/gorm"

	"api-rbac/internal/model"
	"api-rbac/internal/repository"
)

type PermissionCheckService struct {
	userRepo *repository.UserRepo
}

func NewPermissionCheckService(userRepo *repository.UserRepo) *PermissionCheckService {
	return &PermissionCheckService{userRepo: userRepo}
}

// CheckPermission 检查用户是否拥有对指定资源的操作权限
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

	return matchPermission(user.Roles, resource, action), nil
}

// GetUserPermissions 获取用户所有权限，按 resource → []action 聚合
// 用于前端生成动态菜单和控制按钮显隐
func (s *PermissionCheckService) GetUserPermissions(userID uint) (map[string][]string, error) {
	permMap := make(map[string][]string)

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return permMap, errors.New("用户不存在")
		}
		return permMap, err
	}

	for _, role := range user.Roles {
		for _, perm := range role.Permissions {
			permMap[perm.Resource] = appendIfMissing(permMap[perm.Resource], perm.Action)
		}
	}

	// 通配符 *:* 拥有全部权限
	if hasWildcard(permMap) {
		return map[string][]string{
			"user":       {"read", "create", "update", "delete"},
			"role":       {"read", "create", "update", "delete"},
			"permission": {"read", "create", "update", "delete"},
		}, nil
	}

	return permMap, nil
}

func matchPermission(roles []model.Role, resource, action string) bool {
	for _, role := range roles {
		for _, perm := range role.Permissions {
			if perm.Resource == "*" && perm.Action == "*" {
				return true
			}
			if perm.Resource == resource && perm.Action == "*" {
				return true
			}
			if perm.Resource == "*" && perm.Action == action {
				return true
			}
			if perm.Resource == resource && perm.Action == action {
				return true
			}
		}
	}
	return false
}

func hasWildcard(m map[string][]string) bool {
	actions, ok := m["*"]
	if !ok {
		return false
	}
	for _, a := range actions {
		if a == "*" {
			return true
		}
	}
	return false
}

func appendIfMissing(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

// 需要 import model 在函数里用了 model.Role
// 实际该文件 import 里没有 model，需要加上
