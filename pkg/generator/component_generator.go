package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/kzzan/kz/pkg/utils"
)

type ComponentGenerator struct {
	ComponentName string
	PascalName    string
	SnakeName     string
	ProjectRoot   string
	ModuleName    string
}

func NewComponentGenerator(componentName string) *ComponentGenerator {
	return &ComponentGenerator{
		ComponentName: componentName,
		PascalName:    utils.ToPascalCase(componentName),
		SnakeName:     utils.ToSnakeCase(componentName),
		ProjectRoot:   ".",
		ModuleName:    utils.ReadModuleName("."),
	}
}

func (cg *ComponentGenerator) GenerateCron() error {
	if err := cg.ensureDir("internal/cron"); err != nil {
		return err
	}
	const tmpl = `package cron

import (
	"context"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

type {{.Pascal}}Job struct {
	logger *zerolog.Logger
}

func new{{.Pascal}}Job(logger *zerolog.Logger) *{{.Pascal}}Job {
	return &{{.Pascal}}Job{logger: logger}
}

func (j *{{.Pascal}}Job) Register(c *cron.Cron) {
	c.AddFunc("@every 1m", j.run)
}

func (j *{{.Pascal}}Job) run() {
	ctx := context.Background()
	_ = ctx
	j.logger.Info().Str("job", "{{.Snake}}").Msg("cron job triggered")
	// TODO: 实现定时任务逻辑
}
`
	dest := filepath.Join(cg.ProjectRoot, "internal/cron", cg.SnakeName+".go")
	return cg.writeTemplate("cron", tmpl, dest, map[string]string{
		"Pascal": cg.PascalName,
		"Snake":  cg.SnakeName,
	})
}

func (cg *ComponentGenerator) EnsureCronPackage() error {
	pkgPath := filepath.Join(cg.ProjectRoot, "internal/cron/package.go")
	if _, err := os.Stat(pkgPath); err == nil {
		return nil
	}
	content := fmt.Sprintf(`package cron

import (
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
)

type Scheduler struct {
	c      *cron.Cron
	logger *zerolog.Logger
}

func NewScheduler(i do.Injector) (*Scheduler, error) {
	logger := do.MustInvoke[*zerolog.Logger](i)
	c := cron.New(cron.WithSeconds())
	s := &Scheduler{c: c, logger: logger}
	s.registerJobs()
	return s, nil
}

func (s *Scheduler) registerJobs() {
	// JOBS_PLACEHOLDER
}

func (s *Scheduler) Start() {
	s.c.Start()
	s.logger.Info().Msg("cron scheduler started")
}

func (s *Scheduler) Stop() {
	s.c.Stop()
	s.logger.Info().Msg("cron scheduler stopped")
}

func (s *Scheduler) Shutdown() error {
	s.Stop()
	return nil
}

var Package = do.Package(
	do.Lazy(NewScheduler),
)
`, cg.ModuleName)
	return os.WriteFile(pkgPath, []byte(content), 0o644)
}

func (cg *ComponentGenerator) AppendCronToScheduler() error {
	pkgPath := filepath.Join(cg.ProjectRoot, "internal/cron/package.go")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return fmt.Errorf("读取 cron/package.go 失败: %w", err)
	}
	content := string(data)

	jobLine := fmt.Sprintf("new%sJob(s.logger).Register(s.c)", cg.PascalName)
	if strings.Contains(content, jobLine) {
		return nil
	}

	placeholder := "// JOBS_PLACEHOLDER"
	newLine := fmt.Sprintf("%s\n\t%s", placeholder, jobLine)
	if strings.Contains(content, placeholder) {
		content = strings.Replace(content, placeholder, newLine, 1)
	} else {
		content = appendBeforeLastFuncClose(content, "func (s *Scheduler) registerJobs", "\t"+jobLine)
	}
	return os.WriteFile(pkgPath, []byte(content), 0o644)
}

func (cg *ComponentGenerator) GenerateCronAndRegister() error {
	return cg.runSteps([]step{
		{"生成 Cron Job 文件", cg.GenerateCron},
		{"确保 cron/package.go", cg.EnsureCronPackage},
		{"注册到 Scheduler", cg.AppendCronToScheduler},
		{"注册 cron.Package 到 server", cg.AppendCronPackageToServer},
	})
}

func (cg *ComponentGenerator) AppendCronPackageToServer() error {
	return cg.appendPackageToServerDI("cron")
}

