# Go SOCKS5 代理服务器

一个基于 Go 语言开发的高性能 SOCKS5 代理服务器，具有高并发处理能力、自动重试机制、IPv6 支持等特性。

## 主要特性

### 协议支持
- 完整支持 SOCKS5 协议
- 支持 IPv4/IPv6 双栈
- 支持域名解析

### 性能特性
- 高并发连接处理（默认最大 10000 并发）
- 基于 goroutine 的异步 I/O
- 可配置的缓冲区大小
- 实时连接统计

### 可靠性
- 自动重试连接机制
- 智能超时控制
- 优雅关闭支持
- 完整的错误处理

### 可观测性
- 彩色日志输出
- 详细的连接信息
- 实时传输速率显示
- 定期统计报告

### 可配置性
- 丰富的命令行参数
- 可调节的性能参数
- 灵活的日志级别

## 系统要求

- Go 1.21 或更高版本
- 支持 Linux、macOS 和 Windows
- 建议：至少 2GB 内存用于高并发场景

## 安装说明

1. 获取源码：
   git clone https://github.com/yourusername/socks5proxy.git
   cd socks5proxy/server

2. 安装依赖：
   go mod tidy

3. 编译程序：
   go build -o socks5proxy

## 使用说明

### 基本用法
直接运行程序：./socks5proxy

默认配置：
- 监听地址：127.0.0.1
- 监听端口：1080
- 超时时间：30秒
- 缓冲区大小：32KB

### 命令行参数
-host: 监听地址（默认：127.0.0.1）
-port: 监听端口（默认：1080）
-timeout: 连接超时时间（默认：30s）
-buffer: 缓冲区大小（默认：32KB）
-retries: 连接重试次数（默认：3）
-retry-delay: 重试间隔时间（默认：1s）
-dial-timeout: 连接建立超时（默认：10s）
-ipv6: 是否启用 IPv6（默认：true）
-log-level: 日志级别（默认：info）

### 使用示例
1. 监听所有网络接口：
   ./socks5proxy -host 0.0.0.0

2. 自定义端口：
   ./socks5proxy -port 1081

3. 调整性能参数：
   ./socks5proxy -buffer 65536 -timeout 60s

4. 禁用 IPv6：
   ./socks5proxy -ipv6=false

## 监控和统计

服务器每 30 秒输出一次统计信息：
- 运行时间
- 当前活跃连接数
- 总连接数
- 总传输字节数

## 日志说明

日志颜色说明：
- 绿色：时间戳
- 青色：客户端地址
- 蓝色：目标地址
- 黄色：传输数据量
- 紫色：时间信息
- 红色：错误信息

## 性能优化建议

1. 调整缓冲区大小：
   - 大文件传输：32KB - 64KB
   - 普通使用：8KB - 16KB

2. 并发连接数：
   - 默认支持 10000 个并发
   - 可根据服务器资源调整

3. 超时设置：
   - 根据网络情况调整超时参数
   - 不稳定网络建议增加重试次数

## 故障排除

1. 连接失败：
   - 检查监听地址和端口
   - 确认防火墙设置
   - 查看错误日志

2. 性能问题：
   - 调整缓冲区大小
   - 检查并发连接数
   - 监控系统资源

## 系统配置建议

### Linux 系统优化
1. 文件描述符限制：
   编辑 /etc/security/limits.conf
   * soft nofile 65535
   * hard nofile 65535

2. 内核参数优化：
   编辑 /etc/sysctl.conf
   net.ipv4.tcp_fin_timeout = 30
   net.ipv4.tcp_keepalive_time = 1200
   net.ipv4.tcp_max_syn_backlog = 8192
   net.ipv4.tcp_max_tw_buckets = 5000
   net.ipv4.tcp_tw_reuse = 1
   net.ipv4.tcp_fastopen = 3

### 安全建议
1. 访问控制：
   - 使用防火墙限制访问源IP
   - 避免监听公网地址
   - 定期检查连接日志

2. 资源控制：
   - 设置合理的连接数限制
   - 监控系统资源使用情况
   - 配置适当的超时时间

3. 网络安全：
   - 建议在可信网络环境使用
   - 考虑配置 TLS 加密
   - 定期更新系统和依赖

## 代码结构
server/
├── main.go        // 程序入口，参数解析
├── config.go      // 配置相关定义
├── server.go      // 服务器核心实现
├── handler.go     // 请求处理逻辑
├── go.mod         // 依赖管理
└── README.md      // 项目文档

## 开发计划

### 即将支持的功能
- 用户认证机制
- 流量统计和限制
- Web管理界面
- 负载均衡
- 集群支持

## 许可证

MIT License

## 作者

[Your Name]

## 版本历史

### v1.0.0
- 初始版本发布
- 基本 SOCKS5 代理功能
- 高并发支持
- 详细日志记录