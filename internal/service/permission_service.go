package service

import (
	"errors"

	"gorm.io/gorm"

	"api-rbac/internal/model"
	"api-rbac/internal/repository"
)

type PermissionService struct {
	permRepo *repository.PermissionRepo
}

func NewPermissionService(permRepo *repository.PermissionRepo) *PermissionService {
	return &PermissionService{permRepo: permRepo}
}

func (s *PermissionService) Create(req *model.CreatePermissionRequest) (*model.Permission, error) {
	exists, err := s.permRepo.ExistsByName(req.Name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("权限名称已存在")
	}

	perm := &model.Permission{
		Name:        req.Name,
		Resource:    req.Resource,
		Action:      req.Action,
		Description: req.Description,
	}

	if err := s.permRepo.Create(perm); err != nil {
		return nil, err
	}
	return perm, nil
}

func (s *PermissionService) GetByID(id uint) (*model.Permission, error) {
	perm, err := s.permRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("权限不存在")
		}
		return nil, err
	}
	return perm, nil
}

func (s *PermissionService) List(req *model.ListPermissionRequest) ([]model.Permission, int64, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	return s.permRepo.List(req.Page, req.PageSize, req.Keyword)
}

func (s *PermissionService) Update(id uint, req *model.UpdatePermissionRequest) error {
	perm, err := s.permRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("权限不存在")
		}
		return err
	}

	exists, err := s.permRepo.ExistsByName(req.Name, id)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("权限名称已存在")
	}

	perm.Name = req.Name
	perm.Resource = req.Resource
	perm.Action = req.Action
	perm.Description = req.Description
	return s.permRepo.Update(perm)
}

func (s *PermissionService) Delete(id uint) error {
	_, err := s.permRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("权限不存在")
		}
		return err
	}
	return s.permRepo.Delete(id)
}