func (cg *ComponentGenerator) GenerateConsumer() error {
	if err := cg.ensureDir("internal/consumer"); err != nil {
		return err
	}
	const tmpl = `package consumer

import (
	"context"

	"{{.Module}}/pkg/queue"

	"github.com/rs/zerolog"
)

type {{.Pascal}}Consumer struct {
	logger *zerolog.Logger
	queue  queue.Queue
}

func new{{.Pascal}}Consumer(logger *zerolog.Logger, q queue.Queue) *{{.Pascal}}Consumer {
	return &{{.Pascal}}Consumer{logger: logger, queue: q}
}

func (c *{{.Pascal}}Consumer) Topic() string {
	return "{{.Snake}}"
}

func (c *{{.Pascal}}Consumer) Start(ctx context.Context) error {
	c.logger.Info().Str("topic", c.Topic()).Msg("consumer started")
	return c.queue.Subscribe(ctx, c.Topic(), c.handle)
}

func (c *{{.Pascal}}Consumer) handle(ctx context.Context, msg *queue.Message) error {
	c.logger.Info().
		Str("topic", msg.Topic).
		Str("id", msg.ID).
		Str("payload", msg.Payload).
		Msg("{{.Snake}} message received")
	// TODO: 实现消息处理逻辑
	return nil
}
`
	dest := filepath.Join(cg.ProjectRoot, "internal/consumer", cg.SnakeName+".go")
	return cg.writeTemplate("consumer", tmpl, dest, map[string]string{
		"Pascal": cg.PascalName,
		"Snake":  cg.SnakeName,
		"Module": cg.ModuleName,
	})
}

func (cg *ComponentGenerator) EnsureConsumerPackage() error {
	pkgPath := filepath.Join(cg.ProjectRoot, "internal/consumer/package.go")
	if _, err := os.Stat(pkgPath); err == nil {
		return nil
	}
	content := fmt.Sprintf(`package consumer

import (
	"context"

	"%s/pkg/queue"

	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
)

type Manager struct {
	consumers []starter
	logger    *zerolog.Logger
}

type starter interface {
	Topic() string
	Start(ctx context.Context) error
}

func NewManager(i do.Injector) (*Manager, error) {
	logger := do.MustInvoke[*zerolog.Logger](i)
	q      := do.MustInvoke[queue.Queue](i)
	m := &Manager{logger: logger}
	// CONSUMERS_PLACEHOLDER
	_ = q
	return m, nil
}

func (m *Manager) Start(ctx context.Context) {
	for _, c := range m.consumers {
		go func(cs starter) {
			m.logger.Info().Str("topic", cs.Topic()).Msg("starting consumer")
			if err := cs.Start(ctx); err != nil {
				m.logger.Error().Err(err).Str("topic", cs.Topic()).Msg("consumer exited with error")
			}
		}(c)
	}
}

func (m *Manager) Shutdown() error {
	m.logger.Info().Msg("consumer manager shutdown")
	return nil
}

var Package = do.Package(
	do.Lazy(NewManager),
)
`, cg.ModuleName)
	return os.WriteFile(pkgPath, []byte(content), 0o644)
}

func (cg *ComponentGenerator) AppendConsumerToManager() error {
	pkgPath := filepath.Join(cg.ProjectRoot, "internal/consumer/package.go")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return fmt.Errorf("读取 consumer/package.go 失败: %w", err)
	}
	content := string(data)

	consumerLine := fmt.Sprintf("new%sConsumer(logger, q)", cg.PascalName)
	if strings.Contains(content, consumerLine) {
		return nil
	}

	placeholder := "// CONSUMERS_PLACEHOLDER"
	appendLine := fmt.Sprintf("%s\n\tm.consumers = append(m.consumers, new%sConsumer(logger, q))", placeholder, cg.PascalName)
	if strings.Contains(content, placeholder) {
		content = strings.Replace(content, placeholder, appendLine, 1)
	} else {
		content = appendBeforeLastFuncClose(content, "func NewManager",
			fmt.Sprintf("\tm.consumers = append(m.consumers, new%sConsumer(logger, q))", cg.PascalName))
	}
	return os.WriteFile(pkgPath, []byte(content), 0o644)
}

