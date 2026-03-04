export function ObservabilitySection() {
  return (
    <div className="grid grid-2">
      <div className="card">
        <div className="card-header">
          <h3 className="card-title">实时日志</h3>
          <span className="pill">WebSocket</span>
        </div>
        <div className="log-console">
          10:00:01 信息 HTTP 服务启动 :8080{'\n'}
          10:00:02 就绪 上游健康检查通过: 127.0.0.1:3000{'\n'}
          10:00:05 警告 IP 触发限流: 192.168.1.100{'\n'}
          10:00:12 错误 Redis 连接失败: connection refused
        </div>
      </div>
      <div className="card">
        <div className="card-header">
          <h3 className="card-title">日志分析</h3>
          <span className="pill">筛选器</span>
        </div>
        <div className="stack">
          <div className="form-row">
            <div className="form-field">
              <label className="label">时间范围</label>
              <select className="select">
                <option>最近 1 小时</option>
                <option>最近 24 小时</option>
                <option>最近 7 天</option>
              </select>
            </div>
            <div className="form-field">
              <label className="label">状态码</label>
              <select className="select">
                <option>全部</option>
                <option>2xx</option>
                <option>4xx</option>
                <option>5xx</option>
              </select>
            </div>
          </div>
          <div className="muted">分析数据按需接入后端查询接口</div>
        </div>
      </div>
    </div>
  )
}
