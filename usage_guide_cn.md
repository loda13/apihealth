# API 健康检查工具 - 中文使用指南

## 快速开始

### 1. 设置 API 密钥

```bash
export ANTHROPIC_API_KEY="sk-ant-你的密钥"
```

### 2. 运行检查

```bash
./apihealth --config config.yaml
```

## 配置说明

### 基本配置

```yaml
timeout: 30        # 请求超时时间（秒）
workers: 5         # 并发工作线程数
log_file: "debug.log"

targets:
  - name: "测试目标名称"
    api_key: "${ANTHROPIC_API_KEY}"
    model: "claude-3-5-haiku-20241022"
```

### 代理配置

如果需要通过代理访问 API：

```yaml
targets:
  - name: "通过代理的 Claude"
    base_url: "https://你的代理地址.com"
    api_key: "${PROXY_API_KEY}"
    model: "claude-3-5-sonnet-20241022"
```

## 命令行选项

```bash
./apihealth [选项]

选项：
  --config string     配置文件路径（默认 "config.yaml"）
  --workers int       并发工作线程数（0 = 使用配置文件默认值）
  --timeout int       请求超时时间（秒）（0 = 使用配置文件默认值）
  --log-file string   日志文件路径（空 = 使用配置文件默认值）
```

## 使用示例

### 基本使用

```bash
./apihealth --config config.yaml
```

### 自定义超时时间

```bash
./apihealth --config config.yaml --timeout 60
```

### 增加并发数

```bash
./apihealth --config config.yaml --workers 10
```

### 使用自定义日志文件

```bash
./apihealth --config config.yaml --log-file custom.log
```

## 支持的模型

### Claude 4.5 系列（最新）
- `claude-opus-4-5-20250514` - 最强大
- `claude-sonnet-4-5-20250514` - 平衡性能
- `claude-haiku-4-5-20251001` - 最快速、最经济

### Claude 3.5 系列
- `claude-3-5-sonnet-20241022` - 最新 3.5 Sonnet
- `claude-3-5-haiku-20241022` - 快速且经济

**推荐**：使用 `claude-3-5-haiku-20241022` 或 `claude-haiku-4-5-20251001` 进行连接测试，成本最低。

## 输出说明

### 成功示例

```
正在检查 3 个目标...

目标                          模型                        状态  响应时间  状态码  错误
Claude 3.5 Haiku             claude-3-5-haiku-20241022    ✓     245ms    200     -
Claude 3.5 Sonnet            claude-3-5-sonnet-20241022   ✓     312ms    200     -
Claude 4.5 Haiku             claude-haiku-4-5-20251001    ✓     198ms    200     -

所有检查通过！ ✓
```

### 失败示例

```
正在检查 2 个目标...

目标                          模型                        状态  响应时间  状态码  错误
Claude 3.5 Haiku             claude-3-5-haiku-20241022    ✗     89ms     401     认证：认证失败
Claude 3.5 Sonnet (代理)     claude-3-5-sonnet-20241022   ⚠     1523ms   429     速率限制：超过速率限制

部分检查失败。详情请查看 debug.log。
```

## 错误类型说明

工具会将错误分类为以下类型：

- **认证错误** (401/403)：API 密钥无效或权限不足
- **速率限制** (429)：请求过多，需要等待后重试
- **超时**：请求超过超时时间
- **DNS**：DNS 解析失败
- **连接**：连接被拒绝或失败
- **服务器错误** (500+)：Anthropic 服务问题
- **未知**：其他错误

## 常见问题排查

### 401 未授权
- 检查 API 密钥是否正确
- 确认 API 密钥有权限访问指定模型
- 确保 API 密钥未过期

### 429 速率限制
- 等待一段时间后重试
- 减少并发工作线程数
- 检查 API 使用限额

### 超时
- 增加超时时间设置
- 检查网络连接
- 验证 base_url 是否正确

### DNS 解析失败
- 检查互联网连接
- 验证 base_url 主机名是否正确
- 尝试使用不同的 DNS 服务器

## 日志文件

详细日志以 JSON 格式写入 `debug.log`（或您配置的日志文件）：

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

## 环境变量配置

您也可以使用 `.env` 文件：

```bash
# 复制示例文件
cp configs/.env.example .env

# 编辑 .env 文件
ANTHROPIC_API_KEY=sk-ant-你的密钥
PROXY_API_KEY=你的代理密钥

# 可选：覆盖配置
APIHEALTH_TIMEOUT=30
APIHEALTH_WORKERS=5
APIHEALTH_LOG_FILE=debug.log
```

## 技术特性

- ✓ **并发测试**：使用可配置的工作池同时测试多个端点
- ✓ **真实模型验证**：执行实际 API 调用（最小 token）验证模型可用性
- ✓ **代理支持**：支持自定义 base URL 用于代理配置
- ✓ **智能重试**：在 429/5xx 错误时重试，但不在认证错误时重试
- ✓ **错误分类**：区分认证失败、速率限制、超时、DNS 问题等
- ✓ **彩色输出**：带有颜色编码状态指示器的清晰表格显示
- ✓ **详细日志**：JSON 格式的详细日志用于故障排查
- ✓ **优雅关闭**：支持 Ctrl+C 优雅退出

## 扩展性

该架构设计易于扩展。要添加其他提供商（OpenAI、DeepSeek）的支持：

1. 在 `internal/checker/` 中创建新的检查器（如 `openai.go`）
2. 实现相同的接口模式
3. 更新 `checker.go` 以支持新提供商
4. 为新提供商添加配置选项
