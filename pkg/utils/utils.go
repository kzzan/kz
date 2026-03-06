package utils

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

func ToPascalCase(input string) string {
	return toCamelCase(input, true)
}

func ToSnakeCase(input string) string {
	result := ""
	for i, r := range input {
		if i > 0 && unicode.IsUpper(r) {
			result += "_"
		}
		result += strings.ToLower(string(r))
	}
	return result
}

func ToCamelCase(input string) string {
	return toCamelCase(input, false)
}

// toCamelCase 内部驼峰转换函数
func toCamelCase(input string, isPascal bool) string {
	// 替换分隔符为空格用于处理
	normalized := strings.NewReplacer(
		"-", " ",
		"_", " ",
		"/", " ",
		".", " ",
	).Replace(input)

	// 分割字符串
	words := strings.Fields(normalized)

	if len(words) == 0 {
		return ""
	}

	result := ""
	for i, word := range words {
		if i == 0 && !isPascal {
			// camelCase 的首字母小写
			result += strings.ToLower(word)
		} else {
			// 其他情况首字母大写
			if len(word) > 0 {
				result += strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
			}
		}
	}

	return result
}

func IsValidProjectName(name string) bool {
	if name == "" {
		return false
	}

	// 检查首字符是否为字母
	if !unicode.IsLetter(rune(name[0])) {
		return false
	}

	// 检查其余字符是否为字母、数字或下划线
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}

	return true
}

func IsValidComponentName(name string) bool {
	return IsValidProjectName(name)
}

func IsProjectRoot() bool {
	// 检查 go.mod 文件
	if _, err := os.Stat("go.mod"); err != nil {
		return false
	}

	// 检查 internal 目录
	if _, err := os.Stat("internal"); err != nil {
		return false
	}

	return true
}

// CreateDirectoryStructure 创建目录结构
func CreateDirectoryStructure(basePath string, dirs []string) error {
	for _, dir := range dirs {
		dirPath := filepath.Join(basePath, dir)
		if err := os.MkdirAll(dirPath, 0o755); err != nil {
			return err
		}
	}
	return nil
}

// FileExists 检查文件是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// DirectoryExists 检查目录是否存在
func DirectoryExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func GetProjectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if FileExists(filepath.Join(cwd, "go.mod")) {
			return cwd, nil
		}

		parent := filepath.Dir(cwd)
		if parent == cwd {
			// 已经到达根目录
			break
		}
		cwd = parent
	}

	return "", os.ErrNotExist
}

// ReadFile 读取文件内容
func ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteFile 写入文件
func WriteFile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

// AppendFile 追加文件内容
func AppendFile(path string, content string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}()

	_, err = f.WriteString(content)
	return err
}

func FileNameToPackageName(filename string) string {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	name = strings.TrimSuffix(name, "_handler")
	name = strings.TrimSuffix(name, "_service")
	name = strings.TrimSuffix(name, "_repository")
	name = strings.TrimSuffix(name, "_model")
	return name
}

func GenerateImportPath(projectName string, relativePath string) string {
	components := []string{projectName}
	if relativePath != "" {
		relativePath = strings.TrimPrefix(relativePath, "./")
		relativePath = strings.TrimPrefix(relativePath, "/")
		components = append(components, strings.Split(relativePath, "/")...)
	}
	return strings.Join(components, "/")
}

func ValidateGoPackageName(name string) bool {
	if name == "" {
		return false
	}

	if !unicode.IsLetter(rune(name[0])) && name[0] != '_' {
		return false
	}

	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}

	return true
}

func SanitizeFileName(name string) string {
	re := regexp.MustCompile(`[^\w]`)
	return re.ReplaceAllString(name, "_")
}

func JoinPath(parts ...string) string {
	return filepath.Join(parts...)
}

func GetRelativePath(basePath string, targetPath string) (string, error) {
	return filepath.Rel(basePath, targetPath)
}

func ExtractComponentName(filename string) string {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	for _, suffix := range []string{"_handler", "_service", "_repository", "_model"} {
		if strings.HasSuffix(name, suffix) {
			return strings.TrimSuffix(name, suffix)
		}
	}
	return name
}

func NormalizeComponentName(name string) string {
	snakeCase := ToSnakeCase(name)

	for _, suffix := range []string{"_handler", "_service", "_repository", "_model"} {
		snakeCase = strings.TrimSuffix(snakeCase, suffix)
	}

	return snakeCase
}

func IsReservedKeyword(name string) bool {
	keywords := []string{
		"break", "case", "chan", "const", "continue", "default", "defer", "else",
		"fallthrough", "for", "func", "go", "goto", "if", "import", "interface",
		"map", "package", "range", "return", "select", "struct", "switch", "type",
		"var",
	}

	for _, kw := range keywords {
		if name == kw {
			return true
		}
	}
	return false
}

func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func ListFiles(dirPath string) ([]string, error) {
	var files []string
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

func ListDirectories(dirPath string) ([]string, error) {
	var dirs []string
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		}
	}

	return dirs, nil
}

func ReadModuleName(dir string) string {
	gomodPath := filepath.Join(dir, "go.mod")

	data, err := os.ReadFile(gomodPath)
	if err != nil {
		return filepath.Base(dir)
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			moduleName := strings.TrimPrefix(line, "module ")
			moduleName = strings.TrimSpace(moduleName)
			if moduleName != "" {
				return moduleName
			}
		}
	}

	return filepath.Base(dir)
}
