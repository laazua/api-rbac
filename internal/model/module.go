package model

// Module 功能模块，用于对权限进行逻辑分组
type Module struct {
	BaseModel
	Name        string       `json:"name" gorm:"type:varchar(64);uniqueIndex;not null"`
	Code        string       `json:"code" gorm:"type:varchar(64);uniqueIndex;not null"`
	Icon        string       `json:"icon" gorm:"type:varchar(64);default:''"`
	Description string       `json:"description" gorm:"type:varchar(255);default:''"`
	Sort        int          `json:"sort" gorm:"type:int;default:0"`
	Status      int          `json:"status" gorm:"type:tinyint;default:1;comment:1=启用 0=禁用"`
	Url         string       `json:"url" gorm:"type:varchar(512);default:'';comment:模块前端入口地址，空则使用内置路由"`
	Permissions []Permission `json:"permissions,omitempty" gorm:"foreignKey:ModuleID"`
}

func (Module) TableName() string {
	return "modules"
}

// CreateModuleRequest 创建模块请求
type CreateModuleRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=64" example:"系统管理"`
	Code        string `json:"code" binding:"required,min=2,max=64" example:"system_mgmt"`
	Icon        string `json:"icon" binding:"max=64" example:"el-icon-setting"`
	Description string `json:"description" binding:"max=255" example:"系统配置与权限管理"`
	Sort        int    `json:"sort" example:"1"`
	Url         string `json:"url" binding:"max=512" example:"http://localhost:8090"`
}

// UpdateModuleRequest 更新模块请求
type UpdateModuleRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=64" example:"系统管理"`
	Code        string `json:"code" binding:"required,min=2,max=64" example:"system_mgmt"`
	Icon        string `json:"icon" binding:"max=64" example:"el-icon-setting"`
	Description string `json:"description" binding:"max=255" example:"系统配置与权限管理"`
	Sort        int    `json:"sort" example:"1"`
	Status      *int   `json:"status" binding:"omitempty,oneof=0 1"`
	Url         string `json:"url" binding:"max=512" example:"http://localhost:8090"`
}

// ListModuleRequest 模块列表查询
type ListModuleRequest struct {
	Page     int    `form:"page" binding:"omitempty,min=1" example:"1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=1000" example:"10"`
	Keyword  string `form:"keyword" example:"系统"`
}
