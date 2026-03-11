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

// hasLoadBalancerRuleForDomain 判断给定域名是否已由负载均衡规则接管。
// 参数：domain 为已标准化域名（支持精确域名或通配域名表达）。
// 返回：true 表示应优先使用负载均衡规则。
// 异常：无。
func (s *snapshot) hasLoadBalancerRuleForDomain(domain string) bool {
	if domain == "" {
		return false
	}
	if _, ok := s.lbExactRoutes[domain]; ok {
		return true
	}
	if strings.HasPrefix(domain, "*.") {
		suffix := domain[1:]
		for _, item := range s.lbWildcardRoutes {
			if item.Suffix == suffix {
				return true
			}
		}
		return false
	}
	for _, item := range s.lbWildcardRoutes {
		if strings.HasSuffix(domain, item.Suffix) {
			return true
		}
	}
	return false
}

func prepareRoutingAndLoadBalancer(cfg Config) (Config, []ValidationIssue) {
	result := cfg
	if hasLoadBalancerBinding(result.LoadBalancer) {
		return result, checkRoutingLoadBalancerConflict(result.Routing, result.LoadBalancer)
	}
	if !hasLegacyRouting(result.Routing) {
		return result, nil
	}
	result.LoadBalancer = migrateRoutingToLoadBalancer(result.Routing, result.LoadBalancer)
	result.Routing = RoutingConfig{}
	return result, nil
}

func hasLoadBalancerBinding(lb LoadBalancerConfig) bool {
	if strings.TrimSpace(lb.DefaultPoolID) != "" {
		return true
	}
	return len(lb.Pools) > 0 || len(lb.Routes) > 0
}

func hasLegacyRouting(routing RoutingConfig) bool {
	if strings.TrimSpace(routing.DefaultUpstream) != "" {
		return true
	}
	return len(routing.Domains) > 0
}

func checkRoutingLoadBalancerConflict(routing RoutingConfig, lb LoadBalancerConfig) []ValidationIssue {
	issues := make([]ValidationIssue, 0)
	if !hasLegacyRouting(routing) || !hasLoadBalancerBinding(lb) {
		return issues
	}

	index := &snapshot{
		lbExactRoutes:    make(map[string]lbRouteItem),
		lbWildcardRoutes: make([]lbWildcardItem, 0),
	}
	for _, item := range lb.Routes {
		domain := normalizeHost(item.Domain)
		if domain == "" {
			continue
		}
		routeItemValue := lbRouteItem{
			Domain:         domain,
			ForceHTTPS:     item.ForceHTTPS,
			PoolID:         strings.TrimSpace(item.PoolID),
			FallbackPoolID: strings.TrimSpace(item.FallbackPoolID),
		}
		if strings.HasPrefix(domain, "*.") {
			index.lbWildcardRoutes = append(index.lbWildcardRoutes, lbWildcardItem{
				Suffix: domain[1:],
				Route:  routeItemValue,
			})
			continue
		}
		index.lbExactRoutes[domain] = routeItemValue
	}

	for idx, item := range routing.Domains {
		domain := normalizeHost(item.Domain)
		if domain == "" {
			continue
		}
		if index.hasLoadBalancerRuleForDomain(domain) {
			issues = append(issues, ValidationIssue{
				Path:    fmt.Sprintf("routing.domains[%d].domain", idx),
				Message: fmt.Sprintf("domain %q conflicts with loadBalancer.routes", item.Domain),
			})
		}
	}
	if strings.TrimSpace(routing.DefaultUpstream) != "" && strings.TrimSpace(lb.DefaultPoolID) != "" {
		issues = append(issues, ValidationIssue{
			Path:    "routing.defaultUpstream",
			Message: "defaultUpstream conflicts with loadBalancer.defaultPoolId",
		})
	}
	return issues
}

func migrateRoutingToLoadBalancer(routing RoutingConfig, lb LoadBalancerConfig) LoadBalancerConfig {
	result := lb
	result.Pools = append([]LBPool(nil), result.Pools...)
	result.Routes = append([]LBRouteRule(nil), result.Routes...)

	usedPoolID := make(map[string]struct{}, len(result.Pools))
	for _, item := range result.Pools {
		poolID := strings.TrimSpace(item.ID)
		if poolID != "" {
			usedPoolID[poolID] = struct{}{}
		}
	}

	for _, item := range routing.Domains {
		domain := normalizeHost(item.Domain)
		upstream := strings.TrimSpace(item.Upstream)
		if domain == "" || upstream == "" {
			continue
		}
		poolID := buildMigratedPoolID(domain, usedPoolID)
		result.Pools = append(result.Pools, LBPool{
			ID:       poolID,
			Name:     "迁移池_" + domain,
			Strategy: lbStrategyWeightedRR,
			Nodes: []LBNode{
				{
					ID:       "node_primary",
					Upstream: upstream,
					Weight:   1,
					Enabled:  true,
				},
			},
		})
		result.Routes = append(result.Routes, LBRouteRule{
			Domain:     item.Domain,
			PoolID:     poolID,
			ForceHTTPS: item.ForceHTTPS,
		})
	}

	defaultUpstream := strings.TrimSpace(routing.DefaultUpstream)
	if defaultUpstream != "" {
		poolID := buildMigratedPoolID("default", usedPoolID)
		result.Pools = append(result.Pools, LBPool{
			ID:       poolID,
			Name:     "迁移池_default",
			Strategy: lbStrategyWeightedRR,
			Nodes: []LBNode{
				{
					ID:       "node_primary",
					Upstream: defaultUpstream,
					Weight:   1,
					Enabled:  true,
				},
			},
		})
		result.DefaultPoolID = poolID
	}

	return result
}

func buildMigratedPoolID(source string, used map[string]struct{}) string {
	candidate := "pool_mig_" + sanitizePoolToken(source)
	if candidate == "pool_mig_" {
		candidate = "pool_mig_item"
	}
	poolID := candidate
	index := 1
	for {
		if _, ok := used[poolID]; !ok {
			used[poolID] = struct{}{}
			return poolID
		}
		index++
		poolID = fmt.Sprintf("%s_%d", candidate, index)
	}
}

func sanitizePoolToken(source string) string {
	var builder strings.Builder
	previousUnderline := false
	for _, charValue := range strings.ToLower(strings.TrimSpace(source)) {
		isLetter := charValue >= 'a' && charValue <= 'z'
		isNumber := charValue >= '0' && charValue <= '9'
		if isLetter || isNumber {
			builder.WriteRune(charValue)
			previousUnderline = false
			continue
		}
		if !previousUnderline {
			builder.WriteByte('_')
			previousUnderline = true
		}
	}
	return strings.Trim(builder.String(), "_")
}
