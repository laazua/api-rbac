package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/laazua/api-rbac/internal/model"
	"github.com/laazua/api-rbac/internal/repository"
)

type ServiceAccountService struct {
	repo *repository.ServiceAccountRepo
}

func NewServiceAccountService(repo *repository.ServiceAccountRepo) *ServiceAccountService {
	return &ServiceAccountService{repo: repo}
}

// Create 创建服务账号，返回的 ApiKey 仅此一次明文展示
func (s *ServiceAccountService) Create(req *model.CreateServiceAccountRequest) (*model.ServiceAccount, string, error) {
	apiKey, err := generateApiKey()
	if err != nil {
		return nil, "", fmt.Errorf("生成API Key失败: %w", err)
	}

	sa := &model.ServiceAccount{
		Name:        req.Name,
		ApiKeyHash:  hashApiKey(apiKey),
		Status:      1,
		Description: req.Description,
	}

	if err := s.repo.Create(sa); err != nil {
		return nil, "", err
	}

	return sa, apiKey, nil
}

func (s *ServiceAccountService) GetByID(id uint) (*model.ServiceAccount, error) {
	sa, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("服务账号不存在")
		}
		return nil, err
	}
	return sa, nil
}

func (s *ServiceAccountService) List(req *model.ListServiceAccountRequest) ([]model.ServiceAccount, int64, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	return s.repo.List(req.Page, req.PageSize, req.Keyword)
}

func (s *ServiceAccountService) Update(id uint, req *model.UpdateServiceAccountRequest) error {
	sa, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("服务账号不存在")
		}
		return err
	}

	sa.Description = req.Description
	if req.Status != nil {
		sa.Status = *req.Status
	}

	return s.repo.Update(sa)
}

func (s *ServiceAccountService) Delete(id uint) error {
	_, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("服务账号不存在")
		}
		return err
	}
	return s.repo.Delete(id)
}

// VerifyApiKey 验证 API Key 并返回对应的服务账号
func (s *ServiceAccountService) VerifyApiKey(apiKey string) (*model.ServiceAccount, error) {
	return s.repo.FindByApiKeyHash(hashApiKey(apiKey))
}

// generateApiKey 生成 API Key，格式: rbac_sa_ + 32位随机hex
func generateApiKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "rbac_sa_" + hex.EncodeToString(b), nil
}

func hashApiKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}
