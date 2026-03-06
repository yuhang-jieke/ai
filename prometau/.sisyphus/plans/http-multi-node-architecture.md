# HTTP 多节点启动架构设计

## API 设计

```go
package gospacex

// HTTP 命名空间
var http = &HTTPNamespace{}

type HTTPNamespace struct {
    launcher *Launcher
}

// 启动 HTTP 服务
// 单节点：http.launch()
// 多节点：http.launch([]string{"127.0.0.1:8081", "127.0.0.1:8082"})
func (h *HTTPNamespace) launch(nodes ...interface{}) error {
    addrs := parseAddresses(nodes)
    return h.launcher.Start(addrs)
}

// 停止所有节点
func (h *HTTPNamespace) shutdown() error {
    return h.launcher.Stop()
}

// 获取节点状态
func (h *HTTPNamespace) status() []*NodeStatus {
    return h.launcher.GetStatus()
}
```

## 核心实现

### 1. Launcher - 启动器

```go
type Launcher struct {
    config     *ServerConfig
    nodes      map[string]*Node
    mu         sync.RWMutex
    wg         sync.WaitGroup
    stopChan   chan struct{}
    errHandler ErrorHandler
}

type Node struct {
    Addr       string
    Server     *http.Server
    Handler    http.Handler
    Status     NodeStatus
    RestartCount int
    lastError  error
}

type NodeStatus string

const (
    StatusRunning   NodeStatus = "running"
    StatusStopped   NodeStatus = "stopped"
    StatusFailed    NodeStatus = "failed"
    StatusStarting  NodeStatus = "starting"
    StatusRestarting NodeStatus = "restarting"
)

func NewLauncher(config *ServerConfig) *Launcher {
    return &Launcher{
        config:   config,
        nodes:    make(map[string]*Node),
        stopChan: make(chan struct{}),
    }
}

func (l *Launcher) Start(addrs []string) error {
    if len(addrs) == 0 {
        // 默认单节点
        addrs = []string{l.config.DefaultAddr}
    }

    for _, addr := range addrs {
        if err := l.startNode(addr); err != nil {
            return fmt.Errorf("failed to start node %s: %w", addr, err)
        }
    }

    // 启动监控协程
    go l.monitorNodes()

    return nil
}

func (l *Launcher) startNode(addr string) error {
    l.mu.Lock()
    defer l.mu.Unlock()

    if _, exists := l.nodes[addr]; exists {
        return fmt.Errorf("node %s already exists", addr)
    }

    // 创建 HTTP 服务器
    handler := l.createHandler()
    server := &http.Server{
        Addr:         addr,
        Handler:      handler,
        ReadTimeout:  l.config.ReadTimeout,
        WriteTimeout: l.config.WriteTimeout,
    }

    node := &Node{
        Addr:   addr,
        Server: server,
        Handler: handler,
        Status: StatusStarting,
    }

    l.nodes[addr] = node
    l.wg.Add(1)

    // 启动节点
    go l.runNode(node)

    return nil
}

func (l *Launcher) runNode(node *Node) {
    defer l.wg.Done()

    for {
        node.Status = StatusRunning
        err := node.Server.ListenAndServe()

        // 检查是否是正常关闭
        select {
        case <-l.stopChan:
            node.Status = StatusStopped
            return
        default:
        }

        // 节点异常退出
        node.Status = StatusFailed
        node.lastError = err

        if l.errHandler != nil {
            l.errHandler(node.Addr, err)
        }

        // 自动重启（带退避）
        if node.RestartCount < l.config.MaxRestarts {
            node.Status = StatusRestarting
            node.RestartCount++
            time.Sleep(l.getBackoffDelay(node.RestartCount))
            continue
        }

        // 超过最大重启次数，停止
        return
    }
}

func (l *Launcher) monitorNodes() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-l.stopChan:
            return
        case <-ticker.C:
            l.checkNodeHealth()
        }
        }
}

func (l *Launcher) Stop() error {
    close(l.stopChan)

    l.mu.RLock()
    defer l.mu.RUnlock()

    // 优雅关闭所有节点
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    for _, node := range l.nodes {
        node.Server.Shutdown(ctx)
    }

    // 等待所有 goroutine 结束
    done := make(chan struct{})
    go func() {
        l.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        return nil
    case <-ctx.Done():
        return fmt.Errorf("shutdown timeout")
    }
}

func (l *Launcher) GetStatus() []*NodeStatus {
    l.mu.RLock()
    defer l.mu.RUnlock()

    statuses := make([]*NodeStatus, 0, len(l.nodes))
    for addr, node := range l.nodes {
        statuses = append(statuses, &NodeStatus{
            Addr:         addr,
            Status:       node.Status,
            RestartCount: node.RestartCount,
            LastError:    node.lastError,
            Uptime:       l.getNodeUptime(node),
        })
    }

    return statuses
}
```

