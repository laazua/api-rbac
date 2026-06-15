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
	permRepo *repository.PermissionRepo
	saRepo   *repository.ServiceAccountRepo
	cache    *cache.PermissionCache
}

func NewPermissionCheckService(userRepo *repository.UserRepo, permRepo *repository.PermissionRepo, saRepo *repository.ServiceAccountRepo, cache *cache.PermissionCache) *PermissionCheckService {
	return &PermissionCheckService{userRepo: userRepo, permRepo: permRepo, saRepo: saRepo, cache: cache}
}

// CheckPermission 检查用户是否拥有对指定资源的操作权限
func (s *PermissionCheckService) CheckPermission(userID uint, resource, action string) (bool, error) {
	// 提前检查是否为超级管理员，避免依赖硬编码展开列表
	isSuper, err := s.isWildcardAdmin(userID)
	if err != nil {
		return false, err
	}
	if isSuper {
		return true, nil
	}

	// 非超级管理员走正常的缓存+聚合流程
	perms, err := s.getUserPermissionsCached(userID)
	if err != nil {
		return false, err
	}
	return matchPermissionMap(perms, resource, action), nil
}

// GetUserPermissions 获取用户所有权限，按 resource → []action 聚合
// 用于前端生成动态菜单和控制按钮显隐
func (s *PermissionCheckService) GetUserPermissions(userID uint) (map[string][]string, error) {
	// 检查是否为超级管理员
	isSuper, err := s.isWildcardAdmin(userID)
	if err != nil {
		return nil, err
	}
	if isSuper {
		return s.getAllPermissionsFromDB()
	}
	return s.getUserPermissionsCached(userID)
}

// isWildcardAdmin 带缓存检查用户是否为超级管理员
func (s *PermissionCheckService) isWildcardAdmin(userID uint) (bool, error) {
	ctx := context.Background()

	// 尝试从缓存读取
	if s.cache != nil {
		found, isSuper, err := s.cache.HasWildcard(ctx, userID)
		if err == nil && found {
			return isSuper, nil
		}
	}

	// 缓存未命中，查库
	isSuper, err := s.HasWildcard(userID)
	if err != nil {
		return false, err
	}

	// 回填缓存
	if s.cache != nil {
		_ = s.cache.SetHasWildcard(ctx, userID, isSuper)
	}

	return isSuper, nil
}

// getAllPermissionsFromDB 从数据库获取系统中所有权限
func (s *PermissionCheckService) getAllPermissionsFromDB() (map[string][]string, error) {
	perms, _, err := s.permRepo.List(1, 10000, "")
	if err != nil {
		return nil, err
	}
	permMap := make(map[string][]string)
	for _, p := range perms {
		permMap[p.Resource] = appendIfMissing(permMap[p.Resource], p.Action)
	}
	return permMap, nil
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
			"user":            {"read", "create", "update", "delete"},
			"role":            {"read", "create", "update", "delete"},
			"permission":      {"read", "create", "update", "delete"},
			"module":          {"read", "create", "update", "delete"},
			"service_account": {"read", "create", "update", "delete"},
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
	// 超级管理员所有权限均为 true
	isSuper, err := s.isWildcardAdmin(userID)
	if err != nil {
		return nil, err
	}
	if isSuper {
		results := make(map[string]bool, len(items))
		for _, item := range items {
			key := item.Resource + ":" + item.Action
			results[key] = true
		}
		return results, nil
	}

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

// HasWildcard 判断用户是否拥有 *:* 超级管理员通配符权限
// 直接查库，不受 aggregatePermissions 的展开影响
func (s *PermissionCheckService) HasWildcard(userID uint) (bool, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return false, err
	}
	for _, role := range user.Roles {
		for _, perm := range role.Permissions {
			if perm.Resource == "*" && perm.Action == "*" {
				return true, nil
			}
		}
	}
	return false, nil
}

// CheckServiceAccountPermission 检查服务账号是否拥有指定权限
// 向后兼容：无角色的服务账号保持全放行行为
func (s *PermissionCheckService) CheckServiceAccountPermission(saID uint, resource, action string) (bool, error) {
	sa, err := s.saRepo.FindByIDWithRoles(saID)
	if err != nil {
		return false, err
	}

	// 向后兼容：无角色的服务账号保持旧行为（全部放行）
	if len(sa.Roles) == 0 {
		return true, nil
	}

	permMap := aggregatePermissions(sa.Roles)
	return matchPermissionMap(permMap, resource, action), nil
}
