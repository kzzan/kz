package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kzzan/kz/pkg/generator"
	"github.com/kzzan/kz/pkg/utils"

	"github.com/spf13/cobra"
)

var version = "dev"

type app struct {
	quiet bool
}

type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string {
	return e.err.Error()
}

func (e *exitError) Unwrap() error {
	return e.err
}

func Execute() int {
	cmd := NewRootCmd()
	if err := cmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), err)
		return exitCode(err)
	}
	return 0
}

func NewRootCmd() *cobra.Command {
	a := &app{}

	rootCmd := &cobra.Command{
		Use:           "kz",
		Short:         "Generate Go project scaffolds and components",
		Long:          "kz initializes Go service scaffolds and generates focused application components.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}

	rootCmd.PersistentFlags().BoolVarP(&a.quiet, "quiet", "q", false, "suppress progress output")

	rootCmd.AddCommand(
		a.newInitCmd(),
		a.newGenerateCmd(),
		a.newLegacyNewCmd(),
		newVersionCmd(),
	)

	return rootCmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print kz version",
		Args:  noArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), version)
			return err
		},
	}
}

func (a *app) newInitCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init [directory]",
		Short: "Initialize a scaffold in the current or target directory",
		Long: strings.Join([]string{
			"Initialize a Go service scaffold.",
			"If no directory is given, kz writes into the current working directory.",
		}, "\n"),
		Args: maxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runInit(cmd, args, force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "allow initializing a non-empty directory")

	return cmd
}

func (a *app) newGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate",
		Aliases: []string{"gen", "g"},
		Short:   "Generate a component inside an existing kz project",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(
		a.newComponentCmd("all", "Generate model, repository, service, and handler", a.runGenerateAll),
		a.newComponentCmd("handler", "Generate a handler and register routes", a.runGenerateHandler),
		a.newComponentCmd("service", "Generate a service and register it", a.runGenerateService),
		a.newComponentCmd("repo", "Generate a repository and register it", a.runGenerateRepo),
		a.newComponentCmd("model", "Generate a model", a.runGenerateModel),
		a.newComponentCmd("cron", "Generate a cron job and register it", a.runGenerateCron),
		a.newComponentCmd("consumer", "Generate a consumer and register it", a.runGenerateConsumer),
		a.newComponentCmd("middleware", "Generate a middleware template", a.runGenerateMiddleware),
	)

	return cmd
}

func (a *app) newLegacyNewCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "new [name]",
		Short:  "Compatibility alias for init and generate",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runLegacyNew(cmd, args)
		},
	}
}

func (a *app) newComponentCmd(name, short string, run func(cmd *cobra.Command, componentName string) error) *cobra.Command {
	return &cobra.Command{
		Use:   name + " <name>",
		Short: short,
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, args[0])
		},
	}
}

func (a *app) runInit(cmd *cobra.Command, args []string, force bool) error {
	target := "."
	if len(args) == 1 {
		target = args[0]
	}

	projectPath, projectName, err := resolveProjectTarget(target)
	if err != nil {
		return usageError(err.Error())
	}
	if !utils.IsValidProjectName(projectName) {
		return usageError(fmt.Sprintf("invalid project name %q: use letters, digits, and underscores", projectName))
	}
	if err := ensureInitTarget(projectPath, force); err != nil {
		return err
	}

	a.logf(cmd.ErrOrStderr(), "initializing project in %s", projectPath)

	if err := generator.NewProjectGenerator(projectName, projectPath).GenerateProject(); err != nil {
		return runtimeError(fmt.Errorf("init project: %w", err))
	}

	a.logf(cmd.ErrOrStderr(), "created scaffold for %s", projectName)
	a.logf(cmd.ErrOrStderr(), "next: go mod tidy")

	return nil
}

func (a *app) runGenerateHandler(cmd *cobra.Command, componentName string) error {
	gen, err := componentGenerator(componentName)
	if err != nil {
		return err
	}

	a.logf(cmd.ErrOrStderr(), "generating handler %s", componentName)

	return a.runSteps(cmd.ErrOrStderr(), []step{
		{
			label: "write handler",
			fn:    gen.GenerateHandler,
			file:  fmt.Sprintf("internal/handler/%s.go", gen.SnakeName),
		},
		{
			label: "register handler package",
			fn:    gen.AppendHandlerToPackage,
			file:  "internal/handler/package.go",
		},
		{
			label: "register routes",
			fn:    gen.AppendHandlerToRoutes,
			file:  "internal/server/routes.go, internal/server/server.go",
		},
	})
}

func (a *app) runGenerateService(cmd *cobra.Command, componentName string) error {
	gen, err := componentGenerator(componentName)
	if err != nil {
		return err
	}

	a.logf(cmd.ErrOrStderr(), "generating service %s", componentName)

	return a.runSteps(cmd.ErrOrStderr(), []step{
		{
			label: "write service",
			fn:    gen.GenerateService,
			file:  fmt.Sprintf("internal/service/%s.go", gen.SnakeName),
		},
		{
			label: "register service package",
			fn:    gen.AppendServiceToPackage,
			file:  "internal/service/package.go",
		},
	})
}

