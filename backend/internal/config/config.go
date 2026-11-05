// Package config 从环境变量加载并校验进程运行配置。
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config 包含随部署环境变化的运行参数。
type Config struct {
	Addr, DSN, FrontendDir, UploadDir string
	MaxUploadBytes                    int64
	SessionTTL                        time.Duration
}

// Load 返回已经校验的配置；未配置 MySQL 时立即返回错误。
func Load() (Config, error) {
	c := Config{Addr: env("APP_ADDR", ":8080"), DSN: os.Getenv("MYSQL_DSN"), FrontendDir: env("FRONTEND_DIR", "../frontend"), UploadDir: env("UPLOAD_DIR", "data/uploads"), MaxUploadBytes: 20 << 20, SessionTTL: 12 * time.Hour}
	if c.DSN == "" {
		return Config{}, fmt.Errorf("MYSQL_DSN is required, for example: supervision:password@tcp(127.0.0.1:3306)/supervision?parseTime=true&charset=utf8mb4")
	}
	if v := os.Getenv("MAX_UPLOAD_MB"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 1 || n > 100 {
			return Config{}, fmt.Errorf("MAX_UPLOAD_MB must be between 1 and 100")
		}
		c.MaxUploadBytes = n << 20
	}
	return c, nil
}
func env(k, v string) string {
	if x := os.Getenv(k); x != "" {
		return x
	}
	return v
}
