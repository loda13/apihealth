# API Health Checker

一个高性能的 Go 语言 CLI 工具，用于测试 LLM API 连接性，专为 Anthropic 的 Claude API 设计，支持自定义代理配置。

## 功能特性

- **并发测试**：使用可配置的工作池同时测试多个 API 端点
- **真实模型验证**：使用最少的 token 执行实际 API 调用以验证模型可用性
- **代理支持**：为代理/网关配置自定义基础 URL
- **详细错误分类**：区分认证失败、速率限制、超时、DNS 问题等
- **彩色输出**：带有颜色编码状态指示器的清晰表格显示
- **全面日志记录**：用于故障排除的详细 JSON 日志
- **灵活配置**：支持 YAML 配置文件和环境变量

## 安装

### 从源码构建

```bash
git clone https://github.com/lodatang/apihealth.git
cd apihealth
go build -o apihealth ./cmd/apihealth
```

### 直接运行

```bash
go run ./cmd/apihealth --config config.yaml
```

## 快速开始

1. **创建配置文件**：

```bash
cp configs/config.example.yaml config.yaml
```

2. **设置 API 密钥**：

```bash
export ANTHROPIC_API_KEY="sk-ant-your-key-here"
```

3. **运行健康检查**：

```bash
./apihealth --config config.yaml
```

## 配置

### YAML 配置

创建 `config.yaml` 文件：

```yaml
timeout: 30        # 请求超时时间（秒）
workers: 5         # 并发工作数
log_file: "debug.log"

targets:
  - name: "Claude 3.5 Haiku"
    api_key: "${ANTHROPIC_API_KEY}"
    model: "claude-3-5-haiku-20241022"

  - name: "Claude 3.5 Sonnet (通过代理)"
    base_url: "https://proxy.example.com"
    api_key: "${PROXY_API_KEY}"
    model: "claude-3-5-sonnet-20241022"
```

### 环境变量

你也可以使用 `.env` 文件：

```bash
ANTHROPIC_API_KEY=sk-ant-your-key-here
PROXY_API_KEY=your-proxy-key-here
```

或通过环境变量覆盖配置：

```bash
export APIHEALTH_TIMEOUT=60
export APIHEALTH_WORKERS=10
export APIHEALTH_LOG_FILE=custom.log
```

## 使用方法

### 基本用法

```bash
./apihealth --config config.yaml
```

### 命令行选项

```bash
./apihealth [选项]

选项：
  --config string     配置文件路径（默认 "config.yaml"）
  --workers int       并发工作数（0 = 使用配置默认值）
  --timeout int       请求超时时间（秒）（0 = 使用配置默认值）
  --log-file string   日志文件路径（空 = 使用配置默认值）
```

### 示例

**使用自定义超时测试：**
```bash
./apihealth --config config.yaml --timeout 60
```

**使用更多工作线程测试：**
```bash
./apihealth --config config.yaml --workers 10
```

**使用自定义日志文件：**
```bash
./apihealth --config config.yaml --log-file custom.log
```

## 支持的模型

### Claude 4.5 系列（最新）
- `claude-opus-4-5-20250514` - 最强大
- `claude-sonnet-4-5-20250514` - 平衡
- `claude-haiku-4-5-20251001` - 最快、最经济

### Claude 3.5 系列
- `claude-3-5-sonnet-20241022` - 最新 3.5 Sonnet
- `claude-3-5-haiku-20241022` - 快速且经济

### Claude 3 系列（旧版）
- `claude-3-opus-20240229`
- `claude-3-sonnet-20240229`
- `claude-3-haiku-20240307`

**推荐**：使用 `claude-3-5-haiku-20241022` 或 `claude-haiku-4-5-20251001` 进行经济高效的连接测试。

## 输出

### 成功示例

```
Checking 3 target(s)...

Target                          Model                        Status  Response Time  Status Code  Error
Claude 3.5 Haiku               claude-3-5-haiku-20241022    ✓       245ms          200          -
Claude 3.5 Sonnet              claude-3-5-sonnet-20241022   ✓       312ms          200          -
Claude 4.5 Haiku               claude-haiku-4-5-20251001    ✓       198ms          200          -

All checks passed! ✓
```

### 失败示例

```
Checking 2 target(s)...

Target                          Model                        Status  Response Time  Status Code  Error
Claude 3.5 Haiku               claude-3-5-haiku-20241022    ✗       89ms           401          Authentication: Authentication failed
Claude 3.5 Sonnet (Proxy)      claude-3-5-sonnet-20241022   ⚠       1523ms         429          Rate Limit: Rate limit exceeded

Some checks failed. See debug.log for details.
```

## 错误类型

工具将错误分类为以下类别：

- **Authentication**（401/403）：无效的 API 密钥或权限不足
- **Rate Limit**（429）：请求过多，冷却后重试
- **Timeout**：请求超过超时时间
- **DNS**：DNS 解析失败
- **Connection**：连接被拒绝或失败
- **Server Error**（500+）：Anthropic 服务问题
- **Unknown**：其他错误

## 代理配置

通过代理或网关测试 API 端点：

```yaml
targets:
  - name: "Claude via Proxy"
    base_url: "https://your-proxy.example.com"  # 自定义基础 URL
    api_key: "${PROXY_API_KEY}"
    model: "claude-3-5-sonnet-20241022"
```

工具将使用 `{base_url}/v1/messages` 作为端点。

## 日志记录

详细日志以 JSON 格式写入 `debug.log`（或你配置的日志文件）：

```json
{
  "level": "info",
  "target": "Claude 3.5 Haiku",
  "model": "claude-3-5-haiku-20241022",
  "success": true,
  "status_code": 200,
  "duration": 245,
  "time": "2026-03-31T17:00:00Z",
  "message": "Health check completed"
}
```

## 故障排除

### 401 未授权
- 验证你的 API 密钥是否正确
- 检查 API 密钥是否有权访问指定模型
- 确保 API 密钥未过期

### 429 速率限制
- 重试前等待
- 减少并发工作数
- 检查你的 API 使用限制

### 超时
- 增加超时值
- 检查你的网络连接
- 验证基础 URL 是否正确

### DNS 解析失败
- 检查你的互联网连接
- 验证基础 URL 主机名是否正确
- 尝试使用不同的 DNS 服务器

## 扩展

该架构设计易于扩展。要添加对其他提供商的支持：

1. 在 `internal/checker/` 中创建新的检查器（例如 `openai.go`）
2. 实现相同的接口模式
3. 更新 `checker.go` 以支持新提供商
4. 为新提供商添加配置选项

## 许可证

MIT

## 贡献

欢迎贡献！请随时提交 Pull Request。