func (cg *ComponentGenerator) GenerateConsumerAndRegister() error {
	return cg.runSteps([]step{
		{"生成 Consumer 文件", cg.GenerateConsumer},
		{"确保 consumer/package.go", cg.EnsureConsumerPackage},
		{"注册到 Manager", cg.AppendConsumerToManager},
		{"注册 consumer.Package 到 server", cg.AppendConsumerPackageToServer},
	})
}

func (cg *ComponentGenerator) AppendConsumerPackageToServer() error {
	return cg.appendPackageToServerDI("consumer")
}

func (cg *ComponentGenerator) GenerateMiddleware() error {
	if err := cg.ensureDir("internal/middleware"); err != nil {
		return err
	}
	const tmpl = `package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func {{.Pascal}}(logger *zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 请求处理前逻辑
		c.Next()
		// TODO: 请求处理后逻辑（可选）
	}
}
`
	dest := filepath.Join(cg.ProjectRoot, "internal/middleware", cg.SnakeName+".go")
	return cg.writeTemplate("middleware", tmpl, dest, map[string]string{
		"Pascal": cg.PascalName,
		"Snake":  cg.SnakeName,
	})
}

func (cg *ComponentGenerator) GenerateHandler() error {
	if err := cg.ensureDir("internal/handler"); err != nil {
		return err
	}
	const tmpl = `package handler

import (
	"net/http"

	"{{.Module}}/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
)

type {{.Pascal}}Handler interface {
	List(c *gin.Context)
	Get(c *gin.Context)
	Create(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
}

type {{.Snake}}Handler struct {
	logger  *zerolog.Logger
	service service.{{.Pascal}}Service
}

func New{{.Pascal}}Handler(i do.Injector) ({{.Pascal}}Handler, error) {
	return &{{.Snake}}Handler{
		logger:  do.MustInvoke[*zerolog.Logger](i),
		service: do.MustInvoke[service.{{.Pascal}}Service](i),
	}, nil
}

func (h *{{.Snake}}Handler) List(c *gin.Context) {
	result, err := h.service.List(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("List {{.Component}} failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *{{.Snake}}Handler) Get(c *gin.Context) {
	id := c.Param("id")
	result, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("id", id).Msg("Get {{.Component}} failed")
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *{{.Snake}}Handler) Create(c *gin.Context) {
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.service.Create(c.Request.Context(), body)
	if err != nil {
		h.logger.Error().Err(err).Msg("Create {{.Component}} failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": result})
}

func (h *{{.Snake}}Handler) Update(c *gin.Context) {
	id := c.Param("id")
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.service.Update(c.Request.Context(), id, body)
	if err != nil {
		h.logger.Error().Err(err).Str("id", id).Msg("Update {{.Component}} failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *{{.Snake}}Handler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("id", id).Msg("Delete {{.Component}} failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
`
	dest := filepath.Join(cg.ProjectRoot, "internal/handler", cg.SnakeName+".go")
	return cg.writeTemplate("handler", tmpl, dest, map[string]string{
		"Pascal": cg.PascalName, "Snake": cg.SnakeName,
		"Component": cg.ComponentName, "Module": cg.ModuleName,
	})
}

