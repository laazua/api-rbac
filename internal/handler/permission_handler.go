package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/laazua/api-rbac/internal/model"
	"github.com/laazua/api-rbac/internal/service"
	"github.com/laazua/api-rbac/pkg/errcode"
	"github.com/laazua/api-rbac/pkg/response"
)

type PermissionHandler struct {
	permService *service.PermissionService
}

func NewPermissionHandler(permService *service.PermissionService) *PermissionHandler {
	return &PermissionHandler{permService: permService}
}

// Create 创建权限
// @Summary      创建权限
// @Description  新增一个权限定义
// @Tags         权限管理
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request body model.CreatePermissionRequest true "创建权限参数"
// @Success      200  {object}  response.Response{data=model.Permission}  "创建成功"
// @Failure      400  {object}  response.Response  "参数错误或权限名已存在"
// @Router       /permissions [post]
func (h *PermissionHandler) Create(c *gin.Context) {
	var req model.CreatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, err.Error())
		return
	}

	perm, err := h.permService.Create(&req)
	if err != nil {
		response.ErrorWithMsg(c, errcode.AlreadyExists, err.Error())
		return
	}

	response.Success(c, perm)
}

// GetByID 获取权限详情
// @Summary      获取权限详情
// @Description  根据 ID 获取权限信息
// @Tags         权限管理
// @Security     BearerAuth
// @Produce      json
// @Param        id path int true "权限ID"
// @Success      200  {object}  response.Response{data=model.Permission}  "查询成功"
// @Failure      404  {object}  response.Response  "权限不存在"
// @Router       /permissions/{id} [get]
func (h *PermissionHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}

	perm, err := h.permService.GetByID(uint(id))
	if err != nil {
		response.ErrorWithMsg(c, errcode.NotFound, err.Error())
		return
	}

	response.Success(c, perm)
}

// List 权限列表
// @Summary      权限列表
// @Description  分页查询权限列表，支持关键词搜索
// @Tags         权限管理
// @Security     BearerAuth
// @Produce      json
// @Param        page      query int    false "页码"    default(1) example(1)
// @Param        page_size query int    false "每页条数" default(10) example(10)
// @Param        keyword   query string false "搜索关键词(权限名/资源/操作)" example(删除)
// @Success      200  {object}  response.Response{data=response.PageData}  "查询成功"
// @Router       /permissions [get]
func (h *PermissionHandler) List(c *gin.Context) {
	var req model.ListPermissionRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, err.Error())
		return
	}

	perms, total, err := h.permService.List(&req)
	if err != nil {
		response.Error(c, errcode.DBError)
		return
	}

	response.SuccessWithPage(c, perms, total, req.Page, req.PageSize)
}

// Update 更新权限
// @Summary      更新权限
// @Description  更新权限的名称、资源和操作
// @Tags         权限管理
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id      path int                       true "权限ID"
// @Param        request body model.UpdatePermissionRequest true "更新参数"
// @Success      200  {object}  response.Response  "更新成功"
// @Failure      404  {object}  response.Response  "权限不存在"
// @Router       /permissions/{id} [put]
func (h *PermissionHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}

	var req model.UpdatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, err.Error())
		return
	}

	if err := h.permService.Update(uint(id), &req); err != nil {
		response.ErrorWithMsg(c, errcode.NotFound, err.Error())
		return
	}

	response.Success(c, nil)
}

// Delete 删除权限
// @Summary      删除权限
// @Description  根据 ID 删除一个权限
// @Tags         权限管理
// @Security     BearerAuth
// @Produce      json
// @Param        id path int true "权限ID"
// @Success      200  {object}  response.Response  "删除成功"
// @Failure      404  {object}  response.Response  "权限不存在"
// @Router       /permissions/{id} [delete]
func (h *PermissionHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}

	if err := h.permService.Delete(uint(id)); err != nil {
		response.ErrorWithMsg(c, errcode.NotFound, err.Error())
		return
	}

	response.Success(c, nil)
}
