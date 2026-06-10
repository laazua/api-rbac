package handler

import (
	"github.com/gin-gonic/gin"

	"api-rbac/internal/model"
	"api-rbac/internal/service"
	"api-rbac/pkg/errcode"
	"api-rbac/pkg/response"
)

type AuthHandler struct {
	authService  *service.AuthService
	permService  *service.PermissionCheckService
}

func NewAuthHandler(authService *service.AuthService, permService *service.PermissionCheckService) *AuthHandler {
	return &AuthHandler{authService: authService, permService: permService}
}

// Login 用户登录
// @Summary      用户登录
// @Description  使用用户名或邮箱登录，返回 JWT Token
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        request body model.LoginRequest true "登录参数"
// @Success      200  {object}  response.Response{data=object{token=string,user_id=int,username=string}}  "登录成功"
// @Failure      400  {object}  response.Response  "参数错误"
// @Failure      401  {object}  response.Response  "密码错误或用户禁用"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, err.Error())
		return
	}

	token, user, err := h.authService.Login(&req)
	if err != nil {
		response.ErrorWithMsg(c, errcode.PasswordWrong, err.Error())
		return
	}

	response.Success(c, gin.H{
		"token":    token,
		"user_id":  user.ID,
		"username": user.Username,
	})
}

// Logout 用户登出
// @Summary      用户登出
// @Description  无状态 JWT，客户端丢弃 Token 即完成登出
// @Tags         认证
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response  "登出成功"
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	// 无状态 JWT，客户端只需丢弃 Token 即可
	response.Success(c, nil)
}

// Verify 验证Token
// @Summary      验证Token
// @Description  验证 JWT Token 是否有效，返回用户基本信息。业务系统在收到请求时可调用此接口确认 Token 有效性。
// @Tags         认证
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response{data=object{user_id=int,username=string}}  "Token有效"
// @Failure      401  {object}  response.Response  "Token无效或已过期"
// @Router       /auth/verify [post]
func (h *AuthHandler) Verify(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, errcode.Unauthorized)
		return
	}
	username, _ := c.Get("username")

	response.Success(c, gin.H{
		"user_id":  userID,
		"username": username,
	})
}

// Check 检查权限
// @Summary      检查权限
// @Description  检查当前用户是否拥有对指定资源的指定操作权限。业务系统在执行敏感操作前调用此接口做鉴权。
// @Tags         认证
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request body model.CheckPermissionRequest true "权限检查参数"
// @Success      200  {object}  response.Response{data=object{allowed=bool}}  "有权限"
// @Failure      403  {object}  response.Response{data=object{allowed=bool}}  "无权限"
// @Router       /auth/check [post]
func (h *AuthHandler) Check(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, errcode.Unauthorized)
		return
	}

	var req model.CheckPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, err.Error())
		return
	}

	allowed, err := h.permService.CheckPermission(userID.(uint), req.Resource, req.Action)
	if err != nil {
		response.ErrorWithMsg(c, errcode.NotFound, err.Error())
		return
	}

	if !allowed {
		response.ErrorWithData(c, errcode.Forbidden, gin.H{"allowed": false})
		return
	}

	response.Success(c, gin.H{"allowed": true})
}
