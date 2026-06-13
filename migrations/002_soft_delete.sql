-- ==========================================
-- 软删除迁移 (参考 SQL)
-- GORM AutoMigrate 会自动处理，此文件仅为参考
-- ==========================================

-- 为各表增加 deleted_at 字段（如果使用手动迁移）
ALTER TABLE users ADD COLUMN deleted_at DATETIME(3) NULL DEFAULT NULL;
ALTER TABLE roles ADD COLUMN deleted_at DATETIME(3) NULL DEFAULT NULL;
ALTER TABLE permissions ADD COLUMN deleted_at DATETIME(3) NULL DEFAULT NULL;
ALTER TABLE service_accounts ADD COLUMN deleted_at DATETIME(3) NULL DEFAULT NULL;

-- 添加索引以加速查询（GORM 自动创建）
CREATE INDEX idx_users_deleted_at ON users(deleted_at);
CREATE INDEX idx_roles_deleted_at ON roles(deleted_at);
CREATE INDEX idx_permissions_deleted_at ON permissions(deleted_at);
CREATE INDEX idx_service_accounts_deleted_at ON service_accounts(deleted_at);

-- 恢复已删除记录示例:
-- UPDATE users SET deleted_at = NULL WHERE id = 1;

-- 查询已删除记录:
-- SELECT * FROM users WHERE deleted_at IS NOT NULL;

-- 永久删除（GORM: db.Unscoped().Delete(&user)）:
-- DELETE FROM users WHERE deleted_at IS NOT NULL AND id = 1;
