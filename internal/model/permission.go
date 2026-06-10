package model

type Permission struct {
	BaseModel
	Name        string `json:"name" gorm:"type:varchar(64);uniqueIndex;not null"`
	Resource    string `json:"resource" gorm:"type:varchar(64);not null"`
	Action      string `json:"action" gorm:"type:varchar(64);not null"`
	Description string `json:"description" gorm:"type:varchar(255);default:''"`
}

func (Permission) TableName() string {
	return "permissions"
}

// CreatePermissionRequest 创建权限请求
type CreatePermissionRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=64" example:"删除用户"`
	Resource    string `json:"resource" binding:"required,min=2,max=64" example:"user"`
	Action      string `json:"action" binding:"required,min=2,max=64" example:"delete"`
	Description string `json:"description" binding:"max=255" example:"允许删除其他用户账号"`
}

// UpdatePermissionRequest 更新权限请求
type UpdatePermissionRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=64" example:"删除任意用户"`
	Resource    string `json:"resource" binding:"required,min=2,max=64" example:"user"`
	Action      string `json:"action" binding:"required,min=2,max=64" example:"delete"`
	Description string `json:"description" binding:"max=255" example:"允许删除任意用户账号"`
}

// ListPermissionRequest 权限列表查询
type ListPermissionRequest struct {
	Page     int    `form:"page" binding:"omitempty,min=1" example:"1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100" example:"10"`
	Keyword  string `form:"keyword" example:"删除"`
}
