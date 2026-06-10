package service

import (
	"errors"

	"gorm.io/gorm"

	"api-rbac/internal/model"
	"api-rbac/internal/repository"
)

type RoleService struct {
	roleRepo       *repository.RoleRepo
	permissionRepo *repository.PermissionRepo
}

func NewRoleService(roleRepo *repository.RoleRepo, permissionRepo *repository.PermissionRepo) *RoleService {
	return &RoleService{roleRepo: roleRepo, permissionRepo: permissionRepo}
}

func (s *RoleService) Create(req *model.CreateRoleRequest) (*model.Role, error) {
	exists, err := s.roleRepo.ExistsByName(req.Name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("角色名称已存在")
	}

	role := &model.Role{
		Name:        req.Name,
		Description: req.Description,
	}

	if err := s.roleRepo.Create(role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *RoleService) GetByID(id uint) (*model.Role, error) {
	role, err := s.roleRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("角色不存在")
		}
		return nil, err
	}
	return role, nil
}

func (s *RoleService) List(req *model.ListRoleRequest) ([]model.Role, int64, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	return s.roleRepo.List(req.Page, req.PageSize, req.Keyword)
}

func (s *RoleService) Update(id uint, req *model.UpdateRoleRequest) error {
	role, err := s.roleRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("角色不存在")
		}
		return err
	}

	exists, err := s.roleRepo.ExistsByName(req.Name, id)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("角色名称已存在")
	}

	role.Name = req.Name
	role.Description = req.Description
	return s.roleRepo.Update(role)
}

func (s *RoleService) Delete(id uint) error {
	_, err := s.roleRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("角色不存在")
		}
		return err
	}
	return s.roleRepo.Delete(id)
}

func (s *RoleService) AssignPermissions(id uint, permIDs []uint) error {
	role, err := s.roleRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("角色不存在")
		}
		return err
	}

	perms, err := s.permissionRepo.FindByIDs(permIDs)
	if err != nil {
		return err
	}

	return s.roleRepo.AssignPermissions(role, perms)
}

func (s *RoleService) RemovePermission(id, permID uint) error {
	_, err := s.roleRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("角色不存在")
		}
		return err
	}
	return s.roleRepo.RemovePermission(id, permID)
}
