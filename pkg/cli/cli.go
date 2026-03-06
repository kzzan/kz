package cli

import (
	"fmt"
	"os"
	"path/filepath"

    "github.com/kzzan/kz/pkg/generator"
	"github.com/kzzan/kz/pkg/utils"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newCmd, versionCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "kz",
	Short: "kz v1.0 - 现代化的 Go 项目脚手架生成工具",
	Long: `kz v1.0 是一个强大的 Go 项目脚手架工具，用于快速初始化项目和生成代码组件。

支持的命令：
  kz new [project-name]          创建新项目
  kz new all [name]              生成完整四层组件 + 自动注册
  kz new handler [name]          生成 Handler + 注册到 package.go
  kz new service [name]          生成 Service + 注册到 package.go
  kz new repo [name]             生成 Repository + 注册到 package.go
  kz new model [name]            生成 Model
  kz new cron [name]             生成定时任务 + 注册到 cron/package.go
  kz new consumer [name]         生成队列消费者 + 注册到 consumer/package.go
  kz new middleware [name]       生成空中间件
  kz version                     显示版本信息`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "创建新项目或生成代码组件",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("用法: kz new <command> [args]")
			fmt.Println("")
			fmt.Println("Commands:")
			fmt.Println("  [project-name]    创建新项目")
			fmt.Println("  all [name]        生成完整四层组件（handler/service/repo/model）+ 自动注册")
			fmt.Println("  handler [name]    生成 Handler + 注册到 internal/handler/package.go")
			fmt.Println("  service [name]    生成 Service + 注册到 internal/service/package.go")
			fmt.Println("  repo    [name]    生成 Repository + 注册到 internal/repository/package.go")
			fmt.Println("  model   [name]    生成 Model")
			fmt.Println("  cron    [name]    生成定时任务 + 注册到 internal/cron/package.go")
			fmt.Println("  consumer [name]   生成队列消费者 + 注册到 internal/consumer/package.go")
			fmt.Println("  middleware [name] 生成空中间件")
			return
		}

		switch args[0] {
		case "handler":
			handleNewHandler(args[1:])
		case "service":
			handleNewService(args[1:])
		case "repo":
			handleNewRepo(args[1:])
		case "model":
			handleNewModel(args[1:])
		case "all":
			handleNewAll(args[1:])
		case "cron":
			handleNewCron(args[1:])
		case "consumer":
			handleNewConsumer(args[1:])
		case "middleware":
			handleNewMiddleware(args[1:])
		default:
			handleNewProject(args)
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("kz v1.0")
		fmt.Println("A modern Go project scaffold generator")
	},
}

func handleNewProject(args []string) {
	projectName := ""
	if len(args) > 0 {
		projectName = args[0]
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			exitWithError("获取当前目录失败", err)
		}
		projectName = filepath.Base(cwd)
	}

	if projectName == "" {
		exitWithMsg("项目名称不能为空")
	}
	if !utils.IsValidProjectName(projectName) {
		exitWithMsg(fmt.Sprintf("项目名称 '%s' 无效，请使用字母、数字和下划线", projectName))
	}

	fmt.Printf("正在创建项目: %s\n", projectName)

	if err := generator.NewProjectGenerator(projectName).GenerateProject(); err != nil {
		exitWithError("创建项目失败", err)
	}

	fmt.Printf("\n✓ 项目 '%s' 创建成功！\n\n", projectName)
	fmt.Println("后续步骤:")
	fmt.Printf("  cd %s\n", projectName)
	fmt.Println("  go mod tidy")
	fmt.Println("  make build")
	fmt.Println("\n生成新组件:")
	fmt.Println("  kz new all order")
}

func handleNewHandler(args []string) {
	componentName := requireComponentName(args, "handler")
	requireProjectRoot()

	gen := generator.NewComponentGenerator(componentName)
	snakeName := utils.ToSnakeCase(componentName)

	fmt.Printf("正在生成 Handler: %s\n", componentName)

	runSteps([]step{
		{
			label: "[1/3] 生成文件",
			fn:    gen.GenerateHandler,
			file:  fmt.Sprintf("internal/handler/%s.go", snakeName),
		},
		{
			label: "[2/3] 注册到 package.go",
			fn:    gen.AppendHandlerToPackage,
			file:  "internal/handler/package.go",
		},
		{
			label: "[3/3] 追加路由",
			fn:    gen.AppendHandlerToRoutes,
			file:  "internal/server/routes.go + server.go",
		},
	})

	fmt.Printf("\n✓ Handler '%s' 生成成功！\n", componentName)
}