func (cg *ComponentGenerator) GenerateService() error {
	if err := cg.ensureDir("internal/service"); err != nil {
		return err
	}
	const tmpl = `package service

import (
	"context"
	"fmt"

	"{{.Module}}/internal/repository"
	"{{.Module}}/pkg/queue"

	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
)

type {{.Pascal}}Service interface {
	List(ctx context.Context) ([]interface{}, error)
	GetByID(ctx context.Context, id string) (interface{}, error)
	Create(ctx context.Context, data map[string]interface{}) (interface{}, error)
	Update(ctx context.Context, id string, data map[string]interface{}) (interface{}, error)
	Delete(ctx context.Context, id string) error
}

type {{.Snake}}Service struct {
	logger *zerolog.Logger
	repo   repository.{{.Pascal}}Repository
	queue  queue.Queue
}

func New{{.Pascal}}Service(i do.Injector) ({{.Pascal}}Service, error) {
	return &{{.Snake}}Service{
		logger: do.MustInvoke[*zerolog.Logger](i),
		repo:   do.MustInvoke[repository.{{.Pascal}}Repository](i),
		queue:  do.MustInvoke[queue.Queue](i),
	}, nil
}

func (s *{{.Snake}}Service) List(ctx context.Context) ([]interface{}, error) {
	return s.repo.FindAll(ctx)
}

func (s *{{.Snake}}Service) GetByID(ctx context.Context, id string) (interface{}, error) {
	if id == "" {
		return nil, fmt.Errorf("ID 不能为空")
	}
	return s.repo.FindByID(ctx, id)
}

func (s *{{.Snake}}Service) Create(ctx context.Context, data map[string]interface{}) (interface{}, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("创建数据不能为空")
	}
	result, err := s.repo.Create(ctx, data)
	if err != nil {
		return nil, err
	}
	_ = s.queue.Publish(ctx, "{{.Snake}}.created", result)
	return result, nil
}

func (s *{{.Snake}}Service) Update(ctx context.Context, id string, data map[string]interface{}) (interface{}, error) {
	if id == "" {
		return nil, fmt.Errorf("ID 不能为空")
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("更新数据不能为空")
	}
	if err := s.repo.Update(ctx, id, data); err != nil {
		return nil, err
	}
	result, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	_ = s.queue.Publish(ctx, "{{.Snake}}.updated", result)
	return result, nil
}

func (s *{{.Snake}}Service) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("ID 不能为空")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.queue.Publish(ctx, "{{.Snake}}.deleted", map[string]string{"id": id})
	return nil
}
`
	dest := filepath.Join(cg.ProjectRoot, "internal/service", cg.SnakeName+".go")
	return cg.writeTemplate("service", tmpl, dest, map[string]string{
		"Pascal": cg.PascalName, "Snake": cg.SnakeName,
		"Component": cg.ComponentName, "Module": cg.ModuleName,
	})
}

func (cg *ComponentGenerator) GenerateRepository() error {
	if err := cg.ensureDir("internal/repository"); err != nil {
		return err
	}
	const tmpl = `package repository

import (
	"context"
	"fmt"
	"time"

	"{{.Module}}/internal/models"
	"{{.Module}}/pkg/cache"

	"github.com/samber/do/v2"
	"gorm.io/gorm"
)

type {{.Pascal}}Repository interface {
	FindAll(ctx context.Context) ([]interface{}, error)
	FindByID(ctx context.Context, id string) (interface{}, error)
	Create(ctx context.Context, data map[string]interface{}) (interface{}, error)
	Update(ctx context.Context, id string, data map[string]interface{}) error
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int64, error)
}

type {{.Snake}}Repository struct {
	db    *gorm.DB
	cache cache.Cache
}

func New{{.Pascal}}Repository(i do.Injector) ({{.Pascal}}Repository, error) {
	return &{{.Snake}}Repository{
		db:    do.MustInvoke[*gorm.DB](i),
		cache: do.MustInvoke[cache.Cache](i),
	}, nil
}

func (r *{{.Snake}}Repository) FindAll(ctx context.Context) ([]interface{}, error) {
	var items []models.{{.Pascal}}
	if err := r.db.WithContext(ctx).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("查询 {{.Component}} 列表失败: %w", err)
	}
	result := make([]interface{}, len(items))
	for i, item := range items {
		result[i] = item
	}
	return result, nil
}

func (r *{{.Snake}}Repository) FindByID(ctx context.Context, id string) (interface{}, error) {
	cacheKey := fmt.Sprintf("{{.Snake}}:%s", id)
	var item models.{{.Pascal}}
	if err := r.cache.GetJSON(ctx, cacheKey, &item); err == nil {
		return &item, nil
	}
	if err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("查询 {{.Component}} 失败: %w", err)
	}
	_ = r.cache.SetJSON(ctx, cacheKey, item, 5*time.Minute)
	return &item, nil
}

func (r *{{.Snake}}Repository) Create(ctx context.Context, data map[string]interface{}) (interface{}, error) {
	if err := r.db.WithContext(ctx).Model(&models.{{.Pascal}}{}).Create(data).Error; err != nil {
		return nil, fmt.Errorf("创建 {{.Component}} 失败: %w", err)
	}
	return data, nil
}

func (r *{{.Snake}}Repository) Update(ctx context.Context, id string, data map[string]interface{}) error {
	if err := r.db.WithContext(ctx).Model(&models.{{.Pascal}}{}).Where("id = ?", id).Updates(data).Error; err != nil {
		return fmt.Errorf("更新 {{.Component}} 失败: %w", err)
	}
	_ = r.cache.Delete(ctx, fmt.Sprintf("{{.Snake}}:%s", id))
	return nil
}

func (r *{{.Snake}}Repository) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&models.{{.Pascal}}{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("删除 {{.Component}} 失败: %w", err)
	}
	_ = r.cache.Delete(ctx, fmt.Sprintf("{{.Snake}}:%s", id))
	return nil
}

func (r *{{.Snake}}Repository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.{{.Pascal}}{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计 {{.Component}} 失败: %w", err)
	}
	return count, nil
}
`
	dest := filepath.Join(cg.ProjectRoot, "internal/repository", cg.SnakeName+".go")
	return cg.writeTemplate("repository", tmpl, dest, map[string]string{
		"Pascal": cg.PascalName, "Snake": cg.SnakeName,
		"Component": cg.ComponentName, "Module": cg.ModuleName,
	})
}

