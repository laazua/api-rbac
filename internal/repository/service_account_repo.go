package repository

import (
	"github.com/laazua/api-rbac/internal/model"

	"gorm.io/gorm"
)

type ServiceAccountRepo struct {
	db *gorm.DB
}

func NewServiceAccountRepo(db *gorm.DB) *ServiceAccountRepo {
	return &ServiceAccountRepo{db: db}
}

func (r *ServiceAccountRepo) Create(sa *model.ServiceAccount) error {
	return r.db.Create(sa).Error
}

func (r *ServiceAccountRepo) FindByID(id uint) (*model.ServiceAccount, error) {
	var sa model.ServiceAccount
	err := r.db.First(&sa, id).Error
	if err != nil {
		return nil, err
	}
	return &sa, nil
}

func (r *ServiceAccountRepo) FindByApiKeyHash(hash string) (*model.ServiceAccount, error) {
	var sa model.ServiceAccount
	err := r.db.Where("api_key_hash = ? AND status = 1", hash).First(&sa).Error
	if err != nil {
		return nil, err
	}
	return &sa, nil
}

func (r *ServiceAccountRepo) List(page, pageSize int, keyword string) ([]model.ServiceAccount, int64, error) {
	var accounts []model.ServiceAccount
	var total int64

	query := r.db.Model(&model.ServiceAccount{})
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
	err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&accounts).Error
	return accounts, total, err
}

func (r *ServiceAccountRepo) Update(sa *model.ServiceAccount) error {
	return r.db.Model(sa).Select("description", "status", "updated_at").Updates(sa).Error
}

func (r *ServiceAccountRepo) Delete(id uint) error {
	return r.db.Delete(&model.ServiceAccount{}, id).Error
}
