package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// TokenBlacklist Redis 令牌黑名单
// 用于存储已撤销的 JWT Token (jti)，Redis 不可用时降级为跳过检查
type TokenBlacklist struct {
	rdb *redis.Client
}

// NewTokenBlacklist 创建令牌黑名单实例
// rdb 可以为 nil，此时所有操作降级为 no-op
func NewTokenBlacklist(rdb *redis.Client) *TokenBlacklist {
	return &TokenBlacklist{rdb: rdb}
}

// Revoke 将 jti 加入黑名单，TTL 为令牌剩余有效期
// Redis 不可用时静默返回 nil
func (b *TokenBlacklist) Revoke(ctx context.Context, jti string, ttl time.Duration) error {
	if b.rdb == nil {
		return nil
	}
	return b.rdb.Set(ctx, blacklistKey(jti), "1", ttl).Err()
}

// IsRevoked 检查 jti 是否在黑名单中
// Redis 不可用时返回 false（降级为不检查，可用性优先）
func (b *TokenBlacklist) IsRevoked(ctx context.Context, jti string) bool {
	if b.rdb == nil {
		return false
	}
	n, err := b.rdb.Exists(ctx, blacklistKey(jti)).Result()
	if err != nil {
		return false
	}
	return n > 0
}

func blacklistKey(jti string) string {
	return fmt.Sprintf("rbac:blacklist:%s", jti)
}
