package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kzzan/kz/pkg/utils"
)

type ProjectGenerator struct {
	ProjectName string
	ProjectPath string
	PascalName  string
	SnakeName   string
}

func NewProjectGenerator(projectName string) *ProjectGenerator {
	return &ProjectGenerator{
		ProjectName: projectName,
		ProjectPath: projectName,
		PascalName:  utils.ToPascalCase(projectName),
		SnakeName:   utils.ToSnakeCase(projectName),
	}
}

func (pg *ProjectGenerator) GenerateProject() error {
	if err := os.MkdirAll(pg.ProjectPath, 0o755); err != nil {
		return fmt.Errorf("创建项目目录失败: %w", err)
	}

	dirs := []string{
		"cmd",
		"internal/handler",
		"internal/service",
		"internal/repository",
		"internal/server",
		"internal/middleware",
		"internal/models",
		"internal/cron",
		"internal/consumer",
		"pkg/config",
		"pkg/database",
		"pkg/cache",
		"pkg/queue",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(pg.ProjectPath, dir), 0o755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %w", dir, err)
		}
	}

	files := map[string]string{
		"go.mod":                            pg.generateGoMod(),
		".gitignore":                        pg.generateGitignore(),
		"Makefile":                          pg.generateMakefile(),
		"README.md":                         pg.generateReadme(),
		".env.example":                      pg.generateEnvExample(),
		"docker-compose.yml":                pg.generateDockerCompose(),
		"cmd/main.go":                       pg.generateCmdMainGo(),
		"pkg/package.go":                    pg.generateBasePackage(),
		"pkg/config/config.go":              pg.generateConfigGo(),
		"pkg/database/database.go":          pg.generateDatabaseGo(),
		"pkg/cache/cache.go":                pg.generateCacheGo(),
		"pkg/queue/queue.go":                pg.generateQueueGo(),
		"internal/server/server.go":         pg.generateServerGo(),
		"internal/server/package.go":        pg.generateServerPackage(),
		"internal/server/routes.go":         pg.generateRoutesGo(),
		"internal/handler/package.go":       pg.generateHandlerPackage(),
		"internal/service/package.go":       pg.generateServicePackage(),
		"internal/repository/package.go":    pg.generateRepositoryPackage(),
		"internal/middleware/logger.go":     pg.generateMiddlewareLogger(),
		"internal/middleware/recovery.go":   pg.generateMiddlewareRecovery(),
		"internal/middleware/rate_limit.go": pg.generateMiddlewareRateLimit(),
		"internal/middleware/auth.go":       pg.generateMiddlewareAuth(),
		"internal/cron/package.go":          pg.generateCronPackage(),
		"internal/consumer/package.go":      pg.generateConsumerPackage(),
	}
	for filePath, content := range files {
		if content == "" {
			continue
		}
		fullPath := filepath.Join(pg.ProjectPath, filePath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return fmt.Errorf("创建文件目录失败: %w", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("创建文件 %s 失败: %w", filePath, err)
		}
	}

	return pg.generateDefaultComponents()
}

func (pg *ProjectGenerator) generateDefaultComponents() error {
	cg := &ComponentGenerator{
		ComponentName: "user",
		PascalName:    utils.ToPascalCase("user"),
		SnakeName:     utils.ToSnakeCase("user"),
		ProjectRoot:   pg.ProjectPath,
		ModuleName:    pg.SnakeName,
	}
	return cg.runSteps([]step{
		{"Model", cg.GenerateModel},
		{"Repository", cg.GenerateRepository},
		{"Service", cg.GenerateService},
		{"Handler", cg.GenerateHandler},
	})
}

// ── go.mod ────────────────────────────────────────────────────────────────────

func (pg *ProjectGenerator) generateGoMod() string {
	return fmt.Sprintf(`module %s

go 1.25

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/google/uuid v1.6.0
	github.com/redis/go-redis/v9 v9.5.1
	github.com/robfig/cron/v3 v3.0.1
	github.com/rs/zerolog v1.33.0
	github.com/samber/do/v2 v2.0.0
	github.com/spf13/viper v1.18.0
	gorm.io/driver/postgres v1.5.9
	gorm.io/gorm v1.25.10
)
`, pg.SnakeName)
}