func (a *app) runGenerateRepo(cmd *cobra.Command, componentName string) error {
	gen, err := componentGenerator(componentName)
	if err != nil {
		return err
	}

	a.logf(cmd.ErrOrStderr(), "generating repository %s", componentName)

	return a.runSteps(cmd.ErrOrStderr(), []step{
		{
			label: "write repository",
			fn:    gen.GenerateRepository,
			file:  fmt.Sprintf("internal/repository/%s.go", gen.SnakeName),
		},
		{
			label: "register repository package",
			fn:    gen.AppendRepositoryToPackage,
			file:  "internal/repository/package.go",
		},
	})
}

func (a *app) runGenerateModel(cmd *cobra.Command, componentName string) error {
	gen, err := componentGenerator(componentName)
	if err != nil {
		return err
	}

	a.logf(cmd.ErrOrStderr(), "generating model %s", componentName)

	return a.runSteps(cmd.ErrOrStderr(), []step{
		{
			label: "write model",
			fn:    gen.GenerateModel,
			file:  fmt.Sprintf("internal/models/%s.go", gen.SnakeName),
		},
	})
}

func (a *app) runGenerateAll(cmd *cobra.Command, componentName string) error {
	gen, err := componentGenerator(componentName)
	if err != nil {
		return err
	}

	a.logf(cmd.ErrOrStderr(), "generating component set %s", componentName)

	err = a.runSteps(cmd.ErrOrStderr(), []step{
		{
			label: "write model",
			fn:    gen.GenerateModel,
			file:  fmt.Sprintf("internal/models/%s.go", gen.SnakeName),
		},
		{
			label: "write repository",
			fn:    gen.GenerateRepository,
			file:  fmt.Sprintf("internal/repository/%s.go", gen.SnakeName),
		},
		{
			label: "write service",
			fn:    gen.GenerateService,
			file:  fmt.Sprintf("internal/service/%s.go", gen.SnakeName),
		},
		{
			label: "write handler",
			fn:    gen.GenerateHandler,
			file:  fmt.Sprintf("internal/handler/%s.go", gen.SnakeName),
		},
		{
			label: "register repository package",
			fn:    gen.AppendRepositoryToPackage,
			file:  "internal/repository/package.go",
		},
		{
			label: "register service package",
			fn:    gen.AppendServiceToPackage,
			file:  "internal/service/package.go",
		},
		{
			label: "register handler package",
			fn:    gen.AppendHandlerToPackage,
			file:  "internal/handler/package.go",
		},
		{
			label: "register routes",
			fn:    gen.AppendHandlerToRoutes,
			file:  "internal/server/routes.go, internal/server/server.go",
		},
	})
	if err != nil {
		return err
	}

	a.logf(cmd.ErrOrStderr(), "edit the generated files to add domain logic")

	return nil
}

func (a *app) runGenerateCron(cmd *cobra.Command, componentName string) error {
	gen, err := componentGenerator(componentName)
	if err != nil {
		return err
	}

	a.logf(cmd.ErrOrStderr(), "generating cron job %s", componentName)

	return a.runSteps(cmd.ErrOrStderr(), []step{
		{
			label: "write cron job",
			fn:    gen.GenerateCron,
			file:  fmt.Sprintf("internal/cron/%s.go", gen.SnakeName),
		},
		{
			label: "ensure cron package",
			fn:    gen.EnsureCronPackage,
			file:  "internal/cron/package.go",
		},
		{
			label: "register scheduler",
			fn:    gen.AppendCronToScheduler,
			file:  "internal/cron/package.go",
		},
		{
			label: "register server package",
			fn:    gen.AppendCronPackageToServer,
			file:  "internal/server/package.go",
		},
	})
}

func (a *app) runGenerateConsumer(cmd *cobra.Command, componentName string) error {
	gen, err := componentGenerator(componentName)
	if err != nil {
		return err
	}

	a.logf(cmd.ErrOrStderr(), "generating consumer %s", componentName)

	return a.runSteps(cmd.ErrOrStderr(), []step{
		{
			label: "write consumer",
			fn:    gen.GenerateConsumer,
			file:  fmt.Sprintf("internal/consumer/%s.go", gen.SnakeName),
		},
		{
			label: "ensure consumer package",
			fn:    gen.EnsureConsumerPackage,
			file:  "internal/consumer/package.go",
		},
		{
			label: "register manager",
			fn:    gen.AppendConsumerToManager,
			file:  "internal/consumer/package.go",
		},
		{
			label: "register server package",
			fn:    gen.AppendConsumerPackageToServer,
			file:  "internal/server/package.go",
		},
	})
}

