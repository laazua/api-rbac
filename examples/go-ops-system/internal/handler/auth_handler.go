// 运维系统 — 认证 Handler
// 转发登录/刷新请求到 api-rbac, 提供用户权限查询
package handler

import (
	"github.com/gin-gonic/gin"
	"go-ops-system/internal/middleware"
	"github.com/laazua/api-rbac/pkg/client"
	"github.com/laazua/api-rbac/pkg/errcode"
	"github.com/laazua/api-rbac/pkg/response"
)

// AuthHandler 认证相关 Handler
type AuthHandler struct {
	rbacClient *client.RBACClient
}

// NewAuthHandler 创建 AuthHandler
func NewAuthHandler(rbacClient *client.RBACClient) *AuthHandler {
	return &AuthHandler{rbacClient: rbacClient}
}

// Login 登录 — 转发到 api-rbac
// POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Account  string `json:"account" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}

	resp, err := h.rbacClient.Login(req.Account, req.Password)
	if err != nil {
		response.ErrorWithMsg(c, errcode.InternalError, "RBAC 服务不可达: "+err.Error())
		return
	}
	if resp.Code != 0 {
		response.ErrorWithMsg(c, resp.Code, resp.Message)
		return
	}

	response.Success(c, resp.Data)
}

// Refresh 刷新 Token — 转发到 api-rbac
// POST /api/auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}

	resp, err := h.rbacClient.Refresh(req.RefreshToken)
	if err != nil {
		response.ErrorWithMsg(c, errcode.InternalError, "RBAC 服务不可达: "+err.Error())
		return
	}
	if resp.Code != 0 {
		response.ErrorWithMsg(c, resp.Code, resp.Message)
		return
	}

	response.Success(c, resp.Data)
}

// GetPermissions 获取当前用户的全部权限 (供前端初始化)
// GET /api/auth/permissions
func (h *AuthHandler) GetPermissions(c *gin.Context) {
	token := middleware.GetToken(c)
	if token == "" {
		response.Error(c, errcode.Unauthorized)
		return
	}

	resp, err := h.rbacClient.GetMenu(token)
	if err != nil {
		response.ErrorWithMsg(c, errcode.InternalError, "RBAC 服务不可达: "+err.Error())
		return
	}
	if resp.Code != 0 {
		response.ErrorWithMsg(c, resp.Code, resp.Message)
		return
	}

	// 返回 permission map: {"server":["read","restart"], "deployment":["read","execute"], ...}
	response.Success(c, resp.Data.Permissions)
}
