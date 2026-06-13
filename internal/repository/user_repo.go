package repository

import (
	"github.com/laazua/api-rbac/internal/model"

	"gorm.io/gorm"
)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepo) FindByID(id uint) (*model.User, error) {
	var user model.User
	err := r.db.Preload("Roles.Permissions").First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) FindByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) FindByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) List(page, pageSize int, keyword string) ([]model.User, int64, error) {
	var users []model.User
	var total int64

	query := r.db.Model(&model.User{}).Preload("Roles")
	if keyword != "" {
		query = query.Where("username LIKE ? OR email LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	// 防御性保护：单次最多返回 100 条，最大偏移量不超过 10000
	if pageSize > 100 {
		pageSize = 100
	}
	if offset > 10000 {
		offset = 10000
	}
	err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&users).Error
	return users, total, err
}

func (r *UserRepo) Update(user *model.User) error {
	return r.db.Model(user).Select("email", "status", "updated_at").Updates(user).Error
}

func (r *UserRepo) Delete(id uint) error {
	return r.db.Delete(&model.User{}, id).Error
}

func (r *UserRepo) AssignRoles(user *model.User, roles []model.Role) error {
	return r.db.Model(user).Association("Roles").Replace(roles)
}

func (r *UserRepo) RemoveRole(userID, roleID uint) error {
	return r.db.Model(&model.User{BaseModel: model.BaseModel{ID: userID}}).Association("Roles").Delete(&model.Role{BaseModel: model.BaseModel{ID: roleID}})
}

func (r *UserRepo) UpdatePassword(id uint, hashedPassword string) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Update("password", hashedPassword).Error
}

func (r *UserRepo) ExistsByUsername(username string, excludeID ...uint) (bool, error) {
	var count int64
	query := r.db.Model(&model.User{}).Where("username = ?", username)
	if len(excludeID) > 0 && excludeID[0] > 0 {
		query = query.Where("id != ?", excludeID[0])
	}
	err := query.Count(&count).Error
	return count > 0, err
}
