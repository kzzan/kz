package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/kzzan/kz/pkg/utils"
)

type ComponentTemplateData struct {
	Pascal    string
	Snake     string
	Component string
	Module    string
}

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

func (cg *ComponentGenerator) data() ComponentTemplateData {
	return ComponentTemplateData{
		Pascal:    cg.PascalName,
		Snake:     cg.SnakeName,
		Component: cg.ComponentName,
		Module:    cg.ModuleName,
	}
}

func (cg *ComponentGenerator) render(tmplPath string) (string, error) {
	tmpl, err := template.New(filepath.Base(tmplPath)).
		ParseFS(templateFS, tmplPath)
	if err != nil {
		return "", fmt.Errorf("解析模板 %s 失败: %w", tmplPath, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cg.data()); err != nil {
		return "", fmt.Errorf("渲染模板 %s 失败: %w", tmplPath, err)
	}
	return buf.String(), nil
}

func (cg *ComponentGenerator) renderAndWrite(tmplPath, outRelPath string) error {
	content, err := cg.render(tmplPath)
	if err != nil {
		return err
	}
	fullPath := filepath.Join(cg.ProjectRoot, outRelPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	return os.WriteFile(fullPath, []byte(content), 0o644)
}

func (cg *ComponentGenerator) GenerateModel() error {
	return cg.renderAndWrite(
		"templates/component/model.go.tmpl",
		"internal/models/"+cg.SnakeName+"_model.go",
	)
}

func (cg *ComponentGenerator) GenerateMiddleware() error {
	return cg.renderAndWrite(
		"templates/component/middleware.go.tmpl",
		"internal/middleware/"+cg.SnakeName+"_middleware.go",
	)
}

func (cg *ComponentGenerator) GenerateRepository() error {
	return cg.renderAndWrite(
		"templates/component/repository.go.tmpl",
		"internal/repository/"+cg.SnakeName+"_repo.go",
	)
}

func (cg *ComponentGenerator) GenerateService() error {
	return cg.renderAndWrite(
		"templates/component/service.go.tmpl",
		"internal/service/"+cg.SnakeName+"_service.go",
	)
}

func (cg *ComponentGenerator) GenerateHandler() error {
	return cg.renderAndWrite(
		"templates/component/handler.go.tmpl",
		"internal/handler/"+cg.SnakeName+"_handler.go",
	)
}

func (cg *ComponentGenerator) GenerateCron() error {
	return cg.renderAndWrite(
		"templates/component/cron.go.tmpl",
		"internal/cron/"+cg.SnakeName+"_job.go",
	)
}

func (cg *ComponentGenerator) GenerateConsumer() error {
	return cg.renderAndWrite(
		"templates/component/consumer.go.tmpl",
		"internal/consumer/"+cg.SnakeName+"_consumer.go",
	)
}

func (cg *ComponentGenerator) GenerateConsumerAndRegister() error {
	return cg.runSteps([]step{
		{"生成 Consumer 文件", cg.GenerateConsumer},
		{"确保 consumer/package.go", cg.EnsureConsumerPackage},
		{"注册到 Manager", cg.AppendConsumerToManager},
		{"注册 consumer.Package 到 server", cg.AppendConsumerPackageToServer},
	})
}

func (cg *ComponentGenerator) EnsureConsumerPackage() error {
	pkgPath := filepath.Join(cg.ProjectRoot, "internal/consumer/package.go")
	if _, err := os.Stat(pkgPath); err == nil {
		return nil
	}
	return cg.renderAndWrite("templates/component/consumer_package.go.tmpl", "internal/consumer/package.go")
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

	appendStmt := fmt.Sprintf("\tm.consumers = append(m.consumers, new%sConsumer(logger, q))", cg.PascalName)

	placeholder := "// CONSUMERS_PLACEHOLDER"
	if strings.Contains(content, placeholder) {
		content = strings.Replace(content,
			placeholder,
			placeholder+"\n"+appendStmt,
			1,
		)
	} else {
		content = appendBeforeLastFuncClose(content, "func NewManager", appendStmt)
	}

	content = strings.Replace(content, "\n\t_ = q\n", "\n", 1)
	return os.WriteFile(pkgPath, []byte(content), 0o644)
}

func (cg *ComponentGenerator) EnsureCronPackage() error {
	pkgPath := filepath.Join(cg.ProjectRoot, "internal/cron/package.go")
	if _, err := os.Stat(pkgPath); err == nil {
		return nil
	}
	return cg.renderAndWrite("templates/component/cron_package.go.tmpl", "internal/cron/package.go")
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

	registerStmt := fmt.Sprintf("\tnew%sJob(s.logger).Register(s.c)", cg.PascalName)
	placeholder := "// JOBS_PLACEHOLDER"
	if strings.Contains(content, placeholder) {
		content = strings.Replace(content,
			placeholder,
			placeholder+"\n"+registerStmt,
			1,
		)
	} else {
		content = appendBeforeLastFuncClose(content, "func (s *Scheduler) registerJobs", registerStmt)
	}

	return os.WriteFile(pkgPath, []byte(content), 0o644)
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

func (cg *ComponentGenerator) AppendCronPackageToServer() error {
	return cg.appendPackageToServerDI("cron")
}

func (cg *ComponentGenerator) AppendConsumerPackageToServer() error {
	return cg.appendPackageToServerDI("consumer")
}

func (cg *ComponentGenerator) AppendHandlerToRoutes() error {
	if err := cg.appendHandlerFieldToServer(); err != nil {
		return err
	}
	return cg.appendRouteGroupToRoutes()
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

func (cg *ComponentGenerator) GenerateCronAndRegister() error {
	return cg.runSteps([]step{
		{"生成 Cron Job 文件", cg.GenerateCron},
		{"确保 cron/package.go", cg.EnsureCronPackage},
		{"注册到 Scheduler", cg.AppendCronToScheduler},
		{"注册 cron.Package 到 server", cg.AppendCronPackageToServer},
	})
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

	content = appendAfterLastMatch(
		content,
		"Handler handler.",
		fmt.Sprintf("\t%sHandler handler.%sHandler", cg.SnakeName, cg.PascalName),
	)

	content = appendAfterLastMatch(
		content,
		"do.MustInvoke[handler.",
		fmt.Sprintf("\t\t%sHandler: do.MustInvoke[handler.%sHandler](i),", cg.SnakeName, cg.PascalName),
	)

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

	routeBlock := fmt.Sprintf(
		"\n\t\t%ss := api.Group(\"/%ss\")\n"+
			"\t\t{\n"+
			"\t\t\t%ss.GET(\"\",        s.%sHandler.List)\n"+
			"\t\t\t%ss.GET(\"/:id\",    s.%sHandler.Get)\n"+
			"\t\t\t%ss.POST(\"\",       s.%sHandler.Create)\n"+
			"\t\t\t%ss.PUT(\"/:id\",    s.%sHandler.Update)\n"+
			"\t\t\t%ss.DELETE(\"/:id\", s.%sHandler.Delete)\n"+
			"\t\t}", sn, sn, sn, sn, sn, sn, sn, sn, sn, sn, sn, sn,
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

	result := make([]string, 0, len(lines)+15)
	result = append(result, lines[:insertIdx]...)
	result = append(result, routeBlock)
	result = append(result, lines[insertIdx:]...)

	return os.WriteFile(fullPath, []byte(strings.Join(result, "\n")), 0o644)
}