// ── cmd/main.go ───────────────────────────────────────────────────────────────

func (pg *ProjectGenerator) generateCmdMainGo() string {
	return fmt.Sprintf(`package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"%s/internal/consumer"
	"%s/internal/cron"
	"%s/internal/server"
	"%s/pkg"

	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
)

func main() {
	injector := do.New(
		pkg.BasePackage,
		server.Package,
	)

	logger := do.MustInvoke[*zerolog.Logger](injector)
	srv    := do.MustInvoke[*server.Server](injector)

	logger.Info().Msg("Starting application")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if manager, err := do.Invoke[*consumer.Manager](injector); err == nil {
		manager.Start(ctx)
		logger.Info().Msg("Consumer manager started")
	}

	if scheduler, err := do.Invoke[*cron.Scheduler](injector); err == nil {
		scheduler.Start()
		logger.Info().Msg("Cron scheduler started")
		defer scheduler.Stop()
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- srv.Start()
	}()

	select {
	case err := <-serverErrors:
		logger.Error().Err(err).Msg("Server exited with error")
	case sig := <-sigChan:
		logger.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error().Err(err).Msg("Error during graceful shutdown")
		}
	}

	if _, err := injector.ShutdownOnSignals(); err != nil {
		logger.Error().Err(err).Msg("Error shutting down injector")
	}
}
`, pg.SnakeName, pg.SnakeName, pg.SnakeName, pg.SnakeName)
}

// ── pkg/package.go ────────────────────────────────────────────────────────────

func (pg *ProjectGenerator) generateBasePackage() string {
	return fmt.Sprintf(`package pkg

import (
	"os"

	"%s/pkg/cache"
	"%s/pkg/config"
	"%s/pkg/database"
	"%s/pkg/queue"

	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
)

var BasePackage = do.Package(
	do.Lazy(config.NewConfig),
	do.Lazy(NewLogger),
	do.Lazy(database.NewDatabase),
	do.Lazy(cache.NewCache),
	do.Lazy(queue.NewQueue),
)

func NewLogger(i do.Injector) (*zerolog.Logger, error) {
	cfg := do.MustInvoke[*config.Config](i)
	level, err := zerolog.ParseLevel(cfg.Logger.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	logger := zerolog.New(os.Stdout).
		Level(level).
		With().
		Timestamp().
		Str("app", cfg.App.Name).
		Logger()
	return &logger, nil
}
`, pg.SnakeName, pg.SnakeName, pg.SnakeName, pg.SnakeName)
}

// ── pkg/config/config.go ──────────────────────────────────────────────────────

