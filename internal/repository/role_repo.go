package repository

import (
	"github.com/laazua/api-rbac/internal/model"

	"gorm.io/gorm"
)

type RoleRepo struct {
	db *gorm.DB
}

func NewRoleRepo(db *gorm.DB) *RoleRepo {
	return &RoleRepo{db: db}
}

func (r *RoleRepo) Create(role *model.Role) error {
	return r.db.Create(role).Error
}

func (r *RoleRepo) FindByID(id uint) (*model.Role, error) {
	var role model.Role
	err := r.db.Preload("Permissions").Preload("Modules").First(&role, id).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepo) List(page, pageSize int, keyword string) ([]model.Role, int64, error) {
	var roles []model.Role
	var total int64

	query := r.db.Model(&model.Role{})
	if keyword != "" {
		escaped := escapeLike(keyword)
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+escaped+"%", "%"+escaped+"%")
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
	err := query.Preload("Modules").Preload("Permissions").Offset(offset).Limit(pageSize).Order("id DESC").Find(&roles).Error
	return roles, total, err
}

func (r *RoleRepo) Update(role *model.Role) error {
	return r.db.Model(role).Select("name", "description", "updated_at").Updates(role).Error
}

func (r *RoleRepo) Delete(id uint) error {
	return r.db.Delete(&model.Role{}, id).Error
}

// Restore 恢复已软删除的角色（将 deleted_at 置为 NULL）
func (r *RoleRepo) Restore(id uint) error {
	return r.db.Unscoped().Model(&model.Role{}).Where("id = ?", id).Update("deleted_at", nil).Error
}

// FindByNameIncludingDeleted 查询角色（包括已软删除的）
func (r *RoleRepo) FindByNameIncludingDeleted(name string) (*model.Role, error) {
	var role model.Role
	err := r.db.Unscoped().Where("name = ?", name).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepo) AssignPermissions(role *model.Role, perms []model.Permission) error {
	return r.db.Model(role).Association("Permissions").Replace(perms)
}

func (r *RoleRepo) RemovePermission(roleID, permID uint) error {
	return r.db.Model(&model.Role{BaseModel: model.BaseModel{ID: roleID}}).Association("Permissions").Delete(&model.Permission{BaseModel: model.BaseModel{ID: permID}})
}

func (r *RoleRepo) FindByIDs(ids []uint) ([]model.Role, error) {
	var roles []model.Role
	err := r.db.Where("id IN ?", ids).Find(&roles).Error
	return roles, err
}

// FindUserIDsByRoleID 查询拥有该角色的所有用户 ID 列表
func (r *RoleRepo) FindUserIDsByRoleID(roleID uint) ([]uint, error) {
	var userIDs []uint
	err := r.db.Table("user_roles").Where("role_id = ?", roleID).Pluck("user_id", &userIDs).Error
	return userIDs, err
}

func (r *RoleRepo) AssignModules(role *model.Role, modules []model.Module) error {
	return r.db.Model(role).Association("Modules").Replace(modules)
}

func (r *RoleRepo) RemoveModule(roleID, moduleID uint) error {
	return r.db.Model(&model.Role{BaseModel: model.BaseModel{ID: roleID}}).Association("Modules").Delete(&model.Module{BaseModel: model.BaseModel{ID: moduleID}})
}

func (r *RoleRepo) ExistsByName(name string, excludeID ...uint) (bool, error) {
	var count int64
	query := r.db.Model(&model.Role{}).Where("name = ?", name)
	if len(excludeID) > 0 && excludeID[0] > 0 {
		query = query.Where("id != ?", excludeID[0])
	}
	err := query.Count(&count).Error
	return count > 0, err
}
