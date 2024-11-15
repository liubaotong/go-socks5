package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"github.com/google/uuid"
)

type ConnectionStats struct {
	ActiveConnections int64
	TotalConnections int64
	BytesTransferred int64
	StartTime       time.Time
}

type Server struct {
	config     *Config
	listener   net.Listener
	stats      *ConnectionStats
	connLimit  chan struct{} // 连接数限制器
	activeConns sync.Map    // 活跃连接管理
}

func NewServer(config *Config) *Server {
	return &Server{
		config: config,
		stats: &ConnectionStats{
			StartTime: time.Now(),
		},
		connLimit: make(chan struct{}, maxConcurrentConnections),
		activeConns: sync.Map{},
	}
}

func (s *Server) startStatsReporter(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			uptime := time.Since(s.stats.StartTime)
			log.Printf("Server Stats - Uptime: %s, Active Connections: %d, Total Connections: %d, Total Bytes Transferred: %.2f MB",
				uptime.Round(time.Second),
				atomic.LoadInt64(&s.stats.ActiveConnections),
				atomic.LoadInt64(&s.stats.TotalConnections),
				float64(atomic.LoadInt64(&s.stats.BytesTransferred))/1024/1024)
		}
	}
}

func (s *Server) Run() error {
	listener, err := net.Listen("tcp", s.config.ListenAddr)
	if err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}
	s.listener = listener
	defer listener.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go s.startStatsReporter(ctx)

	log.Printf("SOCKS5 proxy server listening on %s (max connections: %d)", 
		s.config.ListenAddr, maxConcurrentConnections)

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		log.Printf("Shutting down server...")
		cancel()
		listener.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			select {
			case s.connLimit <- struct{}{}:
				conn, err := listener.Accept()
				if err != nil {
					<-s.connLimit
					if ne, ok := err.(net.Error); ok && ne.Temporary() {
						log.Printf("Temporary accept error: %v", err)
						time.Sleep(time.Millisecond * 100)
						continue
					}
					return err
				}
				atomic.AddInt64(&s.stats.TotalConnections, 1)
				atomic.AddInt64(&s.stats.ActiveConnections, 1)
				
				connID := uuid.New().String()
				s.activeConns.Store(connID, conn)
				
				go func() {
					defer func() {
						atomic.AddInt64(&s.stats.ActiveConnections, -1)
						s.activeConns.Delete(connID)
						<-s.connLimit
					}()
					s.handleConnection(conn)
				}()
			default:
				log.Printf("Connection limit reached, waiting...")
				time.Sleep(time.Millisecond * 100)
			}
		}
	}
}

func (s *Server) GetStats() ConnectionStats {
	return ConnectionStats{
		ActiveConnections: atomic.LoadInt64(&s.stats.ActiveConnections),
		TotalConnections: atomic.LoadInt64(&s.stats.TotalConnections),
		BytesTransferred: atomic.LoadInt64(&s.stats.BytesTransferred),
		StartTime:        s.stats.StartTime,
	}
} 