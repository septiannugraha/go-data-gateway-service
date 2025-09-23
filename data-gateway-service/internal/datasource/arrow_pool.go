package datasource

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/apache/arrow-go/v18/arrow/flight"
	pb "github.com/apache/arrow-go/v18/arrow/flight/gen/flight"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

var (
	ErrPoolClosed   = errors.New("pool is closed")
	ErrPoolExhausted = errors.New("connection pool exhausted")
	ErrInvalidConfig = errors.New("invalid pool configuration")
)

// PoolConfig defines the connection pool configuration
type PoolConfig struct {
	MaxConnections     int           // Maximum number of connections in pool
	MinConnections     int           // Minimum number of idle connections
	MaxIdleTime        time.Duration // Maximum time a connection can be idle
	ConnectionTimeout  time.Duration // Timeout for creating new connections
	HealthCheckInterval time.Duration // Interval for health checks
}

// DefaultPoolConfig returns sensible defaults
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxConnections:     10,
		MinConnections:     2,
		MaxIdleTime:        30 * time.Minute,
		ConnectionTimeout:  10 * time.Second,
		HealthCheckInterval: 1 * time.Minute,
	}
}

// ArrowConnection wraps a Flight client with metadata
type ArrowConnection struct {
	client      flight.Client
	lastUsed    time.Time
	inUse       bool
	id          string
	healthCheck time.Time
}

// ArrowConnectionPool manages a pool of Arrow Flight connections
type ArrowConnectionPool struct {
	config      *PoolConfig
	dremioConfig *DremioConfig
	logger      *zap.Logger

	connections []*ArrowConnection
	mu          sync.RWMutex
	closed      bool

	// Metrics
	metrics struct {
		totalConnections   int64
		activeConnections  int64
		failedConnections  int64
		totalRequests      int64
		poolExhausted      int64
	}

	// Wait group for graceful shutdown
	wg sync.WaitGroup
}

// NewArrowConnectionPool creates a new connection pool
func NewArrowConnectionPool(dremioConfig *DremioConfig, poolConfig *PoolConfig, logger *zap.Logger) (*ArrowConnectionPool, error) {
	if poolConfig == nil {
		poolConfig = DefaultPoolConfig()
	}

	// Validate configuration
	if poolConfig.MaxConnections < 1 {
		return nil, fmt.Errorf("%w: max connections must be at least 1", ErrInvalidConfig)
	}
	if poolConfig.MinConnections > poolConfig.MaxConnections {
		return nil, fmt.Errorf("%w: min connections cannot exceed max connections", ErrInvalidConfig)
	}

	pool := &ArrowConnectionPool{
		config:       poolConfig,
		dremioConfig: dremioConfig,
		logger:       logger,
		connections:  make([]*ArrowConnection, 0, poolConfig.MaxConnections),
	}

	// Pre-create minimum connections
	for i := 0; i < poolConfig.MinConnections; i++ {
		conn, err := pool.createConnection()
		if err != nil {
			logger.Warn("Failed to create initial connection",
				zap.Int("index", i),
				zap.Error(err))
			continue
		}
		pool.connections = append(pool.connections, conn)
	}

	// Start health check routine
	pool.wg.Add(1)
	go pool.healthCheckRoutine()

	// Start idle connection cleanup
	pool.wg.Add(1)
	go pool.idleCleanupRoutine()

	logger.Info("Arrow connection pool initialized",
		zap.Int("initial_connections", len(pool.connections)),
		zap.Int("max_connections", poolConfig.MaxConnections))

	return pool, nil
}

// Get acquires a connection from the pool
func (p *ArrowConnectionPool) Get(ctx context.Context) (*ArrowConnection, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, ErrPoolClosed
	}

	p.metrics.totalRequests++

	// Try to find an idle connection
	for _, conn := range p.connections {
		if !conn.inUse {
			conn.inUse = true
			conn.lastUsed = time.Now()
			p.metrics.activeConnections++

			p.logger.Debug("Connection acquired from pool",
				zap.String("conn_id", conn.id),
				zap.Int("pool_size", len(p.connections)))

			return conn, nil
		}
	}

	// Create new connection if under limit
	if len(p.connections) < p.config.MaxConnections {
		conn, err := p.createConnection()
		if err != nil {
			p.metrics.failedConnections++
			return nil, fmt.Errorf("failed to create new connection: %w", err)
		}

		conn.inUse = true
		conn.lastUsed = time.Now()
		p.connections = append(p.connections, conn)
		p.metrics.totalConnections++
		p.metrics.activeConnections++

		p.logger.Info("Created new connection",
			zap.String("conn_id", conn.id),
			zap.Int("pool_size", len(p.connections)))

		return conn, nil
	}

	// Pool exhausted
	p.metrics.poolExhausted++
	return nil, ErrPoolExhausted
}

// Put returns a connection to the pool
func (p *ArrowConnectionPool) Put(conn *ArrowConnection) {
	if conn == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	conn.inUse = false
	conn.lastUsed = time.Now()
	p.metrics.activeConnections--

	p.logger.Debug("Connection returned to pool",
		zap.String("conn_id", conn.id),
		zap.Int("active", int(p.metrics.activeConnections)))
}