func handleNewService(args []string) {
	componentName := requireComponentName(args, "service")
	requireProjectRoot()

	gen := generator.NewComponentGenerator(componentName)
	snakeName := utils.ToSnakeCase(componentName)

	fmt.Printf("正在生成 Service: %s\n", componentName)

	runSteps([]step{
		{
			label: "[1/2] 生成文件",
			fn:    gen.GenerateService,
			file:  fmt.Sprintf("internal/service/%s.go", snakeName),
		},
		{
			label: "[2/2] 注册到 package.go",
			fn:    gen.AppendServiceToPackage,
			file:  "internal/service/package.go",
		},
	})

	fmt.Printf("\n✓ Service '%s' 生成成功！\n", componentName)
}

func handleNewRepo(args []string) {
	componentName := requireComponentName(args, "repo")
	requireProjectRoot()

	gen := generator.NewComponentGenerator(componentName)
	snakeName := utils.ToSnakeCase(componentName)

	fmt.Printf("正在生成 Repository: %s\n", componentName)

	runSteps([]step{
		{
			label: "[1/2] 生成文件",
			fn:    gen.GenerateRepository,
			file:  fmt.Sprintf("internal/repository/%s.go", snakeName),
		},
		{
			label: "[2/2] 注册到 package.go",
			fn:    gen.AppendRepositoryToPackage,
			file:  "internal/repository/package.go",
		},
	})

	fmt.Printf("\n✓ Repository '%s' 生成成功！\n", componentName)
}

func handleNewModel(args []string) {
	componentName := requireComponentName(args, "model")
	requireProjectRoot()

	gen := generator.NewComponentGenerator(componentName)
	snakeName := utils.ToSnakeCase(componentName)

	fmt.Printf("正在生成 Model: %s\n", componentName)

	runSteps([]step{
		{
			label: "[1/1] 生成文件",
			fn:    gen.GenerateModel,
			file:  fmt.Sprintf("internal/models/%s.go", snakeName),
		},
	})

	fmt.Printf("\n✓ Model '%s' 生成成功！\n", componentName)
}

func handleNewAll(args []string) {
	componentName := requireComponentName(args, "all")
	requireProjectRoot()

	gen := generator.NewComponentGenerator(componentName)
	snakeName := utils.ToSnakeCase(componentName)

	fmt.Printf("正在为 '%s' 生成完整四层组件...\n\n", componentName)

	runSteps([]step{
		{
			label: "[1/8] 生成 Model",
			fn:    gen.GenerateModel,
			file:  fmt.Sprintf("internal/models/%s.go", snakeName),
		},
		{
			label: "[2/8] 生成 Repository",
			fn:    gen.GenerateRepository,
			file:  fmt.Sprintf("internal/repository/%s.go", snakeName),
		},
		{
			label: "[3/8] 生成 Service",
			fn:    gen.GenerateService,
			file:  fmt.Sprintf("internal/service/%s.go", snakeName),
		},
		{
			label: "[4/8] 生成 Handler",
			fn:    gen.GenerateHandler,
			file:  fmt.Sprintf("internal/handler/%s.go", snakeName),
		},
		{
			label: "[5/8] 注册 Repository",
			fn:    gen.AppendRepositoryToPackage,
			file:  "internal/repository/package.go",
		},
		{
			label: "[6/8] 注册 Service",
			fn:    gen.AppendServiceToPackage,
			file:  "internal/service/package.go",
		},
		{
			label: "[7/8] 注册 Handler",
			fn:    gen.AppendHandlerToPackage,
			file:  "internal/handler/package.go",
		},
		{
			label: "[8/8] 追加路由",
			fn:    gen.AppendHandlerToRoutes,
			file:  "internal/server/routes.go + server.go",
		},
	})

	fmt.Printf("\n✓ '%s' 完整四层组件生成成功！\n\n", componentName)
	fmt.Println("后续步骤:")
	fmt.Printf("  1. 编辑 internal/models/%s.go          定义数据模型\n", snakeName)
	fmt.Printf("  2. 编辑 internal/repository/%s.go      实现数据访问\n", snakeName)
	fmt.Printf("  3. 编辑 internal/service/%s.go         实现业务逻辑\n", snakeName)
	fmt.Printf("  4. 编辑 internal/handler/%s.go         实现 HTTP 处理\n", snakeName)
}

