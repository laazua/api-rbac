package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// RBACClient RBAC 服务 HTTP 客户端，供业务系统集成使用
// 支持任意编程语言通过 HTTP 协议集成，此为 Go 语言官方 SDK
type RBACClient struct {
	baseURL string
	client  *http.Client
}

// NewRBACClient 创建 RBAC 客户端
// baseURL: RBAC 服务的地址，如 "http://localhost:8087/api/v1"
func NewRBACClient(baseURL string) *RBACClient {
	return &RBACClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Login 用户登录，返回 JWT Token
func (c *RBACClient) Login(username, password string) (*LoginResponse, error) {
	body := map[string]string{
		"username": username,
		"password": password,
	}

	var resp LoginResponse
	if err := c.post("/auth/login", nil, body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Verify 验证 Token 是否有效
func (c *RBACClient) Verify(token string) (*VerifyResponse, error) {
	headers := map[string]string{"Authorization": "Bearer " + token}

	var resp VerifyResponse
	if err := c.post("/auth/verify", headers, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CheckPermission 检查用户是否拥有指定权限
func (c *RBACClient) CheckPermission(token, resource, action string) (*PermissionCheckResponse, error) {
	headers := map[string]string{"Authorization": "Bearer " + token}
	body := map[string]string{
		"resource": resource,
		"action":   action,
	}

	var resp PermissionCheckResponse
	if err := c.post("/auth/check", headers, body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// post 通用 POST 请求
func (c *RBACClient) post(path string, headers map[string]string, body any, result any) error {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("序列化请求体失败: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(http.MethodPost, url, bodyReader)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	return nil
}

// --- 响应结构体 ---

type LoginResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Token    string `json:"token"`
		UserID   uint   `json:"user_id"`
		Username string `json:"username"`
	} `json:"data"`
}

type VerifyResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		UserID   uint   `json:"user_id"`
		Username string `json:"username"`
	} `json:"data"`
}

type PermissionCheckResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Allowed bool `json:"allowed"`
	} `json:"data"`
}
