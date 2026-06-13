package model

// IntrospectRequest Token 自省请求
type IntrospectRequest struct {
	Token    string `json:"token" binding:"required"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// IntrospectResponse Token 自省响应
type IntrospectResponse struct {
	Active   bool   `json:"active"`
	UserID   uint   `json:"user_id,omitempty"`
	Username string `json:"username,omitempty"`
}
