package main

import "time"

// 添加ANSI颜色代码常量
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
)

const (
	maxConcurrentConnections = 10000 // 最大并发连接数
)

// Config 服务器配置
type Config struct {
	ListenAddr  string
	Timeout     time.Duration
	BufferSize  int
	MaxRetries  int           // 最大重试次数
	RetryDelay  time.Duration // 重试间隔
	DialTimeout time.Duration // 连接超时
	EnableIPv6  bool         // 是否启用IPv6
	LogLevel    string       // 日志级别: debug, info, warn, error
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		ListenAddr:  "127.0.0.1:1080",
		Timeout:     30 * time.Second,
		BufferSize:  32 * 1024,
		MaxRetries:  3,
		RetryDelay:  time.Second,
		DialTimeout: 10 * time.Second,
		EnableIPv6:  true,
		LogLevel:    "info",
	}
} 