func (a *app) runGenerateMiddleware(cmd *cobra.Command, componentName string) error {
	gen, err := componentGenerator(componentName)
	if err != nil {
		return err
	}

	a.logf(cmd.ErrOrStderr(), "generating middleware %s", componentName)

	err = a.runSteps(cmd.ErrOrStderr(), []step{
		{
			label: "write middleware",
			fn:    gen.GenerateMiddleware,
			file:  fmt.Sprintf("internal/middleware/%s.go", gen.SnakeName),
		},
	})
	if err != nil {
		return err
	}

	a.logf(cmd.ErrOrStderr(), "register it in internal/server/server.go when needed")

	return nil
}

func (a *app) runLegacyNew(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return a.runInit(cmd, nil, false)
	}

	switch args[0] {
	case "all":
		return singleComponentCompat(cmd, args[1:], a.runGenerateAll)
	case "handler":
		return singleComponentCompat(cmd, args[1:], a.runGenerateHandler)
	case "service":
		return singleComponentCompat(cmd, args[1:], a.runGenerateService)
	case "repo":
		return singleComponentCompat(cmd, args[1:], a.runGenerateRepo)
	case "model":
		return singleComponentCompat(cmd, args[1:], a.runGenerateModel)
	case "cron":
		return singleComponentCompat(cmd, args[1:], a.runGenerateCron)
	case "consumer":
		return singleComponentCompat(cmd, args[1:], a.runGenerateConsumer)
	case "middleware":
		return singleComponentCompat(cmd, args[1:], a.runGenerateMiddleware)
	default:
		if len(args) != 1 {
			return usageError(fmt.Sprintf("expected exactly one project directory, got %d", len(args)))
		}
		return a.runInit(cmd, args[:1], false)
	}
}

type step struct {
	label string
	fn    func() error
	file  string
}

func (a *app) runSteps(w io.Writer, steps []step) error {
	for _, s := range steps {
		if err := s.fn(); err != nil {
			return runtimeError(fmt.Errorf("%s: %w", s.label, err))
		}
		a.logf(w, "created %s", s.file)
	}
	return nil
}

func (a *app) logf(w io.Writer, format string, args ...any) {
	if a.quiet {
		return
	}
	_, _ = fmt.Fprintf(w, format+"\n", args...)
}

func componentGenerator(componentName string) (*generator.ComponentGenerator, error) {
	if !utils.IsValidComponentName(componentName) {
		return nil, usageError(fmt.Sprintf("invalid component name %q: use letters, digits, and underscores", componentName))
	}

	projectRoot, err := utils.GetProjectRoot()
	if err != nil {
		return nil, usageError("run generate inside a kz project")
	}
	if !utils.DirectoryExists(filepath.Join(projectRoot, "internal")) {
		return nil, usageError(fmt.Sprintf("%s is missing an internal directory", projectRoot))
	}

	return generator.NewComponentGenerator(componentName, projectRoot), nil
}

func resolveProjectTarget(target string) (string, string, error) {
	cleaned := filepath.Clean(target)
	if cleaned == "" {
		return "", "", errors.New("project directory cannot be empty")
	}

	projectName := filepath.Base(cleaned)
	if cleaned == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return "", "", fmt.Errorf("read current directory: %w", err)
		}
		projectName = filepath.Base(cwd)
	}

	return cleaned, projectName, nil
}

func ensureInitTarget(projectPath string, force bool) error {
	info, err := os.Stat(projectPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return runtimeError(fmt.Errorf("inspect target directory: %w", err))
	}
	if !info.IsDir() {
		return usageError(fmt.Sprintf("%s exists and is not a directory", projectPath))
	}
	if force {
		return nil
	}

	entries, err := os.ReadDir(projectPath)
	if err != nil {
		return runtimeError(fmt.Errorf("read target directory: %w", err))
	}
	if len(entries) > 0 {
		return usageError(fmt.Sprintf("%s is not empty; rerun with --force to overwrite", projectPath))
	}

	return nil
}

func singleComponentCompat(cmd *cobra.Command, args []string, run func(cmd *cobra.Command, componentName string) error) error {
	if len(args) != 1 {
		return usageError(fmt.Sprintf("expected exactly one component name, got %d", len(args)))
	}
	return run(cmd, args[0])
}

func noArgs() cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			return usageError(fmt.Sprintf("%s accepts no arguments", cmd.CommandPath()))
		}
		return nil
	}
}

func exactArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != n {
			return usageError(fmt.Sprintf("%s requires exactly %d argument(s)", cmd.CommandPath(), n))
		}
		return nil
	}
}

func maxArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) > n {
			return usageError(fmt.Sprintf("%s accepts at most %d argument(s)", cmd.CommandPath(), n))
		}
		return nil
	}
}

func usageError(message string) error {
	return &exitError{
		code: 2,
		err:  errors.New(message),
	}
}

func runtimeError(err error) error {
	return &exitError{
		code: 1,
		err:  err,
	}
}

func exitCode(err error) int {
	var cliErr *exitError
	if errors.As(err, &cliErr) {
		return cliErr.code
	}

	msg := err.Error()
	if strings.Contains(msg, "unknown command") || strings.Contains(msg, "unknown flag") {
		return 2
	}

	return 1
}