func (pg *ProjectGenerator) generateConfigGo() string {
	return `package config

import (
	"github.com/samber/do/v2"
	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Logger   LoggerConfig
}

type AppConfig struct {
	Name        string
	Version     string
	Environment string
	Debug       bool
}

type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  int
	WriteTimeout int
}

type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime int
}

type RedisConfig struct {
	Host         string
	Port         int
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
}

type LoggerConfig struct {
	Level   string
	Format  string
	Output  string
	NoColor bool
}

func NewConfig(i do.Injector) (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	viper.SetDefault("SERVER_HOST", "0.0.0.0")
	viper.SetDefault("SERVER_PORT", 8080)
	viper.SetDefault("SERVER_READ_TIMEOUT", 30)
	viper.SetDefault("SERVER_WRITE_TIMEOUT", 30)

	viper.SetDefault("DATABASE_HOST", "localhost")
	viper.SetDefault("DATABASE_PORT", 5432)
	viper.SetDefault("DATABASE_SSL_MODE", "disable")
	viper.SetDefault("DATABASE_MAX_OPEN_CONNS", 25)
	viper.SetDefault("DATABASE_MAX_IDLE_CONNS", 5)
	viper.SetDefault("DATABASE_CONN_MAX_LIFETIME", 300)

	viper.SetDefault("REDIS_HOST", "localhost")
	viper.SetDefault("REDIS_PORT", 6379)
	viper.SetDefault("REDIS_DB", 0)
	viper.SetDefault("REDIS_POOL_SIZE", 10)
	viper.SetDefault("REDIS_MIN_IDLE_CONNS", 5)

	viper.SetDefault("LOGGER_LEVEL", "info")
	viper.SetDefault("LOGGER_FORMAT", "console")
	viper.SetDefault("LOGGER_OUTPUT", "stdout")
	viper.SetDefault("LOGGER_NO_COLOR", false)

	viper.SetDefault("APP_NAME", "app")
	viper.SetDefault("APP_VERSION", "1.0.0")
	viper.SetDefault("APP_ENVIRONMENT", "development")
	viper.SetDefault("APP_DEBUG", false)

	_ = viper.ReadInConfig()

	return &Config{
		App: AppConfig{
			Name:        viper.GetString("APP_NAME"),
			Version:     viper.GetString("APP_VERSION"),
			Environment: viper.GetString("APP_ENVIRONMENT"),
			Debug:       viper.GetBool("APP_DEBUG"),
		},
		Server: ServerConfig{
			Host:         viper.GetString("SERVER_HOST"),
			Port:         viper.GetInt("SERVER_PORT"),
			ReadTimeout:  viper.GetInt("SERVER_READ_TIMEOUT"),
			WriteTimeout: viper.GetInt("SERVER_WRITE_TIMEOUT"),
		},
		Database: DatabaseConfig{
			Host:            viper.GetString("DATABASE_HOST"),
			Port:            viper.GetInt("DATABASE_PORT"),
			User:            viper.GetString("DATABASE_USER"),
			Password:        viper.GetString("DATABASE_PASSWORD"),
			Database:        viper.GetString("DATABASE_DATABASE"),
			SSLMode:         viper.GetString("DATABASE_SSL_MODE"),
			MaxOpenConns:    viper.GetInt("DATABASE_MAX_OPEN_CONNS"),
			MaxIdleConns:    viper.GetInt("DATABASE_MAX_IDLE_CONNS"),
			ConnMaxLifetime: viper.GetInt("DATABASE_CONN_MAX_LIFETIME"),
		},
		Redis: RedisConfig{
			Host:         viper.GetString("REDIS_HOST"),
			Port:         viper.GetInt("REDIS_PORT"),
			Password:     viper.GetString("REDIS_PASSWORD"),
			DB:           viper.GetInt("REDIS_DB"),
			PoolSize:     viper.GetInt("REDIS_POOL_SIZE"),
			MinIdleConns: viper.GetInt("REDIS_MIN_IDLE_CONNS"),
		},
		Logger: LoggerConfig{
			Level:   viper.GetString("LOGGER_LEVEL"),
			Format:  viper.GetString("LOGGER_FORMAT"),
			Output:  viper.GetString("LOGGER_OUTPUT"),
			NoColor: viper.GetBool("LOGGER_NO_COLOR"),
		},
	}, nil
}
`
}

// ── pkg/database/database.go ──────────────────────────────────────────────────

func (pg *ProjectGenerator) generateDatabaseGo() string {
	return fmt.Sprintf(`package database

import (
	"fmt"
	"time"

	"%s/pkg/config"

	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func NewDatabase(i do.Injector) (*gorm.DB, error) {
	cfg    := do.MustInvoke[*config.Config](i)
	logger := do.MustInvoke[*zerolog.Logger](i)

	dsn := fmt.Sprintf(
		"host=%%s port=%%d user=%%s password=%%s dbname=%%s sslmode=%%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Database,
		cfg.Database.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %%w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取 sql.DB 失败: %%w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.Database.ConnMaxLifetime) * time.Second)

	logger.Info().
		Str("host", cfg.Database.Host).
		Int("port", cfg.Database.Port).
		Str("database", cfg.Database.Database).
		Msg("Database connected")

	return db, nil
}
`, pg.SnakeName)
}

// ── pkg/cache/cache.go ────────────────────────────────────────────────────────

