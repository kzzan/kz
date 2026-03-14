package repository

import (
	"context"
	"fmt"
	"time"

	"example/internal/models"
	"example/pkg/cache"
	"example/pkg/pagination"

	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"gorm.io/gorm"
)

type UserRepository interface {
	List(ctx context.Context, q pagination.Query) (pagination.Result[models.User], error)
	FindByID(ctx context.Context, id string) (*models.User, error)
	Create(ctx context.Context, m *models.User) (*models.User, error)
	Update(ctx context.Context, id string, m *models.User) (*models.User, error)
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int64, error)
}

type userRepository struct {
	logger *zerolog.Logger
	db     *gorm.DB
	cache  cache.Cache
}

func NewUserRepository(i do.Injector) (UserRepository, error) {
	return &userRepository{
		logger: do.MustInvoke[*zerolog.Logger](i),
		db:     do.MustInvoke[*gorm.DB](i),
		cache:  do.MustInvoke[cache.Cache](i),
	}, nil
}

func (r *userRepository) List(ctx context.Context, q pagination.Query) (pagination.Result[models.User], error) {
	var list []models.User
	var total int64

	db := r.db.WithContext(ctx).Model(&models.User{})
	if q.Keyword != "" {
		// TODO: 填写关键字搜索字段
		// db = db.Where("name LIKE ?", "%"+q.Keyword+"%")
	}
	if q.Sort != "" {
		db = db.Order(q.Sort)
	}
	if err := db.Count(&total).Error; err != nil {
		return pagination.Result[models.User]{}, fmt.Errorf("统计 user 失败: %w", err)
	}
	if err := db.Offset((q.Page - 1) * q.Size).Limit(q.Size).Find(&list).Error; err != nil {
		return pagination.Result[models.User]{}, fmt.Errorf("查询 user 列表失败: %w", err)
	}
	return pagination.Result[models.User]{Total: total, List: list}, nil
}

func (r *userRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	cacheKey := fmt.Sprintf("user:%s", id)
	var m models.User
	if err := r.cache.GetJSON(ctx, cacheKey, &m); err == nil {
		return &m, nil
	}
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("查询 user 失败: %w", err)
	}
	_ = r.cache.SetJSON(ctx, cacheKey, m, 5*time.Minute)
	return &m, nil
}

func (r *userRepository) Create(ctx context.Context, m *models.User) (*models.User, error) {
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return nil, fmt.Errorf("创建 user 失败: %w", err)
	}
	return m, nil
}

func (r *userRepository) Update(ctx context.Context, id string, m *models.User) (*models.User, error) {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Updates(m).Error; err != nil {
		return nil, fmt.Errorf("更新 user 失败: %w", err)
	}
	_ = r.cache.Delete(ctx, fmt.Sprintf("user:%s", id))
	return r.FindByID(ctx, id)
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&models.User{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("删除 user 失败: %w", err)
	}
	_ = r.cache.Delete(ctx, fmt.Sprintf("user:%s", id))
	return nil
}

func (r *userRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.User{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计 user 失败: %w", err)
	}
	return count, nil
}