func (cg *ComponentGenerator) GenerateModel() error {
	if err := cg.ensureDir("internal/models"); err != nil {
		return err
	}
	const tmpl = `package models

import "time"

type {{.Pascal}} struct {
	ID        string     {{.BT}}gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"{{.BT}}
	CreatedAt time.Time  {{.BT}}gorm:"column:created_at;autoCreateTime" json:"created_at"{{.BT}}
	UpdatedAt time.Time  {{.BT}}gorm:"column:updated_at;autoUpdateTime" json:"updated_at"{{.BT}}
	DeletedAt *time.Time {{.BT}}gorm:"column:deleted_at;index" json:"deleted_at,omitempty"{{.BT}}
}

func (m *{{.Pascal}}) TableName() string {
	return "{{.Snake}}s"
}

type {{.Pascal}}Create struct{}
type {{.Pascal}}Update struct{}

type {{.Pascal}}Response struct {
	ID        string    {{.BT}}json:"id"{{.BT}}
	CreatedAt time.Time {{.BT}}json:"created_at"{{.BT}}
	UpdatedAt time.Time {{.BT}}json:"updated_at"{{.BT}}
}

func (m *{{.Pascal}}) ToResponse() *{{.Pascal}}Response {
	return &{{.Pascal}}Response{
		ID:        m.ID,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
`
	dest := filepath.Join(cg.ProjectRoot, "internal/models", cg.SnakeName+".go")
	return cg.writeTemplate("model", tmpl, dest, map[string]string{
		"Pascal": cg.PascalName, "Snake": cg.SnakeName,
		"Component": cg.ComponentName, "BT": "`",
	})
}

func (cg *ComponentGenerator) GenerateAll() error {
	return cg.runSteps([]step{
		{"Model", cg.GenerateModel},
		{"Repository", cg.GenerateRepository},
		{"Service", cg.GenerateService},
		{"Handler", cg.GenerateHandler},
	})
}

func (cg *ComponentGenerator) GenerateAndRegister() error {
	return cg.runSteps([]step{
		{"Model", cg.GenerateModel},
		{"Repository", cg.GenerateRepository},
		{"Service", cg.GenerateService},
		{"Handler", cg.GenerateHandler},
		{"注册 Repository", cg.AppendRepositoryToPackage},
		{"注册 Service", cg.AppendServiceToPackage},
		{"注册 Handler", cg.AppendHandlerToPackage},
	})
}

func (cg *ComponentGenerator) AppendHandlerToPackage() error {
	return cg.appendToPackage("internal/handler/package.go",
		fmt.Sprintf("do.Lazy(New%sHandler)", cg.PascalName))
}

func (cg *ComponentGenerator) AppendServiceToPackage() error {
	return cg.appendToPackage("internal/service/package.go",
		fmt.Sprintf("do.Lazy(New%sService)", cg.PascalName))
}

func (cg *ComponentGenerator) AppendRepositoryToPackage() error {
	return cg.appendToPackage("internal/repository/package.go",
		fmt.Sprintf("do.Lazy(New%sRepository)", cg.PascalName))
}

func (cg *ComponentGenerator) AppendHandlerToRoutes() error {
	if err := cg.appendHandlerFieldToServer(); err != nil {
		return err
	}
	return cg.appendRouteGroupToRoutes()
}

