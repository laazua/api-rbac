package model

// ServiceAccount 服务账号，用于服务间 API 调用认证
type ServiceAccount struct {
	BaseModel
	Name        string `json:"name" gorm:"type:varchar(64);uniqueIndex;not null"`
	ApiKeyHash  string `json:"-" gorm:"type:varchar(128);not null"`  // SHA256 哈希，不暴露到 JSON
	Status      int    `json:"status" gorm:"type:tinyint;default:1"` // 1=启用, 0=禁用
	Description string `json:"description" gorm:"type:varchar(255)"`
}

func (ServiceAccount) TableName() string {
	return "service_accounts"
}

// CreateServiceAccountRequest 创建服务账号请求
type CreateServiceAccountRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=64"`
	Description string `json:"description" binding:"max=255"`
}

// UpdateServiceAccountRequest 更新服务账号请求
type UpdateServiceAccountRequest struct {
	Description string `json:"description" binding:"max=255"`
	Status      *int   `json:"status"`
}

// ListServiceAccountRequest 列表查询请求
type ListServiceAccountRequest struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"page_size" form:"page_size"`
	Keyword  string `json:"keyword" form:"keyword"`
}
