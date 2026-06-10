package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"api-rbac/internal/model"
	"api-rbac/internal/service"
	"api-rbac/pkg/errcode"
	"api-rbac/pkg/response"
)

type RoleHandler struct {
	roleService *service.RoleService
}

func NewRoleHandler(roleService *service.RoleService) *RoleHandler {
	return &RoleHandler{roleService: roleService}
}

// Create 创建角色
// @Summary      创建角色
// @Description  新增一个角色
// @Tags         角色管理
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request body model.CreateRoleRequest true "创建角色参数"
// @Success      200  {object}  response.Response{data=model.Role}  "创建成功"
// @Failure      400  {object}  response.Response  "参数错误或角色名已存在"
// @Router       /roles [post]
func (h *RoleHandler) Create(c *gin.Context) {
	var req model.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, err.Error())
		return
	}

	role, err := h.roleService.Create(&req)
	if err != nil {
		response.ErrorWithMsg(c, errcode.AlreadyExists, err.Error())
		return
	}

	response.Success(c, role)
}

// GetByID 获取角色详情
// @Summary      获取角色详情
// @Description  根据 ID 获取角色信息，包含关联的权限列表
// @Tags         角色管理
// @Security     BearerAuth
// @Produce      json
// @Param        id path int true "角色ID"
// @Success      200  {object}  response.Response{data=model.Role}  "查询成功"
// @Failure      404  {object}  response.Response  "角色不存在"
// @Router       /roles/{id} [get]
func (h *RoleHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}

	role, err := h.roleService.GetByID(uint(id))
	if err != nil {
		response.ErrorWithMsg(c, errcode.NotFound, err.Error())
		return
	}

	response.Success(c, role)
}

// List 角色列表
// @Summary      角色列表
// @Description  分页查询角色列表，支持关键词搜索
// @Tags         角色管理
// @Security     BearerAuth
// @Produce      json
// @Param        page      query int    false "页码"    default(1) example(1)
// @Param        page_size query int    false "每页条数" default(10) example(10)
// @Param        keyword   query string false "搜索关键词(角色名/描述)" example(管理员)
// @Success      200  {object}  response.Response{data=response.PageData}  "查询成功"
// @Router       /roles [get]
func (h *RoleHandler) List(c *gin.Context) {
	var req model.ListRoleRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, err.Error())
		return
	}

	roles, total, err := h.roleService.List(&req)
	if err != nil {
		response.Error(c, errcode.DBError)
		return
	}

	response.SuccessWithPage(c, roles, total, req.Page, req.PageSize)
}

// Update 更新角色
// @Summary      更新角色
// @Description  更新角色的名称和描述
// @Tags         角色管理
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id      path int                   true "角色ID"
// @Param        request body model.UpdateRoleRequest true "更新参数"
// @Success      200  {object}  response.Response  "更新成功"
// @Failure      404  {object}  response.Response  "角色不存在"
// @Router       /roles/{id} [put]
func (h *RoleHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}

	var req model.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, err.Error())
		return
	}

	if err := h.roleService.Update(uint(id), &req); err != nil {
		response.ErrorWithMsg(c, errcode.NotFound, err.Error())
		return
	}

	response.Success(c, nil)
}

// Delete 删除角色
// @Summary      删除角色
// @Description  根据 ID 删除一个角色
// @Tags         角色管理
// @Security     BearerAuth
// @Produce      json
// @Param        id path int true "角色ID"
// @Success      200  {object}  response.Response  "删除成功"
// @Failure      404  {object}  response.Response  "角色不存在"
// @Router       /roles/{id} [delete]
func (h *RoleHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}

	if err := h.roleService.Delete(uint(id)); err != nil {
		response.ErrorWithMsg(c, errcode.NotFound, err.Error())
		return
	}

	response.Success(c, nil)
}

// AssignPermissions 为角色分配权限
// @Summary      为角色分配权限
// @Description  覆盖式更新角色的权限列表
// @Tags         角色管理
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id      path int                        true "角色ID"
// @Param        request body model.AssignPermissionsRequest true "权限ID列表"
// @Success      200  {object}  response.Response  "分配成功"
// @Failure      404  {object}  response.Response  "角色不存在"
// @Router       /roles/{id}/permissions [post]
func (h *RoleHandler) AssignPermissions(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}

	var req model.AssignPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, err.Error())
		return
	}

	if err := h.roleService.AssignPermissions(uint(id), req.PermissionIDs); err != nil {
		response.ErrorWithMsg(c, errcode.NotFound, err.Error())
		return
	}

	response.Success(c, nil)
}

// RemovePermission 移除角色权限
// @Summary      移除角色权限
// @Description  移除角色的某个权限
// @Tags         角色管理
// @Security     BearerAuth
// @Produce      json
// @Param        id     path int true "角色ID"
// @Param        permId path int true "权限ID"
// @Success      200  {object}  response.Response  "移除成功"
// @Failure      404  {object}  response.Response  "角色不存在"
// @Router       /roles/{id}/permissions/{permId} [delete]
func (h *RoleHandler) RemovePermission(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	permID, _ := strconv.ParseUint(c.Param("permId"), 10, 64)

	if err := h.roleService.RemovePermission(uint(id), uint(permID)); err != nil {
		response.ErrorWithMsg(c, errcode.NotFound, err.Error())
		return
	}

	response.Success(c, nil)
}
