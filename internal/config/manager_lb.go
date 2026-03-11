package config

import (
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"
)

const (
	lbStrategyWeightedRR = "weighted_rr"
	lbMaxNodeWeight      = 100
)

type lbRouteItem struct {
	Domain         string
	ForceHTTPS     bool
	PoolID         string
	FallbackPoolID string
}

type lbWildcardItem struct {
	Suffix string
	Route  lbRouteItem
}

type compiledPoolNode struct {
	ID     string
	Target *url.URL
}

type compiledPool struct {
	ID       string
	Strategy string
	Nodes    []compiledPoolNode
	Ring     []int
	Counter  atomic.Uint64
}

// compileLoadBalancerConfig 将负载均衡配置编译为运行时快照结构。
// 参数：data 为待填充快照，lb 为负载均衡配置。
// 返回：编译失败错误。
// 异常：池节点上游地址非法时返回错误。
func compileLoadBalancerConfig(data *snapshot, lb LoadBalancerConfig) error {
	for _, pool := range lb.Pools {
		poolID := strings.TrimSpace(pool.ID)
		if poolID == "" {
			continue
		}
		compiled, err := compileLoadBalancerPool(pool)
		if err != nil {
			return err
		}
		data.lbPools[poolID] = compiled
	}

	for _, rule := range lb.Routes {
		domain := normalizeHost(rule.Domain)
		if domain == "" {
			continue
		}
		item := lbRouteItem{
			Domain:         domain,
			ForceHTTPS:     rule.ForceHTTPS,
			PoolID:         strings.TrimSpace(rule.PoolID),
			FallbackPoolID: strings.TrimSpace(rule.FallbackPoolID),
		}
		if strings.HasPrefix(domain, "*.") {
			data.lbWildcardRoutes = append(data.lbWildcardRoutes, lbWildcardItem{
				Suffix: domain[1:],
				Route:  item,
			})
			continue
		}
		data.lbExactRoutes[domain] = item
	}

	defaultPoolID := strings.TrimSpace(lb.DefaultPoolID)
	if defaultPoolID != "" {
		data.lbDefaultRoute = &lbRouteItem{
			Domain:     "_lb_default",
			PoolID:     defaultPoolID,
			ForceHTTPS: false,
		}
	}

	return nil
}

// compileLoadBalancerPool 编译单个流量池为可并发选路的结构。
// 参数：pool 为原始流量池配置。
// 返回：可直接用于选路的编译结果。
// 异常：节点上游地址非法时返回错误。
func compileLoadBalancerPool(pool LBPool) (*compiledPool, error) {
	strategy := strings.TrimSpace(pool.Strategy)
	if strategy == "" {
		strategy = lbStrategyWeightedRR
	}
	compiled := &compiledPool{
		ID:       strings.TrimSpace(pool.ID),
		Strategy: strategy,
		Nodes:    make([]compiledPoolNode, 0, len(pool.Nodes)),
	}
	ring := make([]int, 0, len(pool.Nodes))
	for idx, node := range pool.Nodes {
		if !node.Enabled {
			continue
		}
		upstreamURL, err := url.Parse(strings.TrimSpace(node.Upstream))
		if err != nil || upstreamURL.Host == "" || upstreamURL.Scheme == "" {
			return nil, fmt.Errorf("流量池 %q 节点 %q 的 upstream 非法", pool.ID, node.ID)
		}
		compiled.Nodes = append(compiled.Nodes, compiledPoolNode{
			ID:     strings.TrimSpace(node.ID),
			Target: upstreamURL,
		})
		weight := normalizeNodeWeight(node.Weight)
		targetIndex := len(compiled.Nodes) - 1
		for times := 0; times < weight; times++ {
			ring = append(ring, targetIndex)
		}
		if idx == 0 && weight == 0 {
			ring = append(ring, targetIndex)
		}
	}

	if len(compiled.Nodes) > 0 && len(ring) == 0 {
		for idx := range compiled.Nodes {
			ring = append(ring, idx)
		}
	}
	compiled.Ring = ring
	return compiled, nil
}

// normalizeNodeWeight 归一化节点权重，避免运行时环过大导致内存浪费。
// 参数：weight 为原始权重值。
// 返回：用于构建加权环的有效权重值。
// 异常：无。
func normalizeNodeWeight(weight int) int {
	if weight <= 0 {
		return 1
	}
	if weight > lbMaxNodeWeight {
		return lbMaxNodeWeight
	}
	return weight
}

// resolveLoadBalancerRoute 按域名查找负载均衡路由并执行池选路。
// 参数：normalizedHost 为规范化域名。
// 返回：命中后的路由结果与命中标识。
// 异常：无。
func (s *snapshot) resolveLoadBalancerRoute(normalizedHost string) (RouteMatch, bool) {
	if route, ok := s.lbExactRoutes[normalizedHost]; ok {
		if target, found := s.resolvePoolTarget(route.PoolID, route.FallbackPoolID); found {
			return RouteMatch{
				Domain:     route.Domain,
				ForceHTTPS: route.ForceHTTPS,
				Target:     target,
			}, true
		}
	}
	for _, item := range s.lbWildcardRoutes {
		if strings.HasSuffix(normalizedHost, item.Suffix) {
			if target, found := s.resolvePoolTarget(item.Route.PoolID, item.Route.FallbackPoolID); found {
				return RouteMatch{
					Domain:     item.Route.Domain,
					ForceHTTPS: item.Route.ForceHTTPS,
					Target:     target,
				}, true
			}
		}
	}
	if s.lbDefaultRoute != nil {
		if target, found := s.resolvePoolTarget(s.lbDefaultRoute.PoolID, s.lbDefaultRoute.FallbackPoolID); found {
			return RouteMatch{
				Domain:     s.lbDefaultRoute.Domain,
				ForceHTTPS: s.lbDefaultRoute.ForceHTTPS,
				Target:     target,
			}, true
		}
	}
	return RouteMatch{}, false
}

// resolvePoolTarget 按主池与回退池顺序选取可用上游地址。
// 参数：primaryPoolID 为主池，fallbackPoolID 为回退池。
// 返回：选中的上游地址与是否成功。
// 异常：无。
func (s *snapshot) resolvePoolTarget(primaryPoolID string, fallbackPoolID string) (*url.URL, bool) {
	if target, ok := s.pickTargetFromPool(primaryPoolID); ok {
		return target, true
	}
	if fallbackPoolID != "" {
		return s.pickTargetFromPool(fallbackPoolID)
	}
	return nil, false
}

// pickTargetFromPool 在给定流量池内执行无锁轮询选路。
// 参数：poolID 为目标流量池标识。
// 返回：选中的上游地址与是否成功。
// 异常：无。
func (s *snapshot) pickTargetFromPool(poolID string) (*url.URL, bool) {
	compiled := s.lbPools[strings.TrimSpace(poolID)]
	if compiled == nil || len(compiled.Ring) == 0 {
		return nil, false
	}
	next := compiled.Counter.Add(1)
	index := compiled.Ring[int(next%uint64(len(compiled.Ring)))]
	if index < 0 || index >= len(compiled.Nodes) {
		return nil, false
	}
	return compiled.Nodes[index].Target, true
}
