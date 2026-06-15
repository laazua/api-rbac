package service

import (
	"errors"

	"gorm.io/gorm"

	"github.com/laazua/api-rbac/internal/model"
	"github.com/laazua/api-rbac/internal/repository"
)

type ModuleService struct {
	moduleRepo *repository.ModuleRepo
}

func NewModuleService(moduleRepo *repository.ModuleRepo) *ModuleService {
	return &ModuleService{moduleRepo: moduleRepo}
}

func (s *ModuleService) Create(req *model.CreateModuleRequest) (*model.Module, error) {
	// 检查是否存在同名模块（包括已软删除的）
	existing, err := s.moduleRepo.FindByNameIncludingDeleted(req.Name)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing != nil {
		if existing.DeletedAt.Valid {
			// 存在同名已软删除的模块，恢复并更新
			if err := s.moduleRepo.Restore(existing.ID); err != nil {
				return nil, err
			}
			existing.Name = req.Name
			existing.Code = req.Code
			existing.Icon = req.Icon
			existing.Description = req.Description
			existing.Sort = req.Sort
			existing.Url = req.Url
			if err := s.moduleRepo.Update(existing); err != nil {
				return nil, err
			}
			return s.moduleRepo.FindByID(existing.ID)
		}
		return nil, errors.New("模块名称已存在")
	}

	// 检查编码唯一性
	codeExists, err := s.moduleRepo.ExistsByCode(req.Code)
	if err != nil {
		return nil, err
	}
	if codeExists {
		// 检查是否为已软删除的
		codeExisting, err := s.moduleRepo.FindByCodeIncludingDeleted(req.Code)
		if err == nil && codeExisting != nil && codeExisting.DeletedAt.Valid {
			if err := s.moduleRepo.Restore(codeExisting.ID); err != nil {
				return nil, err
			}
			codeExisting.Name = req.Name
			codeExisting.Code = req.Code
			codeExisting.Icon = req.Icon
			codeExisting.Description = req.Description
			codeExisting.Sort = req.Sort
			if err := s.moduleRepo.Update(codeExisting); err != nil {
				return nil, err
			}
			return s.moduleRepo.FindByID(codeExisting.ID)
		}
		return nil, errors.New("模块编码已存在")
	}

	m := &model.Module{
		Name:        req.Name,
		Code:        req.Code,
		Icon:        req.Icon,
		Description: req.Description,
		Sort:        req.Sort,
		Status:      1,
		Url:         req.Url,
	}

	if err := s.moduleRepo.Create(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *ModuleService) GetByID(id uint) (*model.Module, error) {
	m, err := s.moduleRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("模块不存在")
		}
		return nil, err
	}
	return m, nil
}

func (s *ModuleService) List(req *model.ListModuleRequest) ([]model.Module, int64, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	return s.moduleRepo.List(req.Page, req.PageSize, req.Keyword)
}

func (s *ModuleService) Update(id uint, req *model.UpdateModuleRequest) error {
	m, err := s.moduleRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("模块不存在")
		}
		return err
	}

	exists, err := s.moduleRepo.ExistsByName(req.Name, id)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("模块名称已存在")
	}

	codeExists, err := s.moduleRepo.ExistsByCode(req.Code, id)
	if err != nil {
		return err
	}
	if codeExists {
		return errors.New("模块编码已存在")
	}

	m.Name = req.Name
	m.Code = req.Code
	m.Icon = req.Icon
	m.Description = req.Description
	m.Sort = req.Sort
	m.Url = req.Url
	if req.Status != nil {
		m.Status = *req.Status
	}
	return s.moduleRepo.Update(m)
}

func (s *ModuleService) Delete(id uint) error {
	_, err := s.moduleRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("模块不存在")
		}
		return err
	}

	// 检查是否有权限关联了该模块，有关联则不允许删除
	permCount, err := s.moduleRepo.CountPermissionsByModuleID(id)
	if err != nil {
		return err
	}
	if permCount > 0 {
		return errors.New("该模块下存在关联的权限，请先移除权限后再删除模块")
	}

	return s.moduleRepo.Delete(id)
}

// GetUserModules 获取用户可访问的模块列表
func (s *ModuleService) GetUserModules(userID uint) ([]model.Module, error) {
	return s.moduleRepo.FindModulesByUserID(userID)
}

// GetAllEnabledModules 获取所有启用状态的模块（供超管使用）
func (s *ModuleService) GetAllEnabledModules() ([]model.Module, error) {
	return s.moduleRepo.FindAllEnabled()
}