// createConnection creates a new Arrow Flight connection
func (p *ArrowConnectionPool) createConnection() (*ArrowConnection, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.config.ConnectionTimeout)
	defer cancel()

	// Create gRPC connection options
	var dialOpts []grpc.DialOption
	if !p.dremioConfig.UseTLS {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Create Flight client
	addr := fmt.Sprintf("%s:%d", p.dremioConfig.Host, p.dremioConfig.Port)
	flightClient, err := flight.NewClientWithMiddleware(addr, nil, nil, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create flight client: %w", err)
	}

	// Authenticate
	authCtx := metadata.AppendToOutgoingContext(ctx,
		"authorization", "Basic "+basicAuth(p.dremioConfig.Username, p.dremioConfig.Password))

	// Test connection with a simple action
	_, err = flightClient.ListActions(authCtx, &pb.Empty{})
	if err != nil {
		flightClient.Close()
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}

	connID := fmt.Sprintf("conn-%d-%d", time.Now().Unix(), len(p.connections))

	return &ArrowConnection{
		client:      flightClient,
		lastUsed:    time.Now(),
		id:          connID,
		healthCheck: time.Now(),
	}, nil
}

// healthCheckRoutine periodically checks connection health
func (p *ArrowConnectionPool) healthCheckRoutine() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.performHealthChecks()
		}

		p.mu.RLock()
		if p.closed {
			p.mu.RUnlock()
			return
		}
		p.mu.RUnlock()
	}
}

// performHealthChecks tests all idle connections
func (p *ArrowConnectionPool) performHealthChecks() {
	p.mu.Lock()
	defer p.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	authCtx := metadata.AppendToOutgoingContext(ctx,
		"authorization", "Basic "+basicAuth(p.dremioConfig.Username, p.dremioConfig.Password))

	var healthyConns []*ArrowConnection
	for _, conn := range p.connections {
		if conn.inUse {
			healthyConns = append(healthyConns, conn)
			continue
		}

		// Test connection
		_, err := conn.client.ListActions(authCtx, &pb.Empty{})
		if err != nil {
			p.logger.Warn("Connection failed health check",
				zap.String("conn_id", conn.id),
				zap.Error(err))
			conn.client.Close()
			continue
		}

		conn.healthCheck = time.Now()
		healthyConns = append(healthyConns, conn)
	}

	p.connections = healthyConns

	p.logger.Debug("Health check completed",
		zap.Int("healthy_connections", len(healthyConns)))
}

// idleCleanupRoutine removes idle connections exceeding max idle time
func (p *ArrowConnectionPool) idleCleanupRoutine() {
	defer p.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.cleanupIdleConnections()
		}

		p.mu.RLock()
		if p.closed {
			p.mu.RUnlock()
			return
		}
		p.mu.RUnlock()
	}
}

// cleanupIdleConnections removes connections that have been idle too long
func (p *ArrowConnectionPool) cleanupIdleConnections() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	var activeConns []*ArrowConnection

	for _, conn := range p.connections {
		// Keep minimum connections
		if len(activeConns) < p.config.MinConnections {
			activeConns = append(activeConns, conn)
			continue
		}

		// Keep in-use connections
		if conn.inUse {
			activeConns = append(activeConns, conn)
			continue
		}

		// Check idle time
		if now.Sub(conn.lastUsed) > p.config.MaxIdleTime {
			p.logger.Info("Closing idle connection",
				zap.String("conn_id", conn.id),
				zap.Duration("idle_time", now.Sub(conn.lastUsed)))
			conn.client.Close()
			continue
		}

		activeConns = append(activeConns, conn)
	}

	p.connections = activeConns
}

// GetMetrics returns pool metrics
func (p *ArrowConnectionPool) GetMetrics() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]interface{}{
		"total_connections":   p.metrics.totalConnections,
		"active_connections":  p.metrics.activeConnections,
		"pool_size":          len(p.connections),
		"failed_connections": p.metrics.failedConnections,
		"total_requests":     p.metrics.totalRequests,
		"pool_exhausted":     p.metrics.poolExhausted,
		"max_connections":    p.config.MaxConnections,
	}
}

// Close gracefully shuts down the pool
func (p *ArrowConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	// Close all connections
	for _, conn := range p.connections {
		if err := conn.client.Close(); err != nil {
			p.logger.Warn("Error closing connection",
				zap.String("conn_id", conn.id),
				zap.Error(err))
		}
	}

	p.connections = nil

	// Wait for routines to finish
	go func() {
		p.wg.Wait()
	}()

	p.logger.Info("Connection pool closed",
		zap.Int64("total_requests", p.metrics.totalRequests),
		zap.Int64("failed_connections", p.metrics.failedConnections))

	return nil
}

// WithConnection executes a function with a pooled connection
func (p *ArrowConnectionPool) WithConnection(ctx context.Context, fn func(flight.Client) error) error {
	conn, err := p.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get connection from pool: %w", err)
	}
	defer p.Put(conn)

	return fn(conn.client)
}