func (pg *ProjectGenerator) generateCacheGo() string {
	return fmt.Sprintf(`package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"%s/pkg/config"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
)

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	GetJSON(ctx context.Context, key string, v interface{}) error
	Set(ctx context.Context, key string, value string, expiration time.Duration) error
	SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	TTL(ctx context.Context, key string) (time.Duration, error)
}

type redisCache struct {
	client *redis.Client
	logger *zerolog.Logger
}

func NewCache(i do.Injector) (Cache, error) {
	cfg    := do.MustInvoke[*config.Config](i)
	logger := do.MustInvoke[*zerolog.Logger](i)

	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%%s:%%d", cfg.Redis.Host, cfg.Redis.Port),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("连接 Redis 失败: %%w", err)
	}

	logger.Info().
		Str("host", cfg.Redis.Host).
		Int("port", cfg.Redis.Port).
		Msg("Redis cache connected")

	return &redisCache{client: client, logger: logger}, nil
}

func (c *redisCache) Get(ctx context.Context, key string) (string, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		c.logger.Error().Err(err).Str("key", key).Msg("cache get failed")
		return "", err
	}
	return val, nil
}

func (c *redisCache) GetJSON(ctx context.Context, key string, v interface{}) error {
	val, err := c.Get(ctx, key)
	if err != nil {
		return err
	}
	if val == "" {
		return redis.Nil
	}
	return json.Unmarshal([]byte(val), v)
}

func (c *redisCache) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	if err := c.client.Set(ctx, key, value, expiration).Err(); err != nil {
		c.logger.Error().Err(err).Str("key", key).Msg("cache set failed")
		return err
	}
	return nil
}

func (c *redisCache) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.Set(ctx, key, string(data), expiration)
}

func (c *redisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *redisCache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, key).Result()
	return n > 0, err
}

func (c *redisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, key).Result()
}

func (c *redisCache) Shutdown() error {
	return c.client.Close()
}
`, pg.SnakeName)
}

// ── pkg/queue/queue.go ────────────────────────────────────────────────────────

func (pg *ProjectGenerator) generateQueueGo() string {
	return fmt.Sprintf(`package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"%s/pkg/config"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
)

type Message struct {
	ID         string    `+"`"+`json:"id"`+"`"+`
	Topic      string    `+"`"+`json:"topic"`+"`"+`
	Payload    string    `+"`"+`json:"payload"`+"`"+`
	CreatedAt  time.Time `+"`"+`json:"created_at"`+"`"+`
	RetryCount int       `+"`"+`json:"retry_count"`+"`"+`
}

type MessageHandler func(ctx context.Context, msg *Message) error

type Queue interface {
	Publish(ctx context.Context, topic string, payload interface{}) error
	Subscribe(ctx context.Context, topic string, handler MessageHandler) error
	GetLength(ctx context.Context, topic string) (int64, error)
}

type redisQueue struct {
	client *redis.Client
	logger *zerolog.Logger
}

func NewQueue(i do.Injector) (Queue, error) {
	cfg    := do.MustInvoke[*config.Config](i)
	logger := do.MustInvoke[*zerolog.Logger](i)

	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%%s:%%d", cfg.Redis.Host, cfg.Redis.Port),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("连接 Redis 失败: %%w", err)
	}

	logger.Info().
		Str("host", cfg.Redis.Host).
		Int("port", cfg.Redis.Port).
		Msg("Redis queue connected")

	return &redisQueue{client: client, logger: logger}, nil
}

func (q *redisQueue) Publish(ctx context.Context, topic string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	msg := &Message{
		ID:        fmt.Sprintf("%%d", time.Now().UnixNano()),
		Topic:     topic,
		Payload:   string(data),
		CreatedAt: time.Now(),
	}
	msgData, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: fmt.Sprintf("queue:%%s", topic),
		Values: map[string]interface{}{"data": string(msgData)},
	}).Result()
	if err != nil {
		q.logger.Error().Err(err).Str("topic", topic).Msg("publish failed")
	}
	return err
}

func (q *redisQueue) Subscribe(ctx context.Context, topic string, handler MessageHandler) error {
	streamKey := fmt.Sprintf("queue:%%s", topic)
	group     := fmt.Sprintf("group:%%s", topic)
	consumer  := fmt.Sprintf("consumer:%%d", time.Now().UnixNano())

	_ = q.client.XGroupCreateMkStream(ctx, streamKey, group, "0").Err()

	q.logger.Info().Str("topic", topic).Str("group", group).Msg("queue subscription started")

	for {
		select {
		case <-ctx.Done():
			q.logger.Info().Str("topic", topic).Msg("queue subscription cancelled")
			return nil
		default:
			messages, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    group,
				Consumer: consumer,
				Streams:  []string{streamKey, ">"},
				Count:    1,
				Block:    time.Second,
			}).Result()
			if err != nil && err != redis.Nil {
				q.logger.Error().Err(err).Str("topic", topic).Msg("read failed")
				time.Sleep(time.Second)
				continue
			}
			for _, stream := range messages {
				for _, m := range stream.Messages {
					if raw, ok := m.Values["data"].(string); ok {
						var qMsg Message
						if err := json.Unmarshal([]byte(raw), &qMsg); err != nil {
							q.logger.Error().Err(err).Str("id", m.ID).Msg("unmarshal failed")
						} else if err := handler(ctx, &qMsg); err != nil {
							q.logger.Warn().Err(err).Str("id", m.ID).Msg("handler failed")
						}
					}
					_ = q.client.XAck(ctx, streamKey, group, m.ID).Err()
				}
			}
		}
	}
}

func (q *redisQueue) GetLength(ctx context.Context, topic string) (int64, error) {
	return q.client.XLen(ctx, fmt.Sprintf("queue:%%s", topic)).Result()
}

func (q *redisQueue) Shutdown() error {
	return q.client.Close()
}
`, pg.SnakeName)
}

