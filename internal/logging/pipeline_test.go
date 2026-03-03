package logging

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

// TestPipelineSnapshot_CountsEvents 验证统计快照会累加不同事件类型计数。
func TestPipelineSnapshot_CountsEvents(t *testing.T) {
	pipeline := NewPipeline(8, zap.NewNop())
	defer pipeline.Close()

	pipeline.Emit(Event{Level: "info", Type: "proxy"})
	pipeline.Emit(Event{Level: "warn", Type: "blocked"})
	pipeline.Emit(Event{Level: "info", Type: "other"})

	stats := pipeline.Snapshot()
	if stats.Total != 3 {
		t.Fatalf("Total 不匹配: got=%d", stats.Total)
	}
	if stats.Proxied != 1 {
		t.Fatalf("Proxied 不匹配: got=%d", stats.Proxied)
	}
	if stats.Blocked != 1 {
		t.Fatalf("Blocked 不匹配: got=%d", stats.Blocked)
	}
}

// TestPipelineEmit_IgnoresAfterClose 验证 Close 后 Emit 不再改变统计。
func TestPipelineEmit_IgnoresAfterClose(t *testing.T) {
	pipeline := NewPipeline(8, zap.NewNop())
	pipeline.Emit(Event{Level: "info", Type: "proxy"})
	before := pipeline.Snapshot()

	pipeline.Close()
	pipeline.Emit(Event{Level: "info", Type: "proxy"})
	after := pipeline.Snapshot()

	if after.Total != before.Total {
		t.Fatalf("关闭后仍在计数: before=%d after=%d", before.Total, after.Total)
	}
}

// TestPipelineWorker_CoversLevelBranches 验证 worker 针对不同日志等级分支可运行且不阻塞。
func TestPipelineWorker_CoversLevelBranches(t *testing.T) {
	pipeline := &Pipeline{
		ch:     make(chan Event, 16),
		logger: zap.NewNop(),
	}
	done := make(chan struct{})
	go func() {
		pipeline.worker()
		close(done)
	}()

	pipeline.ch <- Event{Level: "debug", Type: "proxy", Message: "m"}
	pipeline.ch <- Event{Level: "warn", Type: "proxy", Message: "m"}
	pipeline.ch <- Event{Level: "error", Type: "proxy", Message: "m"}
	pipeline.ch <- Event{Level: "dpanic", Type: "proxy", Message: "m"}
	pipeline.ch <- Event{Level: "panic", Type: "proxy", Message: "m"}
	pipeline.ch <- Event{Level: "fatal", Type: "proxy", Message: "m"}
	pipeline.ch <- Event{Level: "unknown", Type: "proxy", Message: "m"}
	close(pipeline.ch)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("worker 未及时退出")
	}
}