func handleNewCron(args []string) {
	componentName := requireComponentName(args, "cron")
	requireProjectRoot()

	gen := generator.NewComponentGenerator(componentName)
	snakeName := utils.ToSnakeCase(componentName)

	fmt.Printf("正在生成 Cron Job: %s\n", componentName)

	runSteps([]step{
		{
			label: "[1/4] 生成 Job 文件",
			fn:    gen.GenerateCron,
			file:  fmt.Sprintf("internal/cron/%s.go", snakeName),
		},
		{
			label: "[2/4] 确保 cron/package.go",
			fn:    gen.EnsureCronPackage,
			file:  "internal/cron/package.go",
		},
		{
			label: "[3/4] 注册 Job 到 Scheduler",
			fn:    gen.AppendCronToScheduler,
			file:  "internal/cron/package.go",
		},
		{
			label: "[4/4] 注册 cron.Package 到 server",
			fn:    gen.AppendCronPackageToServer,
			file:  "internal/server/package.go",
		},
	})

	fmt.Printf("\n✓ Cron Job '%s' 生成成功！\n\n", componentName)
}

func handleNewConsumer(args []string) {
	componentName := requireComponentName(args, "consumer")
	requireProjectRoot()

	gen := generator.NewComponentGenerator(componentName)
	snakeName := utils.ToSnakeCase(componentName)

	fmt.Printf("正在生成 Consumer: %s\n", componentName)

	runSteps([]step{
		{
			label: "[1/4] 生成 Consumer 文件",
			fn:    gen.GenerateConsumer,
			file:  fmt.Sprintf("internal/consumer/%s.go", snakeName),
		},
		{
			label: "[2/4] 确保 consumer/package.go",
			fn:    gen.EnsureConsumerPackage,
			file:  "internal/consumer/package.go",
		},
		{
			label: "[3/4] 注册 Consumer 到 Manager",
			fn:    gen.AppendConsumerToManager,
			file:  "internal/consumer/package.go",
		},
		{
			label: "[4/4] 注册 consumer.Package 到 server",
			fn:    gen.AppendConsumerPackageToServer,
			file:  "internal/server/package.go",
		},
	})

	fmt.Printf("\n✓ Consumer '%s' 生成成功！\n\n", componentName)
}

func handleNewMiddleware(args []string) {
	componentName := requireComponentName(args, "middleware")
	requireProjectRoot()

	gen := generator.NewComponentGenerator(componentName)
	snakeName := utils.ToSnakeCase(componentName)
	pascalName := utils.ToPascalCase(componentName)

	fmt.Printf("正在生成 Middleware: %s\n", componentName)

	runSteps([]step{
		{
			label: "[1/1] 生成文件",
			fn:    gen.GenerateMiddleware,
			file:  fmt.Sprintf("internal/middleware/%s.go", snakeName),
		},
	})

	fmt.Printf("\n✓ Middleware '%s' 生成成功！\n\n", componentName)
	fmt.Println("后续步骤:")
	fmt.Printf("  在 internal/server/server.go 中使用:\n")
	fmt.Printf("  engine.Use(middleware.%s(logger))\n", pascalName)
}

type step struct {
	label string
	fn    func() error
	file  string
}

func runSteps(steps []step) {
	for _, s := range steps {
		fmt.Printf("  %s\n", s.label)
		if err := s.fn(); err != nil {
			fmt.Printf("  ✗ 失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  ✓ %s\n", s.file)
	}
}

func requireComponentName(args []string, cmd string) string {
	if len(args) == 0 {
		fmt.Printf("错误: 请指定组件名称\n")
		fmt.Printf("用法: kz new %s <name>\n", cmd)
		fmt.Printf("示例: kz new %s user\n", cmd)
		os.Exit(1)
	}
	name := args[0]
	if !utils.IsValidComponentName(name) {
		exitWithMsg(fmt.Sprintf("组件名称 '%s' 无效，请使用字母、数字和下划线", name))
	}
	return name
}

func requireProjectRoot() {
	if !utils.IsProjectRoot() {
		fmt.Println("错误: 当前目录不是有效的项目根目录")
		fmt.Println("请在项目根目录（含 go.mod）下运行此命令")
		os.Exit(1)
	}
}

func exitWithError(msg string, err error) {
	fmt.Printf("错误: %s - %v\n", msg, err)
	os.Exit(1)
}

func exitWithMsg(msg string) {
	fmt.Printf("错误: %s\n", msg)
	os.Exit(1)
}
