package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"
)

// 自定义日志格式
func init() {
	log.SetFlags(0) // 清除默认的时间格式
	log.SetOutput(new(logWriter))
}

type logWriter struct{}

func (writer *logWriter) Write(bytes []byte) (int, error) {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	return fmt.Printf("%s%s%s %s", colorGreen, timestamp, colorReset, string(bytes))
}

func main() {
	// 定义命令行参数
	var (
		listenHost = flag.String("host", "127.0.0.1", "Host to listen on")
		listenPort = flag.Int("port", 1080, "Port to listen on")
		timeout    = flag.Duration("timeout", 30*time.Second, "Connection timeout")
		bufferSize = flag.Int("buffer", 32*1024, "Buffer size in bytes")
		maxRetries = flag.Int("retries", 3, "Maximum number of connection retries")
		retryDelay = flag.Duration("retry-delay", time.Second, "Delay between retries")
		dialTimeout = flag.Duration("dial-timeout", 10*time.Second, "Dial timeout")
		enableIPv6 = flag.Bool("ipv6", true, "Enable IPv6 support")
		logLevel   = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	)

	// 解析命令行参数
	flag.Parse()

	// 验证参数
	if *listenPort < 1 || *listenPort > 65535 {
		log.Fatalf("Invalid port number: %d", *listenPort)
	}

	// 构建监听地址
	listenAddr := fmt.Sprintf("%s:%d", *listenHost, *listenPort)

	// 创建配置
	config := &Config{
		ListenAddr:  listenAddr,
		Timeout:     *timeout,
		BufferSize:  *bufferSize,
		MaxRetries:  *maxRetries,
		RetryDelay:  *retryDelay,
		DialTimeout: *dialTimeout,
		EnableIPv6:  *enableIPv6,
		LogLevel:    strings.ToLower(*logLevel),
	}

	// 打印配置信息
	log.Printf("Starting SOCKS5 proxy server with configuration:")
	log.Printf("  Listen address: %s%s%s", colorBlue, listenAddr, colorReset)
	log.Printf("  Timeout: %s%v%s", colorYellow, timeout, colorReset)
	log.Printf("  Buffer size: %s%d%s bytes", colorYellow, *bufferSize, colorReset)
	log.Printf("  Max retries: %s%d%s", colorYellow, *maxRetries, colorReset)
	log.Printf("  Retry delay: %s%v%s", colorYellow, *retryDelay, colorReset)
	log.Printf("  Dial timeout: %s%v%s", colorYellow, *dialTimeout, colorReset)
	log.Printf("  IPv6 enabled: %s%v%s", colorYellow, *enableIPv6, colorReset)
	log.Printf("  Log level: %s%s%s", colorYellow, *logLevel, colorReset)

	// 创建并运行服务器
	server := NewServer(config)
	if err := server.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
