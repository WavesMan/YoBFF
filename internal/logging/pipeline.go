package logging

import (
	"strings"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// Event 定义网关异步日志事件的统一结构。
type Event struct {
	Time      time.Time `json:"time"`
	Level     string    `json:"level"`
	Type      string    `json:"type"`
	ClientIP  string    `json:"clientIp"`
	Host      string    `json:"host"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Status    int       `json:"status"`
	LatencyMS int64     `json:"latencyMs"`
	Message   string    `json:"message"`
}

// Stats 描述日志管线运行统计指标。
type Stats struct {
	Total   uint64 `json:"total"`
	Blocked uint64 `json:"blocked"`
	Proxied uint64 `json:"proxied"`
	Dropped uint64 `json:"dropped"`
}

// Pipeline 负责异步消费日志事件并输出到控制台。
type Pipeline struct {
	ch      chan Event
	logger  *zap.Logger
	closed  atomic.Bool
	total   atomic.Uint64
	blocked atomic.Uint64
	proxied atomic.Uint64
	dropped atomic.Uint64
}

// NewPipeline 初始化异步日志管线并启动消费协程。
// 参数：buffer 为事件缓冲队列大小，logger 为输出日志实例。
// 返回：可复用的日志管线对象。
// 异常：无。
func NewPipeline(buffer int, logger *zap.Logger) *Pipeline {
	if logger == nil {
		logger = zap.NewNop()
	}
	p := &Pipeline{
		ch:     make(chan Event, buffer),
		logger: logger,
	}
	go p.worker()
	return p
}

// Emit 提交单条日志事件到异步管线。
// 参数：event 为待写入事件。
// 返回：无。
// 异常：当队列满时丢弃事件并累加 dropped 指标。
func (p *Pipeline) Emit(event Event) {
	if p.closed.Load() {
		return
	}
	event.Time = time.Now()
	switch event.Type {
	case "blocked":
		p.blocked.Add(1)
	case "proxy":
		p.proxied.Add(1)
	}
	p.total.Add(1)

	select {
	case p.ch <- event:
	default:
		p.dropped.Add(1)
	}
}

// Snapshot 获取当前管线统计快照。
// 参数：无。
// 返回：累计总量、拦截量、转发量与丢弃量。
// 异常：无。
func (p *Pipeline) Snapshot() Stats {
	return Stats{
		Total:   p.total.Load(),
		Blocked: p.blocked.Load(),
		Proxied: p.proxied.Load(),
		Dropped: p.dropped.Load(),
	}
}

// Close 关闭日志管线并结束后台消费。
// 参数：无。
// 返回：无。
// 异常：重复关闭将被安全忽略。
func (p *Pipeline) Close() {
	if !p.closed.CompareAndSwap(false, true) {
		return
	}
	close(p.ch)
}

// worker 持续消费队列事件并按等级输出控制台日志。
// 参数：无。
// 返回：无。
// 异常：无，单条日志写入失败不会中断消费循环。
func (p *Pipeline) worker() {
	for event := range p.ch {
		logger := p.logger.With(
			zap.String("type", event.Type),
			zap.String("client_ip", event.ClientIP),
			zap.String("host", event.Host),
			zap.String("method", event.Method),
			zap.String("path", event.Path),
			zap.Int("status", event.Status),
			zap.Int64("latency_ms", event.LatencyMS),
		)
		level := strings.ToLower(strings.TrimSpace(event.Level))
		switch level {
		case "debug":
			logger.Debug(event.Message)
		case "warn", "warning":
			logger.Warn(event.Message)
		case "error":
			logger.Error(event.Message)
		case "dpanic":
			logger.Error("dpanic 级别事件", zap.String("message", event.Message))
		case "panic":
			logger.Error("panic 级别事件", zap.String("message", event.Message))
		case "fatal":
			logger.Error("fatal 级别事件", zap.String("message", event.Message))
		default:
			logger.Info(event.Message)
		}
	}
}
