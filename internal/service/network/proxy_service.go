package network

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"os"

	"TagMatrix/internal/config"
)

// ProxyService 负责处理网络代理配置
type ProxyService struct{}

// NewProxyService 创建网络代理服务
func NewProxyService() *ProxyService {
	return &ProxyService{}
}

// GetHTTPClient 根据系统配置返回一个配置好代理的 *http.Client
func (s *ProxyService) GetHTTPClient() *http.Client {
	cfg := config.GetConfig().Network

	// 基础传输配置
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment, // 默认行为，后续可能会被覆盖
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
	}

	switch cfg.ProxyMode {
	case "direct":
		transport.Proxy = nil // 禁用代理，直连
	case "custom":
		if cfg.ProxyURL != "" {
			if proxyURL, err := url.Parse(cfg.ProxyURL); err == nil {
				transport.Proxy = http.ProxyURL(proxyURL)
				// 针对自定义代理，很多时候需要跳过证书校验以避免报错
				transport.TLSClientConfig.InsecureSkipVerify = true
			}
		}
	case "system", "":
		// "system" 或者空值（兼容老配置文件）时，走系统代理 (默认 HTTP_PROXY/HTTPS_PROXY)
		transport.Proxy = http.ProxyFromEnvironment
	}

	return &http.Client{
		Transport: transport,
	}
}

// GetProxyEnvironment 可以在需要直接执行外部命令（而不是 Go 原生发请求）时提供代理环境变量
func (s *ProxyService) GetProxyEnvironment() []string {
	cfg := config.GetConfig().Network

	env := os.Environ()
	if cfg.ProxyMode == "custom" && cfg.ProxyURL != "" {
		env = append(env, "HTTP_PROXY="+cfg.ProxyURL)
		env = append(env, "HTTPS_PROXY="+cfg.ProxyURL)
		env = append(env, "ALL_PROXY="+cfg.ProxyURL)
	} else if cfg.ProxyMode == "direct" {
		// 覆盖系统默认环境变量为空，强制直连
		// TODO: 可能需要过滤掉 env 里的 HTTP_PROXY 等
	}
	return env
}
