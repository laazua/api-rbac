package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// PermissionCache 用户权限 Redis 缓存
// 缓存 key: rbac:user:{userID}:perms → map[string][]string (JSON)
type PermissionCache struct {
	rdb *redis.Client
	ttl time.Duration
}

// NewPermissionCache 创建权限缓存实例
func NewPermissionCache(rdb *redis.Client, ttl time.Duration) *PermissionCache {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &PermissionCache{rdb: rdb, ttl: ttl}
}

// GetUserPermissions 从缓存获取用户权限，缓存未命中返回 nil
func (c *PermissionCache) GetUserPermissions(ctx context.Context, userID uint) (map[string][]string, error) {
	key := cacheKey(userID)
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 缓存未命中
		}
		return nil, fmt.Errorf("redis get failed: %w", err)
	}

	var perms map[string][]string
	if err := json.Unmarshal(data, &perms); err != nil {
		return nil, fmt.Errorf("unmarshal cached perms: %w", err)
	}
	return perms, nil
}

// SetUserPermissions 将用户权限写入缓存
func (c *PermissionCache) SetUserPermissions(ctx context.Context, userID uint, perms map[string][]string) error {
	data, err := json.Marshal(perms)
	if err != nil {
		return fmt.Errorf("marshal perms: %w", err)
	}
	return c.rdb.Set(ctx, cacheKey(userID), data, c.ttl).Err()
}

// InvalidateUser 清除指定用户的权限缓存
func (c *PermissionCache) InvalidateUser(ctx context.Context, userID uint) error {
	return c.rdb.Del(ctx, cacheKey(userID)).Err()
}

// InvalidateUsers 批量清除多个用户的权限缓存
func (c *PermissionCache) InvalidateUsers(ctx context.Context, userIDs []uint) error {
	if len(userIDs) == 0 {
		return nil
	}
	keys := make([]string, len(userIDs))
	for i, id := range userIDs {
		keys[i] = cacheKey(id)
	}
	return c.rdb.Del(ctx, keys...).Err()
}

func cacheKey(userID uint) string {
	return fmt.Sprintf("rbac:user:%d:perms", userID)
}