### 2. 配置结构

```go
type ServerConfig struct {
    DefaultAddr    string        `yaml:"default_addr" json:"default_addr"`
    ReadTimeout    time.Duration `yaml:"read_timeout" json:"read_timeout"`
    WriteTimeout   time.Duration `yaml:"write_timeout" json:"write_timeout"`
    MaxRestarts    int           `yaml:"max_restarts" json:"max_restarts"`
    RestartBaseDelay time.Duration `yaml:"restart_base_delay" json:"restart_base_delay"`
    Mode           string        `yaml:"mode" json:"mode"`
}

func DefaultServerConfig() *ServerConfig {
    return &ServerConfig{
        DefaultAddr:      "0.0.0.0:8080",
        ReadTimeout:      30 * time.Second,
        WriteTimeout:     30 * time.Second,
        MaxRestarts:      3,
        RestartBaseDelay: time.Second,
        Mode:             "release",
    }
}
```

### 3. 使用示例

```go
package main

import "gospacex"

func main() {
    // 配置
    gospacex.http.configure(&gospacex.ServerConfig{
        DefaultAddr: "0.0.0.0:8080",
        Mode: "release",
    })

    // 单节点启动
    err := gospacex.http.launch()
    if err != nil {
        log.Fatal(err)
    }

    // 或多节点启动
    // err := gospacex.http.launch(
    //     "192.168.0.1:8081",
    //     "192.168.0.1:8082",
    //     "192.168.0.1:8083",
    // )

    // 优雅关闭
    defer gospacex.http.shutdown()

    // 等待信号
    signalChan := make(chan os.Signal, 1)
    signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
    <-signalChan
}

// 查看节点状态
// curl http://localhost:8080/_status
```

### 4. 高级功能

#### 4.1 健康检查
```go
func (l *Launcher) checkNodeHealth() {
    l.mu.RLock()
    defer l.mu.RUnlock()

    for addr, node := range l.nodes {
        if node.Status != StatusRunning {
            continue
        }

        // 发送健康检查请求
        resp, err := http.Get(fmt.Sprintf("http://%s/health", addr))
        if err != nil || resp.StatusCode != 200 {
            // 节点不健康，触发重启
            l.restartNode(addr)
        }
    }
}
```

#### 4.2 负载均衡器
```go
type LoadBalancer struct {
    nodes   []*Node
    current int64
    mu      sync.RWMutex
    strategy string // round-robin, least-conn, ip-hash
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    node := lb.selectNode(r)
    if node == nil {
        http.Error(w, "no available nodes", http.StatusServiceUnavailable)
        return
    }

    // 代理请求到选定节点
    lb.proxyRequest(w, r, node)
}
```

#### 4.3 指标监控
```go
type Metrics struct {
    RequestsTotal   *prometheus.CounterVec
    RequestDuration *prometheus.HistogramVec
    NodeStatus      *prometheus.GaugeVec
}

func (l *Launcher) RegisterMetrics() {
    prometheus.MustRegister(
        prometheus.NewGaugeFunc(
            prometheus.GaugeOpts{
                Name: "gospacex_http_nodes_running",
                Help: "Number of running HTTP nodes",
            },
            func() float64 {
                return float64(l.countRunningNodes())
            },
        ),
    )
}
```

## 配置示例

```yaml
http:
  default_addr: "0.0.0.0:8080"
  read_timeout: 30s
  write_timeout: 30s
  max_restarts: 3
  restart_base_delay: 1s
  mode: "release"
  nodes:
    - addr: "192.168.0.1:8081"
      weight: 1
    - addr: "192.168.0.1:8082"
      weight: 2
  health_check:
    enabled: true
    interval: 30s
    path: "/health"
  load_balancer:
    enabled: true
    strategy: "round-robin"
```

## 关键设计决策

| 决策 | 理由 |
|------|------|
| 多进程 vs 多线程 | 单进程多线程，简单场景够用 |
| 自动重启 | 提高可用性，带退避避免无限重启 |
| 优雅关闭 | 避免请求中断，等待处理完成 |
| 健康检查 | 及时发现并恢复故障节点 |
| 配置驱动 | 支持 YAML/JSON/环境变量 |

## 扩展方向

1. **集群模式** - 跨多机部署
2. **服务发现** - 集成 Consul/Etcd
3. **动态扩缩容** - 根据负载自动调整节点数
4. **热重载** - 配置变更无需重启
