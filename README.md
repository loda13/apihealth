# API Health Checker

A high-performance Go-based CLI tool for testing LLM API connectivity, specifically designed for Anthropic's Claude API with support for custom proxy configurations.

## Features

- **Concurrent Testing**: Test multiple API endpoints simultaneously with configurable worker pools
- **Real Model Validation**: Performs actual API calls with minimal tokens to verify model availability
- **Proxy Support**: Configure custom base URLs for proxy/gateway configurations
- **Detailed Error Classification**: Distinguishes between auth failures, rate limits, timeouts, DNS issues, and more
- **Colorized Output**: Clean table display with color-coded status indicators
- **Comprehensive Logging**: Detailed JSON logs for troubleshooting
- **Flexible Configuration**: Support for YAML config files and environment variables

## Installation

### Build from Source

```bash
git clone https://github.com/lodatang/apihealth.git
cd apihealth
go build -o apihealth ./cmd/apihealth
```

### Run Directly

```bash
go run ./cmd/apihealth --config config.yaml
```

## Quick Start

1. **Create a configuration file**:

```bash
cp configs/config.example.yaml config.yaml
```

2. **Set your API key**:

```bash
export ANTHROPIC_API_KEY="sk-ant-your-key-here"
```

3. **Run the health check**:

```bash
./apihealth --config config.yaml
```

## Configuration

### Generate Default Config

Create a default `config.yaml` file:

```bash
./apihealth --init
```

To regenerate the config file (overwriting existing): 

```bash
./apihealth --init --force
```

### YAML Configuration

Create a `config.yaml` file:

```yaml
timeout: 30        # Request timeout in seconds
workers: 5         # Number of concurrent workers
log_file: "debug.log"

targets:
  - name: "Claude 3.5 Haiku"
    api_key: "${ANTHROPIC_API_KEY}"
    model: "claude-3-5-haiku-20241022"

  - name: "Claude 3.5 Sonnet (via Proxy)"
    base_url: "https://proxy.example.com"
    api_key: "${PROXY_API_KEY}"
    model: "claude-3-5-sonnet-20241022"
```

### Environment Variables

You can also use a `.env` file:

```bash
ANTHROPIC_API_KEY=sk-ant-your-key-here
PROXY_API_KEY=your-proxy-key-here
```

Or override configuration via environment variables:

```bash
export APIHEALTH_TIMEOUT=60
export APIHEALTH_WORKERS=10
export APIHEALTH_LOG_FILE=custom.log
```

## Usage

### Basic Usage

```bash
./apihealth --config config.yaml
```

### Command-Line Options

```bash
./apihealth [options]

Options:
  --config string     Path to configuration file (default "config.yaml")
  --workers int       Number of concurrent workers (0 = use config default)
  --timeout int       Request timeout in seconds (0 = use config default)
  --log-file string   Log file path (empty = use config default)
```

### Examples

**Test with custom timeout:**
```bash
./apihealth --config config.yaml --timeout 60
```

**Test with more workers:**
```bash
./apihealth --config config.yaml --workers 10
```

**Use custom log file:**
```bash
./apihealth --config config.yaml --log-file custom.log
```

## Supported Models

### Claude 4.5 Series (Latest)
- `claude-opus-4-5-20250514` - Most capable
- `claude-sonnet-4-5-20250514` - Balanced
- `claude-haiku-4-5-20251001` - Fastest, most cost-effective

### Claude 3.5 Series
- `claude-3-5-sonnet-20241022` - Latest 3.5 Sonnet
- `claude-3-5-haiku-20241022` - Fast and cost-effective

### Claude 3 Series (Legacy)
- `claude-3-opus-20240229`
- `claude-3-sonnet-20240229`
- `claude-3-haiku-20240307`

**Recommendation**: Use `claude-3-5-haiku-20241022` or `claude-haiku-4-5-20251001` for cost-effective connectivity testing.

## Output

### Success Example

```
Checking 3 target(s)...

Target                          Model                        Status  Response Time  Status Code  Error
Claude 3.5 Haiku               claude-3-5-haiku-20241022    ✓       245ms          200          -
Claude 3.5 Sonnet              claude-3-5-sonnet-20241022   ✓       312ms          200          -
Claude 4.5 Haiku               claude-haiku-4-5-20251001    ✓       198ms          200          -

All checks passed! ✓
```

### Failure Example

```
Checking 2 target(s)...

Target                          Model                        Status  Response Time  Status Code  Error
Claude 3.5 Haiku               claude-3-5-haiku-20241022    ✗       89ms           401          Authentication: Authentication failed
Claude 3.5 Sonnet (Proxy)      claude-3-5-sonnet-20241022   ⚠       1523ms         429          Rate Limit: Rate limit exceeded

Some checks failed. See debug.log for details.
```

## Error Types

The tool classifies errors into the following categories:

- **Authentication** (401/403): Invalid API key or insufficient permissions
- **Rate Limit** (429): Too many requests, retry after cooldown
- **Timeout**: Request exceeded timeout duration
- **DNS**: DNS resolution failed
- **Connection**: Connection refused or failed
- **Server Error** (500+): Anthropic service issues
- **Unknown**: Other errors

## Proxy Configuration

To test API endpoints through a proxy or gateway:

```yaml
targets:
  - name: "Claude via Proxy"
    base_url: "https://your-proxy.example.com"  # Custom base URL
    api_key: "${PROXY_API_KEY}"
    model: "claude-3-5-sonnet-20241022"
```

The tool will use `{base_url}/v1/messages` as the endpoint.

## Logging

Detailed logs are written to `debug.log` (or your configured log file) in JSON format:

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

## Troubleshooting

### 401 Unauthorized
- Verify your API key is correct
- Check that the API key has access to the specified model
- Ensure the API key is not expired

### 429 Rate Limit
- Wait before retrying
- Reduce the number of concurrent workers
- Check your API usage limits

### Timeout
- Increase the timeout value
- Check your network connection
- Verify the base URL is correct

### DNS Resolution Failed
- Check your internet connection
- Verify the base URL hostname is correct
- Try using a different DNS server

## Extension

The architecture is designed to be easily extensible. To add support for other providers:

1. Create a new checker in `internal/checker/` (e.g., `openai.go`)
2. Implement the same interface pattern
3. Update `checker.go` to support the new provider
4. Add configuration options for the new provider

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
