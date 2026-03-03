package admin

import (
	"sync"
	"time"
)

type loginGuard struct {
	mu      sync.Mutex
	ttl     time.Duration
	entries map[string]loginGuardEntry
}

type loginGuardEntry struct {
	failed    int
	updatedAt time.Time
}

// newLoginGuard 创建登录防爆破状态机，用于控制验证码触发。
// 参数：ttl 为失败状态的保留时间，非正数时使用默认值。
// 返回：登录防护实例。
// 异常：无。
func newLoginGuard(ttl time.Duration) *loginGuard {
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	return &loginGuard{
		ttl:     ttl,
		entries: make(map[string]loginGuardEntry),
	}
}

// RequireCaptcha 判断当前客户端是否需要验证码。
// 参数：key 为客户端标识（通常为 IP）。
// 返回：true 表示需要验证码。
// 异常：无。
func (g *loginGuard) RequireCaptcha(key string) bool {
	if g == nil {
		return false
	}
	now := time.Now()
	g.mu.Lock()
	defer g.mu.Unlock()
	entry, ok := g.entries[key]
	if !ok {
		return false
	}
	if now.Sub(entry.updatedAt) > g.ttl {
		delete(g.entries, key)
		return false
	}
	return entry.failed >= 1
}

// OnFailure 记录一次登录失败，使后续登录触发验证码校验。
// 参数：key 为客户端标识（通常为 IP）。
// 返回：无。
// 异常：无。
func (g *loginGuard) OnFailure(key string) {
	if g == nil {
		return
	}
	now := time.Now()
	g.mu.Lock()
	defer g.mu.Unlock()
	entry := g.entries[key]
	entry.failed++
	entry.updatedAt = now
	g.entries[key] = entry
}

// OnSuccess 清理客户端失败状态，使后续登录不再要求验证码。
// 参数：key 为客户端标识（通常为 IP）。
// 返回：无。
// 异常：无。
func (g *loginGuard) OnSuccess(key string) {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.entries, key)
}
