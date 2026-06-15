-- 003_module.sql
-- 引入模块（Module）概念，用于对权限进行逻辑分组
-- 注意：GORM AutoMigrate 会自动处理表创建和字段添加，此文件仅作为参考

-- 1. 创建 modules 表
CREATE TABLE IF NOT EXISTS `modules` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(64) NOT NULL COMMENT '模块名称',
  `code` varchar(64) NOT NULL COMMENT '模块唯一标识',
  `icon` varchar(64) DEFAULT '' COMMENT '前端图标类名',
  `description` varchar(255) DEFAULT '' COMMENT '模块描述',
  `sort` int DEFAULT 0 COMMENT '排序序号',
  `status` tinyint DEFAULT 1 COMMENT '1=启用 0=禁用',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_code` (`code`),
  UNIQUE INDEX `idx_name` (`name`),
  INDEX `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 2. 为 permissions 表添加 module_id 字段
ALTER TABLE `permissions` ADD COLUMN `module_id` bigint unsigned DEFAULT NULL AFTER `description`;
ALTER TABLE `permissions` ADD INDEX `idx_module_id` (`module_id`);
