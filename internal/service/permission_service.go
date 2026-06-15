package service

import (
	"errors"

	"gorm.io/gorm"

	"github.com/laazua/api-rbac/internal/model"
	"github.com/laazua/api-rbac/internal/repository"
)

type PermissionService struct {
	permRepo *repository.PermissionRepo
}

func NewPermissionService(permRepo *repository.PermissionRepo) *PermissionService {
	return &PermissionService{permRepo: permRepo}
}

func (s *PermissionService) Create(req *model.CreatePermissionRequest) (*model.Permission, error) {
	// 检查是否存在同名权限（包括已软删除的）
	existing, err := s.permRepo.FindByNameIncludingDeleted(req.Name)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing != nil {
		if existing.DeletedAt.Valid {
			// 存在同名已软删除的权限，恢复并更新该记录
			if err := s.permRepo.Restore(existing.ID); err != nil {
				return nil, err
			}
			existing.Name = req.Name
			existing.Resource = req.Resource
			existing.Action = req.Action
			existing.Description = req.Description
			if err := s.permRepo.Update(existing); err != nil {
				return nil, err
			}
			return s.permRepo.FindByID(existing.ID)
		}
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

	// 检查是否有角色关联了该权限，有关联则不允许删除
	roleCount, err := s.permRepo.CountRolesByPermissionID(id)
	if err != nil {
		return err
	}
	if roleCount > 0 {
		return errors.New("该权限已被角色关联，请先解除角色关联后再删除")
	}

	return s.permRepo.Delete(id)
}
