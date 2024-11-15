package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

func (s *Server) handleConnection(conn net.Conn) {
	clientAddr := conn.RemoteAddr().String()
	startTime := time.Now()
	log.Printf("[%s%s%s] New connection established", 
		colorCyan, clientAddr, colorReset)

	defer func() {
		conn.Close()
		duration := time.Since(startTime)
		log.Printf("[%s%s%s] Connection closed after %s%.2f seconds%s", 
			colorCyan, clientAddr, colorReset,
			colorPurple, duration.Seconds(), colorReset)
	}()

	// 处理SOCKS5握手
	if err := s.handleSocks5Handshake(conn); err != nil {
		log.Printf("[%s%s%s] Handshake failed: %v", colorCyan, clientAddr, colorReset, err)
		return
	}

	// 解析目标地址
	targetHost, targetPort, err := s.parseTargetAddress(conn)
	if err != nil {
		log.Printf("[%s%s%s] Failed to parse target address: %v", colorCyan, clientAddr, colorReset, err)
		return
	}

	log.Printf("[%s%s%s] Connecting to %s%s:%d%s", 
		colorCyan, clientAddr, colorReset,
		colorBlue, targetHost, targetPort, colorReset)

	// 建立目标连接
	var targetConn net.Conn
	targetConn, err = s.dialTargetWithRetry(targetHost, targetPort)
	if err != nil {
		log.Printf("[%s%s%s] Failed to connect to %s:%d: %v", 
			colorCyan, clientAddr, colorReset, targetHost, targetPort, err)
		return
	}
	defer targetConn.Close()

	// 发送连接成功响应给客户端
	_, err = conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	if err != nil {
		log.Printf("[%s%s%s] Failed to send success response: %v", 
			colorCyan, clientAddr, colorReset, err)
		return
	}

	// 开始数据传输
	s.startBidirectionalTransfer(conn, targetConn, clientAddr, targetHost, targetPort)
}

func (s *Server) handleSocks5Handshake(conn net.Conn) error {
	buf := make([]byte, 256)
	
	// 读取客户端支持的认证方法
	n, err := conn.Read(buf)
	if err != nil {
		return fmt.Errorf("read auth methods: %w", err)
	}

	if n < 2 || buf[0] != 0x05 {
		return fmt.Errorf("invalid SOCKS5 version: %d", buf[0])
	}

	methodCount := int(buf[1])
	methods := make([]byte, methodCount)
	copy(methods, buf[2:2+methodCount])

	log.Printf("Client supports %d auth methods: %v", methodCount, methods)

	// 目前只实现无认证方式
	_, err = conn.Write([]byte{0x05, 0x00})
	return err
}

func (s *Server) parseTargetAddress(conn net.Conn) (string, uint16, error) {
	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil {
		return "", 0, fmt.Errorf("read target address: %w", err)
	}

	if n < 4 {
		return "", 0, fmt.Errorf("invalid address message length")
	}

	var host string
	var port uint16
	addrType := buf[3]

	switch addrType {
	case 0x01: // IPv4
		if n < 10 {
			return "", 0, fmt.Errorf("invalid IPv4 address length")
		}
		ip := net.IP(buf[4:8])
		host = ip.String()
		port = binary.BigEndian.Uint16(buf[8:10])
		
	case 0x03: // Domain name
		if n < 5 {
			return "", 0, fmt.Errorf("invalid domain address length")
		}
		domainLen := int(buf[4])
		if n < 5+domainLen+2 {
			return "", 0, fmt.Errorf("incomplete domain address")
		}
		host = string(buf[5 : 5+domainLen])
		port = binary.BigEndian.Uint16(buf[5+domainLen : 5+domainLen+2])
		
	case 0x04: // IPv6
		if n < 22 {
			return "", 0, fmt.Errorf("invalid IPv6 address length")
		}
		ip := net.IP(buf[4:20])
		host = ip.String()
		port = binary.BigEndian.Uint16(buf[20:22])
		
	default:
		return "", 0, fmt.Errorf("unsupported address type: %d", addrType)
	}

	return host, port, nil
}

