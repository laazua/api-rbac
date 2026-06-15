package repository

import (
	"github.com/laazua/api-rbac/internal/model"

	"gorm.io/gorm"
)

type ModuleRepo struct {
	db *gorm.DB
}

func NewModuleRepo(db *gorm.DB) *ModuleRepo {
	return &ModuleRepo{db: db}
}

func (r *ModuleRepo) Create(m *model.Module) error {
	return r.db.Create(m).Error
}

func (r *ModuleRepo) FindByID(id uint) (*model.Module, error) {
	var m model.Module
	err := r.db.First(&m, id).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *ModuleRepo) FindByIDs(ids []uint) ([]model.Module, error) {
	var modules []model.Module
	err := r.db.Where("id IN ?", ids).Find(&modules).Error
	return modules, err
}

func (r *ModuleRepo) FindByCode(code string) (*model.Module, error) {
	var m model.Module
	err := r.db.Where("code = ?", code).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *ModuleRepo) List(page, pageSize int, keyword string) ([]model.Module, int64, error) {
	var modules []model.Module
	var total int64

	query := r.db.Model(&model.Module{})
	if keyword != "" {
		escaped := escapeLike(keyword)
		query = query.Where("name LIKE ? OR code LIKE ? OR description LIKE ?",
			"%"+escaped+"%", "%"+escaped+"%", "%"+escaped+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if pageSize > 100 {
		pageSize = 100
	}
	if offset > 10000 {
		offset = 10000
	}
	err := query.Offset(offset).Limit(pageSize).Order("sort ASC, id ASC").Find(&modules).Error
	return modules, total, err
}

func (r *ModuleRepo) Update(m *model.Module) error {
	return r.db.Model(m).Select("name", "code", "icon", "description", "sort", "status", "url", "updated_at").Updates(m).Error
}

func (r *ModuleRepo) Delete(id uint) error {
	return r.db.Delete(&model.Module{}, id).Error
}

// Restore 恢复已软删除的模块
func (r *ModuleRepo) Restore(id uint) error {
	return r.db.Unscoped().Model(&model.Module{}).Where("id = ?", id).Update("deleted_at", nil).Error
}

// FindByNameIncludingDeleted 查询模块（包括已软删除的）
func (r *ModuleRepo) FindByNameIncludingDeleted(name string) (*model.Module, error) {
	var m model.Module
	err := r.db.Unscoped().Where("name = ?", name).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// FindByCodeIncludingDeleted 查询模块（包括已软删除的）
func (r *ModuleRepo) FindByCodeIncludingDeleted(code string) (*model.Module, error) {
	var m model.Module
	err := r.db.Unscoped().Where("code = ?", code).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// ExistsByName 检查模块名称是否已存在（排除指定ID）
func (r *ModuleRepo) ExistsByName(name string, excludeID ...uint) (bool, error) {
	var count int64
	query := r.db.Model(&model.Module{}).Where("name = ?", name)
	if len(excludeID) > 0 && excludeID[0] > 0 {
		query = query.Where("id != ?", excludeID[0])
	}
	err := query.Count(&count).Error
	return count > 0, err
}

// ExistsByCode 检查模块编码是否已存在（排除指定ID）
func (r *ModuleRepo) ExistsByCode(code string, excludeID ...uint) (bool, error) {
	var count int64
	query := r.db.Model(&model.Module{}).Where("code = ?", code)
	if len(excludeID) > 0 && excludeID[0] > 0 {
		query = query.Where("id != ?", excludeID[0])
	}
	err := query.Count(&count).Error
	return count > 0, err
}

// CountPermissionsByModuleID 统计模块下关联的权限数量
func (r *ModuleRepo) CountPermissionsByModuleID(moduleID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.Permission{}).Where("module_id = ?", moduleID).Count(&count).Error
	return count, err
}

// FindModulesByUserID 查找用户可访问的模块
// 两条路径：1) 角色→权限→模块  2) 角色→模块直接绑定
func (r *ModuleRepo) FindModulesByUserID(userID uint) ([]model.Module, error) {
	var modules []model.Module

	// 路径1: 通过权限间接获取模块
	permPath := r.db.
		Select("DISTINCT modules.id").
		Table("modules").
		Joins("INNER JOIN permissions p ON p.module_id = modules.id AND p.deleted_at IS NULL").
		Joins("INNER JOIN role_permissions rp ON rp.permission_id = p.id").
		Joins("INNER JOIN user_roles ur ON ur.role_id = rp.role_id").
		Where("ur.user_id = ?", userID)

	// 路径2: 角色直接绑定模块
	modulePath := r.db.
		Select("DISTINCT modules.id").
		Table("modules").
		Joins("INNER JOIN role_modules rm ON rm.module_id = modules.id").
		Joins("INNER JOIN user_roles ur ON ur.role_id = rm.role_id").
		Where("ur.user_id = ?", userID)

	// UNION 两条路径
	err := r.db.
		Where("modules.status = 1").
		Where("modules.id IN (?)", r.db.Raw("? UNION ?", permPath, modulePath)).
		Order("modules.sort ASC, modules.id ASC").
		Find(&modules).Error
	return modules, err
}

// FindAllEnabled 获取所有启用状态的模块
func (r *ModuleRepo) FindAllEnabled() ([]model.Module, error) {
	var modules []model.Module
	err := r.db.Where("status = 1").Order("sort ASC, id ASC").Find(&modules).Error
	return modules, err
}
