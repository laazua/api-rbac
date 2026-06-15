package model

type Permission struct {
	BaseModel
	Name        string  `json:"name" gorm:"type:varchar(64);uniqueIndex;not null"`
	Resource    string  `json:"resource" gorm:"type:varchar(64);not null"`
	Action      string  `json:"action" gorm:"type:varchar(64);not null"`
	Description string  `json:"description" gorm:"type:varchar(255);default:''"`
	ModuleID    *uint   `json:"module_id" gorm:"type:bigint unsigned;default:null;index"`
	Module      *Module `json:"module,omitempty" gorm:"foreignKey:ModuleID"`
}

func (Permission) TableName() string {
	return "permissions"
}

// CreatePermissionRequest 创建权限请求
type CreatePermissionRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=64" example:"删除用户"`
	Resource    string `json:"resource" binding:"required,min=1,max=64" example:"user"`
	Action      string `json:"action" binding:"required,min=1,max=64" example:"delete"`
	Description string `json:"description" binding:"max=255" example:"允许删除其他用户账号"`
	ModuleID    *uint  `json:"module_id" example:"1"`
}

// UpdatePermissionRequest 更新权限请求
type UpdatePermissionRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=64" example:"删除任意用户"`
	Resource    string `json:"resource" binding:"required,min=1,max=64" example:"user"`
	Action      string `json:"action" binding:"required,min=1,max=64" example:"delete"`
	Description string `json:"description" binding:"max=255" example:"允许删除任意用户账号"`
	ModuleID    *uint  `json:"module_id" example:"1"`
}

// BatchCheckPermissionRequest 批量权限检查请求
type BatchCheckPermissionRequest struct {
	Permissions []BatchCheckPermissionItem `json:"permissions" binding:"required,min=1,max=50"`
}

// BatchCheckPermissionItem 批量检查中的单个项目
type BatchCheckPermissionItem struct {
	Resource string `json:"resource" binding:"required,min=1,max=64"`
	Action   string `json:"action" binding:"required,min=1,max=64"`
}

// ListPermissionRequest 权限列表查询
type ListPermissionRequest struct {
	Page     int    `form:"page" binding:"omitempty,min=1" example:"1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=1000" example:"10"`
	Keyword  string `form:"keyword" example:"删除"`
}
