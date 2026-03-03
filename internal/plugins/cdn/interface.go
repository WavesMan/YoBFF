package cdn

import "context"

// Provider 定义 CDN 服务商插件接口。
// 所有 CDN 插件必须实现此接口以提供回源 IP 同步能力。
type Provider interface {
	// Name 返回插件名称（如 "cloudflare"）。
	Name() string

	// FetchCIDRs 获取回源 IP CIDR 列表。
	// 上下文 ctx 用于控制超时。
	FetchCIDRs(ctx context.Context) ([]string, error)
}
