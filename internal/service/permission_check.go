package service

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/laazua/api-rbac/internal/cache"
	"github.com/laazua/api-rbac/internal/model"
	"github.com/laazua/api-rbac/internal/repository"
)

type PermissionCheckService struct {
	userRepo *repository.UserRepo
	cache    *cache.PermissionCache
}

func NewPermissionCheckService(userRepo *repository.UserRepo, cache *cache.PermissionCache) *PermissionCheckService {
	return &PermissionCheckService{userRepo: userRepo, cache: cache}
}

// CheckPermission 检查用户是否拥有对指定资源的操作权限
func (s *PermissionCheckService) CheckPermission(userID uint, resource, action string) (bool, error) {
	// 优先从缓存获取权限
	perms, err := s.getUserPermissionsCached(userID)
	if err != nil {
		return false, err
	}
	return matchPermissionMap(perms, resource, action), nil
}

// GetUserPermissions 获取用户所有权限，按 resource → []action 聚合
// 用于前端生成动态菜单和控制按钮显隐
func (s *PermissionCheckService) GetUserPermissions(userID uint) (map[string][]string, error) {
	return s.getUserPermissionsCached(userID)
}

// getUserPermissionsCached 优先读缓存，miss 时查库并回填缓存
func (s *PermissionCheckService) getUserPermissionsCached(userID uint) (map[string][]string, error) {
	ctx := context.Background()

	// 尝试从缓存读取
	if s.cache != nil {
		perms, err := s.cache.GetUserPermissions(ctx, userID)
		if err == nil && perms != nil {
			return perms, nil
		}
	}

	// 缓存未命中，查询数据库
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}

	if user.Status == 0 {
		return nil, errors.New("用户已被禁用")
	}

	permMap := aggregatePermissions(user.Roles)

	// 回填缓存
	if s.cache != nil {
		_ = s.cache.SetUserPermissions(ctx, userID, permMap)
	}

	return permMap, nil
}

// aggregatePermissions 从角色列表中聚合权限
func aggregatePermissions(roles []model.Role) map[string][]string {
	permMap := make(map[string][]string)

	for _, role := range roles {
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
		}
	}

	return permMap
}

// matchPermissionMap 在权限 map 中匹配
func matchPermissionMap(permMap map[string][]string, resource, action string) bool {
	// 检查 *:*
	if actions, ok := permMap["*"]; ok {
		for _, a := range actions {
			if a == "*" || a == action {
				return true
			}
		}
	}

	// 检查 resource:*
	if actions, ok := permMap[resource]; ok {
		for _, a := range actions {
			if a == "*" || a == action {
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

// BatchCheckItem 批量检查的单个项目
type BatchCheckItem struct {
	Resource string `json:"resource" binding:"required"`
	Action   string `json:"action" binding:"required"`
}

// BatchCheckPermission 批量检查权限，返回 key "resource:action" → bool
func (s *PermissionCheckService) BatchCheckPermission(userID uint, items []BatchCheckItem) (map[string]bool, error) {
	perms, err := s.getUserPermissionsCached(userID)
	if err != nil {
		return nil, err
	}

	results := make(map[string]bool, len(items))
	for _, item := range items {
		key := item.Resource + ":" + item.Action
		results[key] = matchPermissionMap(perms, item.Resource, item.Action)
	}
	return results, nil
}
