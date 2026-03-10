package gateway

import (
	"strings"
)

// blockPageTemplate 是 403 阻断页面的 HTML 模板
const blockPageTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>403 Forbidden - Access Denied | YoBFF</title>
    <style>
        :root {
            /* Next.js (Geist) Dark Theme Variables */
            --bg-main: #000000;
            --bg-surface: #111111;
            --border: #333333;
            --primary: #0070f3;
            --text-primary: #ffffff;
            --text-secondary: #888888;
            --error: #ff0000;
            --font-sans: 'Inter', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
        }

        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            font-family: var(--font-sans);
            background-color: var(--bg-main);
            color: var(--text-primary);
            height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            -webkit-font-smoothing: antialiased;
            -moz-osx-font-smoothing: grayscale;
            overflow: hidden;
        }

        .container {
            text-align: center;
            padding: 0 24px;
            max-width: 600px;
        }

        .status-code {
            font-size: 120px;
            font-weight: 800;
            letter-spacing: -0.05em;
            background: linear-gradient(180deg, #fff 0%, #666 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            line-height: 1;
            margin-bottom: 24px;
        }

        h1 {
            font-size: 24px;
            font-weight: 600;
            margin-bottom: 16px;
            letter-spacing: -0.02em;
        }

        p {
            color: var(--text-secondary);
            font-size: 16px;
            line-height: 1.6;
            margin-bottom: 32px;
        }

        .divider {
            height: 1px;
            background-color: var(--border);
            width: 100px;
            margin: 0 auto 32px;
        }

        .details-box {
            background-color: var(--bg-surface);
            border: 1px solid var(--border);
            border-radius: 8px;
            padding: 16px;
            text-align: left;
            margin-bottom: 32px;
            font-size: 13px;
        }

        .details-item {
            display: flex;
            justify-content: space-between;
            margin-bottom: 8px;
        }

        .details-item:last-child {
            margin-bottom: 0;
        }

        .label {
            color: var(--text-secondary);
        }

        .value {
            color: var(--text-primary);
        }

        .value.error-id {
            color: var(--error);
        }

        .actions {
            display: flex;
            gap: 16px;
            justify-content: center;
            margin-bottom: 32px;
        }

        .footer {
            color: var(--text-secondary);
            font-size: 13px;
            margin-top: 48px;
        }

        .footer a {
            color: var(--text-secondary);
            text-decoration: none;
            border-bottom: 1px solid transparent;
            transition: all 0.2s;
        }

        .footer a:hover {
            color: var(--text-primary);
            border-bottom-color: var(--text-primary);
        }

        .btn {
            height: 40px;
            padding: 0 20px;
            border-radius: 6px;
            font-size: 14px;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.2s;
            display: inline-flex;
            align-items: center;
            text-decoration: none;
        }

        .btn-primary {
            background-color: var(--text-primary);
            color: var(--bg-main);
            border: 1px solid var(--text-primary);
        }

        .btn-primary:hover {
            background-color: #ccc;
            border-color: #ccc;
        }

        .btn-secondary {
            background-color: transparent;
            color: var(--text-secondary);
            border: 1px solid var(--border);
        }

        .btn-secondary:hover {
            color: var(--text-primary);
            border-color: var(--text-primary);
        }

        /* 移动端适配 */
        @media (max-width: 600px) {
            .status-code {
                font-size: 80px;
            }
        }
    </style>
</head>
<body>

    <div class="container">
        <div class="status-code">403</div>
        <h1>Access Denied</h1>
        <p>
            您的请求被安全策略拦截。如果您认为这是一个误判，请联系系统管理员并提供下方的 Request ID。
        </p>
        
        <div class="divider"></div>

        <div class="details-box">
            <div class="details-item">
                <span class="label">Request ID:</span>
                <span class="value error-id">{{REQUEST_ID}}</span>
            </div>
            <div class="details-item">
                <span class="label">Client IP:</span>
                <span class="value">{{CLIENT_IP}}</span>
            </div>
            <div class="details-item">
                <span class="label">Time:</span>
                <span class="value" id="current-time"></span>
            </div>
        </div>

        <div class="actions">
            <a href="/" class="btn btn-primary">返回首页</a>
        </div>

        <div class="footer">
            你访问的源站受 <a href="https://github.com/WavesMan/YoBFF" target="_blank">YoBFF</a> 保护
        </div>
    </div>

    <script>
        // 设置当前时间
        const now = new Date();
        document.getElementById('current-time').textContent = now.toISOString();
    </script>
</body>
</html>`

// renderBlockPage 渲染包含上下文信息的 403 页面
func renderBlockPage(requestID, clientIP string) string {
	html := strings.ReplaceAll(blockPageTemplate, "{{REQUEST_ID}}", requestID)
	html = strings.ReplaceAll(html, "{{CLIENT_IP}}", clientIP)
	return html
}
