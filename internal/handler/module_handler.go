package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/laazua/api-rbac/internal/model"
	"github.com/laazua/api-rbac/internal/service"
	"github.com/laazua/api-rbac/pkg/errcode"
	"github.com/laazua/api-rbac/pkg/response"
)

type ModuleHandler struct {
	moduleService *service.ModuleService
}

func NewModuleHandler(moduleService *service.ModuleService) *ModuleHandler {
	return &ModuleHandler{moduleService: moduleService}
}

// Create 创建模块
// @Summary      创建模块
// @Description  创建一个新的功能模块
// @Tags         模块管理
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request body model.CreateModuleRequest true "模块参数"
// @Success      200  {object}  response.Response{data=model.Module}  "创建成功"
// @Failure      400  {object}  response.Response  "参数错误"
// @Router       /modules [post]
func (h *ModuleHandler) Create(c *gin.Context) {
	var req model.CreateModuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, err.Error())
		return
	}

	m, err := h.moduleService.Create(&req)
	if err != nil {
		response.ErrorWithMsg(c, errcode.AlreadyExists, err.Error())
		return
	}

	response.Success(c, m)
}

// GetByID 获取模块详情
// @Summary      获取模块详情
// @Description  根据 ID 获取模块详细信息
// @Tags         模块管理
// @Security     BearerAuth
// @Produce      json
// @Param        id path int true "模块ID"
// @Success      200  {object}  response.Response{data=model.Module}  "查询成功"
// @Failure      404  {object}  response.Response  "模块不存在"
// @Router       /modules/{id} [get]
func (h *ModuleHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, "无效的模块ID")
		return
	}

	m, err := h.moduleService.GetByID(uint(id))
	if err != nil {
		response.ErrorWithMsg(c, errcode.NotFound, err.Error())
		return
	}

	response.Success(c, m)
}

// List 模块列表
// @Summary      模块列表
// @Description  分页查询模块列表，支持关键词搜索
// @Tags         模块管理
// @Security     BearerAuth
// @Produce      json
// @Param        page      query int    false "页码" default(1)
// @Param        page_size query int    false "每页条数" default(10)
// @Param        keyword   query string false "搜索关键词"
// @Success      200  {object}  response.Response{data=response.PageData}  "查询成功"
// @Router       /modules [get]
func (h *ModuleHandler) List(c *gin.Context) {
	var req model.ListModuleRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, err.Error())
		return
	}

	modules, total, err := h.moduleService.List(&req)
	if err != nil {
		response.ErrorWithMsg(c, errcode.InternalError, err.Error())
		return
	}

	response.SuccessWithPage(c, modules, total, req.Page, req.PageSize)
}

// Update 更新模块
// @Summary      更新模块
// @Description  更新指定模块的信息
// @Tags         模块管理
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id      path int                      true "模块ID"
// @Param        request body model.UpdateModuleRequest true "更新参数"
// @Success      200  {object}  response.Response  "更新成功"
// @Failure      404  {object}  response.Response  "模块不存在"
// @Router       /modules/{id} [put]
func (h *ModuleHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, "无效的模块ID")
		return
	}

	var req model.UpdateModuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, err.Error())
		return
	}

	if err := h.moduleService.Update(uint(id), &req); err != nil {
		response.ErrorWithMsg(c, errcode.NotFound, err.Error())
		return
	}

	response.Success(c, nil)
}

// Delete 删除模块
// @Summary      删除模块
// @Description  软删除指定模块
// @Tags         模块管理
// @Security     BearerAuth
// @Produce      json
// @Param        id path int true "模块ID"
// @Success      200  {object}  response.Response  "删除成功"
// @Failure      404  {object}  response.Response  "模块不存在"
// @Router       /modules/{id} [delete]
func (h *ModuleHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, "无效的模块ID")
		return
	}

	if err := h.moduleService.Delete(uint(id)); err != nil {
		response.ErrorWithMsg(c, errcode.NotFound, err.Error())
		return
	}

	response.Success(c, nil)
}
