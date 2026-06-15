package service

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/laazua/api-rbac/internal/cache"
	"github.com/laazua/api-rbac/internal/model"
	"github.com/laazua/api-rbac/internal/repository"
)

type RoleService struct {
	roleRepo       *repository.RoleRepo
	permissionRepo *repository.PermissionRepo
	moduleRepo     *repository.ModuleRepo
	cache          *cache.PermissionCache
}

func NewRoleService(roleRepo *repository.RoleRepo, permissionRepo *repository.PermissionRepo, moduleRepo *repository.ModuleRepo, cache *cache.PermissionCache) *RoleService {
	return &RoleService{roleRepo: roleRepo, permissionRepo: permissionRepo, moduleRepo: moduleRepo, cache: cache}
}

func (s *RoleService) Create(req *model.CreateRoleRequest) (*model.Role, error) {
	// 检查是否存在同名角色（包括已软删除的）
	existing, err := s.roleRepo.FindByNameIncludingDeleted(req.Name)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing != nil {
		if existing.DeletedAt.Valid {
			// 存在同名已软删除的角色，恢复并更新该记录
			if err := s.roleRepo.Restore(existing.ID); err != nil {
				return nil, err
			}
			existing.Name = req.Name
			existing.Description = req.Description
			if err := s.roleRepo.Update(existing); err != nil {
				return nil, err
			}
			return s.roleRepo.FindByID(existing.ID)
		}
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

	// 检查是否有用户关联了该角色，有关联则不允许删除
	userIDs, err := s.roleRepo.FindUserIDsByRoleID(id)
	if err != nil {
		return err
	}
	if len(userIDs) > 0 {
		return errors.New("该角色已被用户关联，请先解除用户关联后再删除")
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

	// 获取权限实体（空数组则清空关联）
	var perms []model.Permission
	if len(permIDs) > 0 {
		perms, err = s.permissionRepo.FindByIDs(permIDs)
		if err != nil {
			return err
		}
	}

	if err := s.roleRepo.AssignPermissions(role, perms); err != nil {
		return err
	}

	s.invalidateCacheForRole(id)
	return nil
}

func (s *RoleService) RemovePermission(id, permID uint) error {
	_, err := s.roleRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("角色不存在")
		}
		return err
	}

	if err := s.roleRepo.RemovePermission(id, permID); err != nil {
		return err
	}

	// 权限变更后，失效所有拥有该角色的用户的缓存
	s.invalidateCacheForRole(id)
	return nil
}

func (s *RoleService) AssignModules(id uint, moduleIDs []uint) error {
	role, err := s.roleRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("角色不存在")
		}
		return err
	}

	// 获取模块实体（空数组则清空关联）
	var modules []model.Module
	if len(moduleIDs) > 0 {
		modules, err = s.moduleRepo.FindByIDs(moduleIDs)
		if err != nil {
			return err
		}
	}

	if err := s.roleRepo.AssignModules(role, modules); err != nil {
		return err
	}

	s.invalidateCacheForRole(id)
	return nil
}

func (s *RoleService) RemoveModule(id, moduleID uint) error {
	_, err := s.roleRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("角色不存在")
		}
		return err
	}

	if err := s.roleRepo.RemoveModule(id, moduleID); err != nil {
		return err
	}

	s.invalidateCacheForRole(id)
	return nil
}

func (s *RoleService) invalidateCacheForRole(roleID uint) {
	if s.cache == nil {
		return
	}
	userIDs, err := s.roleRepo.FindUserIDsByRoleID(roleID)
	if err != nil {
		return
	}
	_ = s.cache.InvalidateUsers(context.Background(), userIDs)
}
