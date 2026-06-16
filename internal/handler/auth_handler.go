package handler

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/laazua/api-rbac/internal/cache"
	"github.com/laazua/api-rbac/internal/model"
	"github.com/laazua/api-rbac/internal/service"
	"github.com/laazua/api-rbac/pkg/errcode"
	jwtpkg "github.com/laazua/api-rbac/pkg/jwt"
	"github.com/laazua/api-rbac/pkg/response"
)

// getUserID 安全地从 Gin Context 中提取 user_id (uint)
func getUserID(c *gin.Context) (uint, bool) {
	val, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	uid, ok := val.(uint)
	return uid, ok
}

type AuthHandler struct {
	authService   *service.AuthService
	permService   *service.PermissionCheckService
	moduleService *service.ModuleService
	blacklist     *cache.TokenBlacklist
}

func NewAuthHandler(authService *service.AuthService, permService *service.PermissionCheckService, moduleService *service.ModuleService, blacklist *cache.TokenBlacklist) *AuthHandler {
	return &AuthHandler{authService: authService, permService: permService, moduleService: moduleService, blacklist: blacklist}
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
	// 将当前 Access Token 加入黑名单
	jti, _ := c.Get("jti")
	if jtiStr, ok := jti.(string); ok && jtiStr != "" && h.blacklist != nil {
		_ = h.blacklist.Revoke(context.Background(), jtiStr, 2*time.Hour)
	}
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

	// 检查用户是否仍然存在且启用
	user, err := h.authService.GetByID(claims.UserID)
	if err != nil || user.Status == 0 {
		response.ErrorWithMsg(c, errcode.TokenInvalid, "用户不存在或已被禁用")
		return
	}

	// 撤销旧的 Refresh Token，防止重放攻击
	if h.blacklist != nil && claims.JTI != "" {
		refreshTTL := time.Duration(jwtpkg.GetRefreshExpireDay()*24) * time.Hour
		_ = h.blacklist.Revoke(context.Background(), claims.JTI, refreshTTL)
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
	uid, ok := getUserID(c)
	if !ok {
		response.Error(c, errcode.Unauthorized)
		return
	}

	perms, err := h.permService.GetUserPermissions(uid)
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
	uid, ok := getUserID(c)
	if !ok {
		response.Error(c, errcode.Unauthorized)
		return
	}

	var req model.CheckPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, err.Error())
		return
	}

	allowed, err := h.permService.CheckPermission(uid, req.Resource, req.Action)
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
	uid, ok := getUserID(c)
	if !ok {
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

	results, err := h.permService.BatchCheckPermission(uid, items)
	if err != nil {
		response.ErrorWithMsg(c, errcode.NotFound, err.Error())
		return
	}

	response.Success(c, gin.H{"results": results})
}

// Modules 获取用户可见模块列表
// @Summary      获取用户模块
// @Description  返回当前用户有权限访问的模块列表，每个模块包含该用户在此模块下的权限。用于前端生成模块卡片仪表盘。
// @Tags         认证
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response{data=object{modules=array}}  "查询成功"
// @Router       /auth/modules [get]
func (h *AuthHandler) Modules(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		response.Error(c, errcode.Unauthorized)
		return
	}

	// 获取用户所有权限
	perms, err := h.permService.GetUserPermissions(uid)
	if err != nil {
		response.ErrorWithMsg(c, errcode.NotFound, err.Error())
		return
	}

	// 判断是否为超级管理员（拥有 *:* 通配符权限）
	// 注意：不能用 perms["*"] 判断，因为 aggregatePermissions 会把 *:* 展开成具体资源
	isSuperAdmin, err := h.permService.HasWildcard(uid)
	if err != nil {
		response.ErrorWithMsg(c, errcode.InternalError, err.Error())
		return
	}

	var modules []model.Module
	if isSuperAdmin {
		// 超级管理员：返回所有启用的模块
		modules, err = h.moduleService.GetAllEnabledModules()
	} else {
		// 普通用户：通过角色→权限→模块推导
		modules, err = h.moduleService.GetUserModules(uid)
	}
	if err != nil {
		response.ErrorWithMsg(c, errcode.InternalError, err.Error())
		return
	}

	// 为每个模块附加该用户拥有的权限
	type moduleWithPerms struct {
		ID          uint                `json:"id"`
		Name        string              `json:"name"`
		Code        string              `json:"code"`
		Icon        string              `json:"icon"`
		Description string              `json:"description"`
		Sort        int                 `json:"sort"`
		Status      int                 `json:"status"`
		Url         string              `json:"url"`
		Permissions map[string][]string `json:"permissions"`
	}

	result := make([]moduleWithPerms, 0, len(modules))
	for _, m := range modules {
		result = append(result, moduleWithPerms{
			ID:          m.ID,
			Name:        m.Name,
			Code:        m.Code,
			Icon:        m.Icon,
			Description: m.Description,
			Sort:        m.Sort,
			Status:      m.Status,
			Url:         m.Url,
			Permissions: perms,
		})
	}

	response.Success(c, gin.H{"modules": result})
}
