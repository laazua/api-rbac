package model

type Role struct {
	BaseModel
	Name        string       `json:"name" gorm:"type:varchar(64);uniqueIndex;not null"`
	Description string       `json:"description" gorm:"type:varchar(255);default:''"`
	Permissions []Permission `json:"permissions,omitempty" gorm:"many2many:role_permissions;"`
	Modules     []Module     `json:"modules,omitempty" gorm:"many2many:role_modules;"`
}

func (Role) TableName() string {
	return "roles"
}

// CreateRoleRequest 创建角色请求
type CreateRoleRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=64" example:"编辑员"`
	Description string `json:"description" binding:"max=255" example:"负责内容编辑的角色"`
}

// UpdateRoleRequest 更新角色请求
type UpdateRoleRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=64" example:"高级编辑员"`
	Description string `json:"description" binding:"max=255" example:"负责高级内容编辑的角色"`
}

// AssignPermissionsRequest 为角色分配权限请求
type AssignPermissionsRequest struct {
	PermissionIDs []uint `json:"permission_ids"` // 例: [1, 2, 3]，传空数组则清空权限
}

// AssignModulesRequest 为角色分配模块请求
type AssignModulesRequest struct {
	ModuleIDs []uint `json:"module_ids"` // 例: [1, 2]，传空数组则清空模块
}

// ListRoleRequest 角色列表查询
type ListRoleRequest struct {
	Page     int    `form:"page" binding:"omitempty,min=1" example:"1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=1000" example:"10"`
	Keyword  string `form:"keyword" example:"管理员"`
}
