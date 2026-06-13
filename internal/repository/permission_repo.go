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
	return r.db.Model(perm).Select("name", "resource", "action", "description", "updated_at").Updates(perm).Error
}

func (r *PermissionRepo) Delete(id uint) error {
	return r.db.Delete(&model.Permission{}, id).Error
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
