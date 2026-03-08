package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/kzzan/kz/pkg/utils"
)

type ProjectTemplateData struct {
	ProjectName string
	PascalName  string
	SnakeName   string
}

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

func (pg *ProjectGenerator) data() ProjectTemplateData {
	return ProjectTemplateData{
		ProjectName: pg.ProjectName,
		PascalName:  pg.PascalName,
		SnakeName:   pg.SnakeName,
	}
}

// render 从 embed.FS 渲染 project 模板
func (pg *ProjectGenerator) render(tmplPath string) (string, error) {
	tmpl, err := template.New(filepath.Base(tmplPath)).
		ParseFS(templateFS, tmplPath)
	if err != nil {
		return "", fmt.Errorf("解析模板 %s 失败: %w", tmplPath, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, pg.data()); err != nil {
		return "", fmt.Errorf("渲染模板 %s 失败: %w", tmplPath, err)
	}
	return buf.String(), nil
}

func (pg *ProjectGenerator) writeFile(relPath, tmplPath string) error {
	content, err := pg.render(tmplPath)
	if err != nil {
		return err
	}
	fullPath := filepath.Join(pg.ProjectPath, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	return os.WriteFile(fullPath, []byte(content), 0o644)
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
		"pkg/pagination",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(pg.ProjectPath, dir), 0o755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %w", dir, err)
		}
	}

	// relPath -> tmplPath
	files := map[string]string{
		"go.mod":                                       "templates/project/go.mod.tmpl",
		".gitignore":                                   "templates/project/gitignore.tmpl",
		"Makefile":                                     "templates/project/makefile.tmpl",
		"README.md":                                    "templates/project/readme.tmpl",
		".env.example":                                 "templates/project/env_example.tmpl",
		"docker-compose.yml":                           "templates/project/docker_compose.tmpl",
		"cmd/main.go":                                  "templates/project/cmd_main.go.tmpl",
		"pkg/package.go":                               "templates/project/pkg_package.go.tmpl",
		"pkg/config/config.go":                         "templates/project/pkg_config.go.tmpl",
		"pkg/database/database.go":                     "templates/project/pkg_database.go.tmpl",
		"pkg/pagination/pagination.go":                 "templates/project/pkg_pagination.go.tmpl",
		"pkg/cache/cache.go":                           "templates/project/pkg_cache.go.tmpl",
		"pkg/queue/queue.go":                           "templates/project/pkg_queue.go.tmpl",
		"internal/server/server.go":                    "templates/project/server.go.tmpl",
		"internal/server/package.go":                   "templates/project/server_package.go.tmpl",
		"internal/server/routes.go":                    "templates/project/server_routes.go.tmpl",
		"internal/handler/package.go":                  "templates/project/handler_package.go.tmpl",
		"internal/service/package.go":                  "templates/project/service_package.go.tmpl",
		"internal/repository/package.go":               "templates/project/repository_package.go.tmpl",
		"internal/middleware/logger_middleware.go":     "templates/project/middleware_logger.go.tmpl",
		"internal/middleware/recovery_middleware.go":   "templates/project/middleware_recovery.go.tmpl",
		"internal/middleware/rate_limit_middleware.go": "templates/project/middleware_rate_limit.go.tmpl",
		"internal/middleware/auth_middleware.go":       "templates/project/middleware_auth.go.tmpl",
		"internal/cron/package.go":                     "templates/project/cron_package.go.tmpl",
		"internal/consumer/package.go":                 "templates/project/consumer_package.go.tmpl",
	}

	for relPath, tmplPath := range files {
		if err := pg.writeFile(relPath, tmplPath); err != nil {
			return fmt.Errorf("生成 %s 失败: %w", relPath, err)
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