// ── internal/server/server.go ─────────────────────────────────────────────────

func (pg *ProjectGenerator) generateServerGo() string {
	return fmt.Sprintf(`package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"%s/internal/handler"
	"%s/internal/middleware"
	"%s/pkg/cache"
	"%s/pkg/config"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
)

type Server struct {
	config      *config.Config
	logger      *zerolog.Logger
	engine      *gin.Engine
	server      *http.Server
	userHandler handler.UserHandler
}

func NewServer(i do.Injector) (*Server, error) {
	cfg    := do.MustInvoke[*config.Config](i)
	logger := do.MustInvoke[*zerolog.Logger](i)
	c      := do.MustInvoke[cache.Cache](i)

	s := &Server{
		config:      cfg,
		logger:      logger,
		userHandler: do.MustInvoke[handler.UserHandler](i),
	}

	s.engine = gin.New()
	s.engine.Use(
		middleware.Logger(logger),
		middleware.Recovery(logger),
		middleware.RateLimit(c, logger, 100),
	)
	s.setupRoutes(logger)

	s.server = &http.Server{
		Addr:         fmt.Sprintf("%%s:%%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      s.engine,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.Info().
		Str("host", cfg.Server.Host).
		Int("port", cfg.Server.Port).
		Msg("Server initialized")

	return s, nil
}

func (s *Server) Start() error {
	s.logger.Info().Msg("HTTP server starting")
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info().Msg("Shutting down HTTP server")
	return s.server.Shutdown(ctx)
}
`, pg.ProjectName, pg.ProjectName, pg.ProjectName, pg.ProjectName)
}

// ── internal/server/routes.go ─────────────────────────────────────────────────

func (pg *ProjectGenerator) generateRoutesGo() string {
	return fmt.Sprintf(`package server

import (
	"net/http"

	"%s/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func (s *Server) setupRoutes(logger *zerolog.Logger) {
	s.engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := s.engine.Group("/api")
	{
		public := api.Group("")
		{
			public.POST("/login",    s.userHandler.Get)
			public.POST("/register", s.userHandler.Create)
		}

		authed := api.Group("", middleware.Auth(logger))
		{
			users := authed.Group("/users")
			{
				users.GET("",        s.userHandler.List)
				users.GET("/:id",    s.userHandler.Get)
				users.POST("",       s.userHandler.Create)
				users.PUT("/:id",    s.userHandler.Update)
				users.DELETE("/:id", s.userHandler.Delete)
			}

			orders := authed.Group("/orders")
			{
				orders.GET("",        s.userHandler.List)
				orders.GET("/:id",    s.userHandler.Get)
				orders.POST("",       s.userHandler.Create)
				orders.PUT("/:id",    s.userHandler.Update)
				orders.DELETE("/:id", s.userHandler.Delete)
			}
		}
	}
}
`, pg.ProjectName)
}

// ── internal/server/package.go ────────────────────────────────────────────────

