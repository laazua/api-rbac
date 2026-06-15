package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/laazua/api-rbac/internal/model"
	"github.com/laazua/api-rbac/internal/service"
	"github.com/laazua/api-rbac/pkg/errcode"
	"github.com/laazua/api-rbac/pkg/response"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// Create 创建用户
// @Summary      创建用户
// @Description  新增一个用户
// @Tags         用户管理
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request body model.CreateUserRequest true "创建用户参数"
// @Success      200  {object}  response.Response{data=model.User}  "创建成功"
// @Failure      400  {object}  response.Response  "参数错误或用户名已存在"
// @Router       /users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req model.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, err.Error())
		return
	}

	user, err := h.userService.Create(&req)
	if err != nil {
		response.ErrorWithMsg(c, errcode.AlreadyExists, err.Error())
		return
	}

	response.Success(c, user)
}

// GetByID 获取用户详情
// @Summary      获取用户详情
// @Description  根据 ID 获取用户信息，包含关联的角色和权限
// @Tags         用户管理
// @Security     BearerAuth
// @Produce      json
// @Param        id path int true "用户ID"
// @Success      200  {object}  response.Response{data=model.User}  "查询成功"
// @Failure      404  {object}  response.Response  "用户不存在"
// @Router       /users/{id} [get]
func (h *UserHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}

	user, err := h.userService.GetByID(uint(id))
	if err != nil {
		response.ErrorWithMsg(c, errcode.NotFound, err.Error())
		return
	}

	response.Success(c, user)
}

// List 用户列表
// @Summary      用户列表
// @Description  分页查询用户列表，支持关键词搜索
// @Tags         用户管理
// @Security     BearerAuth
// @Produce      json
// @Param        page      query int    false "页码"    default(1) example(1)
// @Param        page_size query int    false "每页条数" default(10) example(10)
// @Param        keyword   query string false "搜索关键词(用户名/邮箱)" example(admin)
// @Success      200  {object}  response.Response{data=response.PageData}  "查询成功"
// @Router       /users [get]
func (h *UserHandler) List(c *gin.Context) {
	var req model.ListUserRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, err.Error())
		return
	}

	users, total, err := h.userService.List(&req)
	if err != nil {
		response.Error(c, errcode.DBError)
		return
	}

	response.SuccessWithPage(c, users, total, req.Page, req.PageSize)
}

// Update 更新用户
// @Summary      更新用户
// @Description  更新用户的邮箱和状态
// @Tags         用户管理
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id      path int                   true "用户ID"
// @Param        request body model.UpdateUserRequest true "更新参数"
// @Success      200  {object}  response.Response  "更新成功"
// @Failure      404  {object}  response.Response  "用户不存在"
// @Router       /users/{id} [put]
func (h *UserHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}

	var req model.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, err.Error())
		return
	}

	if err := h.userService.Update(uint(id), &req); err != nil {
		response.ErrorWithMsg(c, errcode.NotFound, err.Error())
		return
	}

	response.Success(c, nil)
}

// Delete 删除用户
// @Summary      删除用户
// @Description  根据 ID 删除一个用户
// @Tags         用户管理
// @Security     BearerAuth
// @Produce      json
// @Param        id path int true "用户ID"
// @Success      200  {object}  response.Response  "删除成功"
// @Failure      404  {object}  response.Response  "用户不存在"
// @Router       /users/{id} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}

	if err := h.userService.Delete(uint(id)); err != nil {
		if strings.Contains(err.Error(), "不存在") {
			response.ErrorWithMsg(c, errcode.NotFound, err.Error())
		} else {
			response.ErrorWithMsg(c, errcode.InvalidParams, err.Error())
		}
		return
	}

	response.Success(c, nil)
}

// ChangePassword 修改用户密码
// @Summary      修改密码
// @Description  修改指定用户的密码，需提供旧密码验证
// @Tags         用户管理
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id      path int                       true "用户ID"
// @Param        request body model.ChangePasswordRequest true "修改密码参数"
// @Success      200  {object}  response.Response  "修改成功"
// @Failure      400  {object}  response.Response  "旧密码错误"
// @Failure      404  {object}  response.Response  "用户不存在"
// @Router       /users/{id}/password [put]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}

	var req model.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, err.Error())
		return
	}

	if err := h.userService.ChangePassword(uint(id), &req); err != nil {
		if err.Error() == "旧密码错误" {
			response.ErrorWithMsg(c, errcode.PasswordWrong, err.Error())
			return
		}
		response.ErrorWithMsg(c, errcode.NotFound, err.Error())
		return
	}

	response.Success(c, nil)
}

// AssignRoles 为用户分配角色
// @Summary      为用户分配角色
// @Description  覆盖式更新用户的角色列表
// @Tags         用户管理
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id      path int                    true "用户ID"
// @Param        request body model.AssignRolesRequest true "角色ID列表"
// @Success      200  {object}  response.Response  "分配成功"
// @Failure      404  {object}  response.Response  "用户不存在"
// @Router       /users/{id}/roles [post]
func (h *UserHandler) AssignRoles(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}

	var req model.AssignRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, err.Error())
		return
	}

	if err := h.userService.AssignRoles(uint(id), req.RoleIDs); err != nil {
		response.ErrorWithMsg(c, errcode.NotFound, err.Error())
		return
	}

	response.Success(c, nil)
}

// RemoveRole 移除用户角色
// @Summary      移除用户角色
// @Description  移除用户的某个角色
// @Tags         用户管理
// @Security     BearerAuth
// @Produce      json
// @Param        id     path int true "用户ID"
// @Param        roleId path int true "角色ID"
// @Success      200  {object}  response.Response  "移除成功"
// @Failure      404  {object}  response.Response  "用户不存在"
// @Router       /users/{id}/roles/{roleId} [delete]
func (h *UserHandler) RemoveRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}
	roleID, err := strconv.ParseUint(c.Param("roleId"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}

	if err := h.userService.RemoveRole(uint(id), uint(roleID)); err != nil {
		response.ErrorWithMsg(c, errcode.NotFound, err.Error())
		return
	}

	response.Success(c, nil)
}
