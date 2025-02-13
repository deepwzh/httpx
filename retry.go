package httpx

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// RetryConfig 包含重试逻辑的配置
type RetryConfig struct {
	MaxRetries       int
	RetryInterval    time.Duration
	RetryStatusCodes []int
}

// doRequestWithRetry 执行 HTTP 请求并在失败时重试
func doRequestWithRetry(client *http.Client, req *http.Request, config *RetryConfig) (*http.Response, error) {
	var resp *http.Response
	var err error

	if config == nil {
		resp, err = client.Do(req)
		return resp, err
	}
	for i := 0; i < config.MaxRetries; i++ {
		resp, err = client.Do(req)
		if err == nil {
			// 检查响应状态码是否在重试列表中
			retry := false
			for _, code := range config.RetryStatusCodes {
				if resp.StatusCode == code {
					retry = true
					break
				}
			}
			if !retry {
				return resp, nil
			}
			slog.Error("request failed", "err", err, "code", resp.StatusCode, "retry", fmt.Sprintf("%v/%v", i, config.MaxRetries))
		} else {
			slog.Error("request failed", "err", err, "retry", fmt.Sprintf("%v/%v", i, config.MaxRetries))
		}

		// 关闭响应体以防止资源泄漏
		if resp != nil {
			resp.Body.Close()
		}

		time.Sleep(config.RetryInterval)
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", config.MaxRetries, err)
}
