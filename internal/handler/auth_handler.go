package handler

import (
	"github.com/gin-gonic/gin"

	jwtpkg "github.com/laazua/api-rbac/pkg/jwt"
	"github.com/laazua/api-rbac/internal/model"
	"github.com/laazua/api-rbac/internal/service"
	"github.com/laazua/api-rbac/pkg/errcode"
	"github.com/laazua/api-rbac/pkg/response"
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

	result, err := h.authService.Login(&req)
	if err != nil {
		response.ErrorWithMsg(c, errcode.PasswordWrong, err.Error())
		return
	}

	response.Success(c, gin.H{
		"token":         result.Token,
		"refresh_token": result.RefreshToken,
		"expires_in":    result.ExpiresIn,
		"user_id":       result.User.ID,
		"username":      result.User.Username,
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

// Refresh 刷新 Token
// @Summary      刷新 Token
// @Description  使用 Refresh Token 获取新的 Access Token 和 Refresh Token
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        request body model.RefreshTokenRequest true "刷新参数"
// @Success      200  {object}  response.Response{data=object{token=string,refresh_token=string,expires_in=int}}  "刷新成功"
// @Failure      401  {object}  response.Response  "Refresh Token 无效或过期"
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req model.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, err.Error())
		return
	}

	claims, err := jwtpkg.ParseRefreshToken(req.RefreshToken)
	if err != nil {
		response.ErrorWithMsg(c, errcode.TokenInvalid, "刷新令牌无效或已过期")
		return
	}

	// 生成新的 Token 对
	newToken, err := jwtpkg.Generate(claims.UserID, claims.Username)
	if err != nil {
		response.Error(c, errcode.InternalError)
		return
	}

	newRefreshToken, err := jwtpkg.GenerateRefreshToken(claims.UserID, claims.Username)
	if err != nil {
		response.Error(c, errcode.InternalError)
		return
	}

	response.Success(c, gin.H{
		"token":         newToken,
		"refresh_token": newRefreshToken,
		"expires_in":    int64(jwtpkg.GetAccessTokenExpireHour() * 3600),
	})
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

// Menu 获取用户菜单
// @Summary      获取用户菜单
// @Description  返回当前用户拥有的权限列表，用于前端动态生成菜单和控制按钮显隐
// @Tags         认证
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response{data=object{permissions=object}}  "查询成功"
// @Router       /auth/menu [get]
func (h *AuthHandler) Menu(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, errcode.Unauthorized)
		return
	}

	perms, err := h.permService.GetUserPermissions(userID.(uint))
	if err != nil {
		response.ErrorWithMsg(c, errcode.NotFound, err.Error())
		return
	}

	response.Success(c, gin.H{"permissions": perms})
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

// Introspect Token 自省，外部服务一次调用即可完成 Token 验证 + 权限检查
// @Summary      Token 自省
// @Description  验证 Token 有效性，可选同时检查指定资源和操作的权限。供外部业务服务集成使用。
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        request body model.IntrospectRequest true "自省参数"
// @Success      200  {object}  response.Response{data=model.IntrospectResponse}  "查询成功"
// @Router       /auth/introspect [post]
func (h *AuthHandler) Introspect(c *gin.Context) {
	var req model.IntrospectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, err.Error())
		return
	}

	// 解析 Token
	claims, err := jwtpkg.Parse(req.Token)
	if err != nil {
		// Token 无效或过期，返回 inactive
		response.Success(c, model.IntrospectResponse{Active: false})
		return
	}

	resp := model.IntrospectResponse{
		Active:   true,
		UserID:   claims.UserID,
		Username: claims.Username,
	}

	// 如果请求中包含 resource 和 action，则同时检查权限
	if req.Resource != "" && req.Action != "" {
		allowed, err := h.permService.CheckPermission(claims.UserID, req.Resource, req.Action)
		if err != nil || !allowed {
			resp.Active = false
		}
	}

	response.Success(c, resp)
}

// BatchCheck 批量权限检查
// @Summary      批量权限检查
// @Description  一次性检查多个权限，返回每个 permission 对应的结果
// @Tags         认证
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request body model.BatchCheckPermissionRequest true "批量检查参数"
// @Success      200  {object}  response.Response{data=object{results=object}}  "检查完成"
// @Router       /auth/batch-check [post]
func (h *AuthHandler) BatchCheck(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, errcode.Unauthorized)
		return
	}

	var req model.BatchCheckPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, err.Error())
		return
	}

	items := make([]service.BatchCheckItem, len(req.Permissions))
	for i, p := range req.Permissions {
		items[i] = service.BatchCheckItem{Resource: p.Resource, Action: p.Action}
	}

	results, err := h.permService.BatchCheckPermission(userID.(uint), items)
	if err != nil {
		response.ErrorWithMsg(c, errcode.NotFound, err.Error())
		return
	}

	response.Success(c, gin.H{"results": results})
}
