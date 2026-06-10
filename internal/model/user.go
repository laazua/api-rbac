package model

type User struct {
	BaseModel
	Username string `json:"username" gorm:"type:varchar(64);uniqueIndex;not null"`
	Password string `json:"-" gorm:"type:varchar(255);not null"`
	Email    string `json:"email" gorm:"type:varchar(128);default:''"`
	Status   int    `json:"status" gorm:"type:tinyint;default:1;comment:1=启用 0=禁用"`
	Roles    []Role `json:"roles,omitempty" gorm:"many2many:user_roles;"`
}

func (User) TableName() string {
	return "users"
}

// LoginRequest 登录请求
type LoginRequest struct {
	Account  string `json:"account" binding:"required,min=2,max=128" example:"admin"`
	Password string `json:"password" binding:"required,max=128" example:"admin123"`
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=2,max=64" example:"zhangsan"`
	Password string `json:"password" binding:"required,min=6,max=128" example:"123456"`
	Email    string `json:"email" binding:"omitempty,email,max=128" example:"zhangsan@example.com"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Email  string `json:"email" binding:"omitempty,email,max=128" example:"newemail@example.com"`
	Status *int   `json:"status" binding:"omitempty,oneof=0 1"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required,max=128" example:"old123456"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=128" example:"new123456"`
}

// AssignRolesRequest 为用户分配角色请求
type AssignRolesRequest struct {
	RoleIDs []uint `json:"role_ids" binding:"required"` // 例: [1, 2]
}

// CheckPermissionRequest 权限检查请求
type CheckPermissionRequest struct {
	Resource string `json:"resource" binding:"required,min=1,max=64" example:"user"`
	Action   string `json:"action" binding:"required,min=1,max=64" example:"delete"`
}

// ListUserRequest 用户列表查询
type ListUserRequest struct {
	Page     int    `form:"page" binding:"omitempty,min=1" example:"1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100" example:"10"`
	Keyword  string `form:"keyword" example:"admin"`
}
