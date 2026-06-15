package repository

import (
	"github.com/laazua/api-rbac/internal/model"

	"gorm.io/gorm"
)

type PermissionRepo struct {
	db *gorm.DB
}

func NewPermissionRepo(db *gorm.DB) *PermissionRepo {
	return &PermissionRepo{db: db}
}

func (r *PermissionRepo) Create(perm *model.Permission) error {
	return r.db.Create(perm).Error
}

func (r *PermissionRepo) FindByID(id uint) (*model.Permission, error) {
	var perm model.Permission
	err := r.db.First(&perm, id).Error
	if err != nil {
		return nil, err
	}
	return &perm, nil
}

func (r *PermissionRepo) FindByIDs(ids []uint) ([]model.Permission, error) {
	var perms []model.Permission
	err := r.db.Where("id IN ?", ids).Find(&perms).Error
	return perms, err
}

func (r *PermissionRepo) List(page, pageSize int, keyword string) ([]model.Permission, int64, error) {
	var perms []model.Permission
	var total int64

	query := r.db.Model(&model.Permission{})
	if keyword != "" {
		escaped := escapeLike(keyword)
		query = query.Where("name LIKE ? OR resource LIKE ? OR action LIKE ?",
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
	err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&perms).Error
	return perms, total, err
}

func (r *PermissionRepo) Update(perm *model.Permission) error {
	return r.db.Model(perm).Select("name", "resource", "action", "description", "module_id", "updated_at").Updates(perm).Error
}

func (r *PermissionRepo) Delete(id uint) error {
	return r.db.Delete(&model.Permission{}, id).Error
}

// Restore 恢复已软删除的权限（将 deleted_at 置为 NULL）
func (r *PermissionRepo) Restore(id uint) error {
	return r.db.Unscoped().Model(&model.Permission{}).Where("id = ?", id).Update("deleted_at", nil).Error
}

// FindByNameIncludingDeleted 查询权限（包括已软删除的）
func (r *PermissionRepo) FindByNameIncludingDeleted(name string) (*model.Permission, error) {
	var perm model.Permission
	err := r.db.Unscoped().Where("name = ?", name).First(&perm).Error
	if err != nil {
		return nil, err
	}
	return &perm, nil
}

// CountRolesByPermissionID 统计引用了该权限的角色数量
func (r *PermissionRepo) CountRolesByPermissionID(permID uint) (int64, error) {
	var count int64
	err := r.db.Table("role_permissions").Where("permission_id = ?", permID).Count(&count).Error
	return count, err
}

func (r *PermissionRepo) ExistsByName(name string, excludeID ...uint) (bool, error) {
	var count int64
	query := r.db.Model(&model.Permission{}).Where("name = ?", name)
	if len(excludeID) > 0 && excludeID[0] > 0 {
		query = query.Where("id != ?", excludeID[0])
	}
	err := query.Count(&count).Error
	return count > 0, err
}
