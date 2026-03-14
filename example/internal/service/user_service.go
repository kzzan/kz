package service

import (
	"context"
	"fmt"

	"example/internal/models"
	"example/internal/repository"
	"example/pkg/pagination"
	"example/pkg/queue"

	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
)

type UserService interface {
	List(ctx context.Context, q pagination.Query) (pagination.Result[models.User], error)
	GetByID(ctx context.Context, id string) (*models.User, error)
	Create(ctx context.Context, m *models.User) (*models.User, error)
	Update(ctx context.Context, id string, m *models.User) (*models.User, error)
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int64, error)
}

type userService struct {
	logger *zerolog.Logger
	repo   repository.UserRepository
	queue  queue.Queue
}

func NewUserService(i do.Injector) (UserService, error) {
	return &userService{
		logger: do.MustInvoke[*zerolog.Logger](i),
		repo:   do.MustInvoke[repository.UserRepository](i),
		queue:  do.MustInvoke[queue.Queue](i),
	}, nil
}

func (s *userService) List(ctx context.Context, q pagination.Query) (pagination.Result[models.User], error) {
	return s.repo.List(ctx, q)
}

func (s *userService) GetByID(ctx context.Context, id string) (*models.User, error) {
	if id == "" {
		return nil, fmt.Errorf("ID 不能为空")
	}
	return s.repo.FindByID(ctx, id)
}

func (s *userService) Create(ctx context.Context, m *models.User) (*models.User, error) {
	if m == nil {
		return nil, fmt.Errorf("创建数据不能为空")
	}
	result, err := s.repo.Create(ctx, m)
	if err != nil {
		return nil, err
	}
	_ = s.queue.Publish(ctx, "user.created", result)
	return result, nil
}

func (s *userService) Update(ctx context.Context, id string, m *models.User) (*models.User, error) {
	if id == "" {
		return nil, fmt.Errorf("ID 不能为空")
	}
	if m == nil {
		return nil, fmt.Errorf("更新数据不能为空")
	}
	result, err := s.repo.Update(ctx, id, m)
	if err != nil {
		return nil, err
	}
	_ = s.queue.Publish(ctx, "user.updated", result)
	return result, nil
}

func (s *userService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("ID 不能为空")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.queue.Publish(ctx, "user.deleted", map[string]string{"id": id})
	return nil
}

func (s *userService) Count(ctx context.Context) (int64, error) {
	return s.repo.Count(ctx)
}