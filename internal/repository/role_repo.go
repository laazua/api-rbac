package repository

import (
	"api-rbac/internal/model"

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
	err := r.db.Preload("Permissions").First(&role, id).Error
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
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
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
	err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&roles).Error
	return roles, total, err
}

func (r *RoleRepo) Update(role *model.Role) error {
	return r.db.Model(role).Select("name", "description", "updated_at").Updates(role).Error
}

func (r *RoleRepo) Delete(id uint) error {
	return r.db.Delete(&model.Role{}, id).Error
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

func (r *RoleRepo) ExistsByName(name string, excludeID ...uint) (bool, error) {
	var count int64
	query := r.db.Model(&model.Role{}).Where("name = ?", name)
	if len(excludeID) > 0 && excludeID[0] > 0 {
		query = query.Where("id != ?", excludeID[0])
	}
	err := query.Count(&count).Error
	return count > 0, err
}