func (cg *ComponentGenerator) appendHandlerFieldToServer() error {
	fullPath := filepath.Join(cg.ProjectRoot, "internal/server/server.go")
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("读取 server.go 失败: %w", err)
	}
	content := string(data)

	fieldName := fmt.Sprintf("%sHandler", cg.SnakeName)
	if strings.Contains(content, fieldName) {
		return nil
	}
	content = appendAfterLastMatch(content, "Handler handler.",
		fmt.Sprintf("\t%sHandler handler.%sHandler", cg.SnakeName, cg.PascalName))
	content = appendAfterLastMatch(content, "do.MustInvoke[handler.",
		fmt.Sprintf("\t\t%sHandler: do.MustInvoke[handler.%sHandler](i),", cg.SnakeName, cg.PascalName))
	return os.WriteFile(fullPath, []byte(content), 0o644)
}

func (cg *ComponentGenerator) appendRouteGroupToRoutes() error {
	fullPath := filepath.Join(cg.ProjectRoot, "internal/server/routes.go")
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("读取 routes.go 失败: %w", err)
	}
	content := string(data)

	if strings.Contains(content, fmt.Sprintf("/%ss\"", cg.SnakeName)) {
		return nil
	}

	sn := cg.SnakeName
	pa := cg.PascalName
	routeBlock := fmt.Sprintf(`
		%ss := api.Group("/%ss")
		{
			%ss.GET("", s.%sHandler.List)
			%ss.GET("/:id", s.%sHandler.Get)
			%ss.POST("", s.%sHandler.Create)
			%ss.PUT("/:id", s.%sHandler.Update)
			%ss.DELETE("/:id", s.%sHandler.Delete)
		}`,
		sn, sn, sn, pa, sn, pa, sn, pa, sn, pa, sn, pa,
	)

	lines := strings.Split(content, "\n")
	insertIdx := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) == "}" {
			insertIdx = i
			break
		}
	}
	if insertIdx == -1 {
		return fmt.Errorf("routes.go 格式异常：未找到插入位置")
	}

	result := make([]string, 0, len(lines)+10)
	result = append(result, lines[:insertIdx]...)
	result = append(result, routeBlock)
	result = append(result, lines[insertIdx:]...)
	return os.WriteFile(fullPath, []byte(strings.Join(result, "\n")), 0o644)
}

type step struct {
	name string
	fn   func() error
}

func (cg *ComponentGenerator) runSteps(steps []step) error {
	for _, s := range steps {
		if err := s.fn(); err != nil {
			return fmt.Errorf("[%s] 失败: %w", s.name, err)
		}
	}
	return nil
}

func (cg *ComponentGenerator) ensureDir(dir string) error {
	if err := os.MkdirAll(filepath.Join(cg.ProjectRoot, dir), 0o755); err != nil {
		return fmt.Errorf("创建目录 %s 失败: %w", dir, err)
	}
	return nil
}

func (cg *ComponentGenerator) writeTemplate(name, tmpl, destPath string, data map[string]string) error {
	t, err := template.New(name).Parse(tmpl)
	if err != nil {
		return fmt.Errorf("解析 %s 模板失败: %w", name, err)
	}
	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("创建 %s 文件失败: %w", name, err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			// log or ignore
		}
	}()
	return t.Execute(f, data)
}

func (cg *ComponentGenerator) appendToPackage(relPath, newLine string) error {
	fullPath := filepath.Join(cg.ProjectRoot, relPath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("读取 %s 失败: %w", relPath, err)
	}
	content := string(data)
	if strings.Contains(content, newLine) {
		return nil
	}
	lastParen := strings.LastIndex(content, ")")
	if lastParen == -1 {
		return fmt.Errorf("%s 格式异常：未找到结束括号", relPath)
	}
	before := strings.TrimRight(content[:lastParen], " \t\n")
	return os.WriteFile(fullPath, []byte(before+"\n\t"+newLine+",\n"+content[lastParen:]), 0o644)
}

func (cg *ComponentGenerator) appendPackageToServerDI(pkg string) error {
	serverPkgPath := filepath.Join(cg.ProjectRoot, "internal/server/package.go")
	data, err := os.ReadFile(serverPkgPath)
	if err != nil {
		return fmt.Errorf("读取 server/package.go 失败: %w", err)
	}
	content := string(data)

	diLine := fmt.Sprintf("%s.Package,", pkg)
	if strings.Contains(content, diLine) {
		return nil
	}
	content = ensureImport(content, fmt.Sprintf(`"%s/internal/%s"`, cg.ModuleName, pkg))
	content = appendAfterLastMatch(content, "do.Package(", "\t"+diLine)
	return os.WriteFile(serverPkgPath, []byte(content), 0o644)
}