func (s *Server) startBidirectionalTransfer(clientConn, targetConn net.Conn, clientAddr, targetHost string, targetPort uint16) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	// 构建目标地址字符串
	targetAddr := fmt.Sprintf("%s:%d", targetHost, targetPort)

	// 监控连接状态
	connMonitor := make(chan error, 2)

	// 客户端到目标
	go func() {
		defer wg.Done()
		if err := s.copyData(ctx, targetConn, clientConn, 
			fmt.Sprintf("[%s -> %s]", clientAddr, targetAddr), connMonitor); err != nil {
			log.Printf("[%s%s%s -> %s%s%s] Error: %v", 
				colorCyan, clientAddr, colorReset,
				colorBlue, targetAddr, colorReset,
				err)
			cancel()
		}
	}()

	// 目标到客户端
	go func() {
		defer wg.Done()
		if err := s.copyData(ctx, clientConn, targetConn, 
			fmt.Sprintf("[%s <- %s]", clientAddr, targetAddr), connMonitor); err != nil {
			log.Printf("[%s%s%s <- %s%s%s] Error: %v", 
				colorCyan, clientAddr, colorReset,
				colorBlue, targetAddr, colorReset,
				err)
			cancel()
		}
	}()

	// 等待传输完成或出错
	go func() {
		wg.Wait()
		close(connMonitor)
	}()

	// 监控连接状态
	for err := range connMonitor {
		if err != nil {
			log.Printf("[%s%s%s <-> %s%s%s] Connection error: %v", 
				colorCyan, clientAddr, colorReset,
				colorBlue, targetAddr, colorReset,
				err)
		}
	}
}

func (s *Server) copyData(ctx context.Context, dst, src net.Conn, direction string, monitor chan<- error) error {
	buf := make([]byte, s.config.BufferSize)
	var totalBytes int64
	startTime := time.Now()
	lastProgressTime := startTime

	defer func() {
		duration := time.Since(startTime)
		speed := float64(totalBytes) / duration.Seconds() / 1024 / 1024
		log.Printf("%s Transfer completed: %s%.2f MB%s transferred in %s%.2fs%s (avg %.2f MB/s)", 
			direction,
			colorYellow, float64(totalBytes)/1024/1024, colorReset,
			colorPurple, duration.Seconds(), colorReset,
			speed)
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := src.SetReadDeadline(time.Now().Add(s.config.Timeout * 2)); err != nil {
				return fmt.Errorf("set read deadline: %w", err)
			}

			nr, err := src.Read(buf)
			if err != nil {
				if err == io.EOF {
					return nil
				}
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					if totalBytes > 0 {
						return nil
					}
				}
				return err
			}

			if nr > 0 {
				if err := dst.SetWriteDeadline(time.Now().Add(s.config.Timeout)); err != nil {
					return fmt.Errorf("set write deadline: %w", err)
				}

				nw, err := dst.Write(buf[:nr])
				if err != nil {
					return err
				}

				totalBytes += int64(nw)
				atomic.AddInt64(&s.stats.BytesTransferred, int64(nw))

				if time.Since(lastProgressTime) > 5*time.Second && totalBytes > 0 {
					speed := float64(totalBytes) / time.Since(startTime).Seconds() / 1024 / 1024
					log.Printf("%s Progress: %s%.2f MB%s transferred (%.2f MB/s)", 
						direction,
						colorYellow, float64(totalBytes)/1024/1024, colorReset,
						speed)
					lastProgressTime = time.Now()
				}
			}
		}
	}
}

func (s *Server) dialTargetWithRetry(host string, port uint16) (net.Conn, error) {
	var targetConn net.Conn
	var lastErr error

	for i := 0; i < s.config.MaxRetries; i++ {
		targetConn, lastErr = s.dialTarget(host, port)
		if lastErr == nil {
			return targetConn, nil
		}

		if i < s.config.MaxRetries-1 {
			log.Printf("Connection attempt %d/%d failed: %v, retrying in %v...", 
				i+1, s.config.MaxRetries, lastErr, s.config.RetryDelay)
			time.Sleep(s.config.RetryDelay)
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %v", s.config.MaxRetries, lastErr)
}

func (s *Server) dialTarget(host string, port uint16) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout:   s.config.DialTimeout,
		KeepAlive: 30 * time.Second,
	}

	// 如果是域名，先解析IP
	if net.ParseIP(host) == nil {
		ips, err := net.LookupIP(host)
		if err != nil {
			return nil, fmt.Errorf("DNS lookup failed: %w", err)
		}

		// 尝试所有可用的IP地址
		var lastErr error
		for _, ip := range ips {
			// 跳过IPv6地址(如果禁用)
			if !s.config.EnableIPv6 && ip.To4() == nil {
				continue
			}

			addr := fmt.Sprintf("%s:%d", ip.String(), port)
			conn, err := dialer.Dial("tcp", addr)
			if err == nil {
				log.Printf("Successfully connected to %s (%s)", host, addr)
				return conn, nil
			}
			lastErr = err
		}
		return nil, fmt.Errorf("failed to connect to any IP for %s: %v", host, lastErr)
	}

	// 直接IP地址
	addr := fmt.Sprintf("%s:%d", host, port)
	return dialer.Dial("tcp", addr)
} 