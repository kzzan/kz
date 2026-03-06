package config

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	ProjectName string
	ProjectPath string
	Database    string
}

func (c *Config) Validate() error {
	if c.ProjectName == "" {
		return fmt.Errorf("项目名称不能为空")
	}
	if c.Database == "" || (c.Database != "postgresql" && c.Database != "mysql") {
		return fmt.Errorf("无效的数据库类型")
	}
	return nil
}

func (c *Config) GetFullPath() string {
	return filepath.Join(c.ProjectPath, c.ProjectName)
}

func (c *Config) Save(path string) error {
	viper.SetConfigName("aset")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(path)

	viper.Set("project_name", c.ProjectName)
	viper.Set("database", c.Database)

	return viper.WriteConfigAs(filepath.Join(path, "aset.yaml"))
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("aset")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("无法读取配置文件: %v", err)
	}

	return &Config{
		ProjectName: viper.GetString("project_name"),
		Database:    viper.GetString("database"),
	}, nil
}