func (pg *ProjectGenerator) generateServerPackage() string {
	return fmt.Sprintf(`package server

import (
	"%s/internal/handler"
	"%s/internal/repository"
	"%s/internal/service"

	"github.com/samber/do/v2"
)

var Package = do.Package(
	repository.Package,
	service.Package,
	handler.Package,
	do.Lazy(NewServer),
)
`, pg.ProjectName, pg.ProjectName, pg.ProjectName)
}

// ── internal/handler/package.go ───────────────────────────────────────────────

func (pg *ProjectGenerator) generateHandlerPackage() string {
	return `package handler

import "github.com/samber/do/v2"

var Package = do.Package(
	do.Lazy(NewUserHandler),
)
`
}

// ── internal/service/package.go ───────────────────────────────────────────────

func (pg *ProjectGenerator) generateServicePackage() string {
	return `package service

import "github.com/samber/do/v2"

var Package = do.Package(
	do.Lazy(NewUserService),
)
`
}

// ── internal/repository/package.go ───────────────────────────────────────────

func (pg *ProjectGenerator) generateRepositoryPackage() string {
	return `package repository

import "github.com/samber/do/v2"

var Package = do.Package(
	do.Lazy(NewUserRepository),
)
`
}

// ── internal/cron/package.go ──────────────────────────────────────────────────

func (pg *ProjectGenerator) generateCronPackage() string {
	return `package cron

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
	s.logger.Info().Msg("Cron scheduler started")
}

func (s *Scheduler) Stop() {
	s.c.Stop()
	s.logger.Info().Msg("Cron scheduler stopped")
}

func (s *Scheduler) Shutdown() error {
	s.Stop()
	return nil
}

var Package = do.Package(
	do.Lazy(NewScheduler),
)
`
}

// ── internal/consumer/package.go ─────────────────────────────────────────────

func (pg *ProjectGenerator) generateConsumerPackage() string {
	return fmt.Sprintf(`package consumer

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
`, pg.SnakeName)
}

// ── internal/middleware ───────────────────────────────────────────────────────

func (pg *ProjectGenerator) generateMiddlewareLogger() string {
	return `package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func Logger(logger *zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path  := c.Request.URL.Path
		c.Next()
		logger.Info().
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", c.Writer.Status()).
			Dur("latency", time.Since(start)).
			Str("ip", c.ClientIP()).
			Msg("request")
	}
}
`
}

func (pg *ProjectGenerator) generateMiddlewareRecovery() string {
	return `package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func Recovery(logger *zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error().
					Interface("error", err).
					Str("path", c.Request.URL.Path).
					Msg("panic recovered")
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
			}
		}()
		c.Next()
	}
}
`
}

func (pg *ProjectGenerator) generateMiddlewareRateLimit() string {
	return fmt.Sprintf(`package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"%s/pkg/cache"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func RateLimit(c cache.Cache, logger *zerolog.Logger, maxReqs int) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		key     := fmt.Sprintf("rate:%%s", ctx.ClientIP())
		cacheCtx := context.Background()

		val, err := c.Get(cacheCtx, key)
		if err != nil {
			logger.Warn().Err(err).Msg("rate limit cache error")
			ctx.Next()
			return
		}

		if val == "" {
			_ = c.Set(cacheCtx, key, "1", time.Minute)
			ctx.Next()
			return
		}

		count, _ := strconv.Atoi(val)
		if count >= maxReqs {
			logger.Warn().
				Str("ip", ctx.ClientIP()).
				Int("count", count).
				Msg("rate limit exceeded")
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests",
			})
			return
		}

		_ = c.Set(cacheCtx, key, strconv.Itoa(count+1), time.Minute)
		ctx.Next()
	}
}
`, pg.ProjectName)
}

func (pg *ProjectGenerator) generateMiddlewareAuth() string {
	return `package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func Auth(logger *zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing authorization header",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization format",
			})
			return
		}

		token := parts[1]
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "empty token",
			})
			return
		}

		// TODO: 替换为真实 JWT 验证逻辑
		logger.Debug().Str("token_prefix", token[:min(8, len(token))]).Msg("auth passed")
		c.Next()
	}
}
`
}

// ── .gitignore / Makefile / README / .env.example / docker-compose.yml ───────

func (pg *ProjectGenerator) generateGitignore() string {
	return `# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out

# Build output
bin/
dist/

# Dependencies
vendor/

# IDE
.idea/
.vscode/
*.swp

# OS
.DS_Store

# Environment
.env
.env.local

# Logs
*.log

# Database
*.db
*.sqlite
`
}

func (pg *ProjectGenerator) generateMakefile() string {
	return fmt.Sprintf(`.PHONY: help deps build run test clean watch-run

help:
	@echo "可用命令:"
	@echo "  make deps       安装依赖"
	@echo "  make build      编译项目"
	@echo "  make run        运行项目"
	@echo "  make test       运行测试"
	@echo "  make clean      清理输出"
	@echo "  make watch-run  热重载运行（需要 air）"

deps:
	go mod download
	go mod tidy

deps-tools:
	go install github.com/air-verse/air@latest

build:
	mkdir -p bin
	go build -o bin/%s ./cmd

run:
	go run ./cmd/main.go

watch-run:
	air

test:
	go test -v -cover ./...

clean:
	rm -rf bin/
	go clean
`, pg.ProjectName)
}

func (pg *ProjectGenerator) generateEnvExample() string {
	return fmt.Sprintf(`# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_READ_TIMEOUT=30
SERVER_WRITE_TIMEOUT=30

# Database
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_USER=%s
DATABASE_PASSWORD=%s
DATABASE_DATABASE=%s
DATABASE_SSL_MODE=disable
DATABASE_MAX_OPEN_CONNS=25
DATABASE_MAX_IDLE_CONNS=5
DATABASE_CONN_MAX_LIFETIME=300

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_POOL_SIZE=10
REDIS_MIN_IDLE_CONNS=5

# Logger
LOGGER_LEVEL=info
LOGGER_FORMAT=console
LOGGER_OUTPUT=stdout
LOGGER_NO_COLOR=false

# App
APP_NAME=%s
APP_VERSION=1.0.0
APP_ENVIRONMENT=development
APP_DEBUG=false
`, pg.ProjectName, pg.ProjectName, pg.ProjectName, pg.ProjectName)
}

func (pg *ProjectGenerator) generateDockerCompose() string {
	return fmt.Sprintf(`services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  postgres:
    image: postgres:18-alpine
    environment:
      POSTGRES_USER: %s
      POSTGRES_PASSWORD: %s
      POSTGRES_DB: %s
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U %s"]
      interval: 10s
      timeout: 5s
      retries: 5

  pgadmin:
    image: dpage/pgadmin4:latest
    environment:
      PGADMIN_DEFAULT_EMAIL: admin@example.com
      PGADMIN_DEFAULT_PASSWORD: admin
    ports:
      - "8081:80"
    depends_on:
      - postgres
    profiles:
      - tools

volumes:
  postgres_data:
  redis_data:
`, pg.ProjectName, pg.ProjectName, pg.ProjectName, pg.ProjectName)
}

func (pg *ProjectGenerator) generateReadme() string {
	bt := "`"
	cb := bt + bt + bt
	return fmt.Sprintf(`# %s

基于 [samber/do](https://github.com/samber/do) 依赖注入的 Go API 项目。

## 技术栈

- **Gin** - HTTP 框架
- **samber/do v2** - 类型安全的依赖注入
- **GORM + PostgreSQL** - ORM 与数据库
- **Redis** - 缓存 & 消息队列
- **zerolog** - 结构化日志
- **Viper** - 配置管理

## 快速开始

%sbash
docker compose up -d
make deps
make run
%s

## 生成新组件

%sbash
kz new handler order
kz new service order
kz new repo order
kz new model order
kz new all order
%s

## 项目结构

%s
%s/
├── cmd/main.go
├── internal/
│   ├── handler/
│   ├── service/
│   ├── repository/
│   └── server/
├── pkg/
│   ├── config/
│   ├── database/
│   ├── cache/
│   ├── queue/
│   └── models/
├── docker-compose.yml
├── Makefile
└── go.mod
%s

## 许可证

MIT License
`,
		pg.PascalName,
		cb, cb,
		cb, cb,
		cb, pg.ProjectName, cb,
	)
}
