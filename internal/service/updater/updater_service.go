package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go.uber.org/zap"

	"TagMatrix/internal/pkg/logger"
	"TagMatrix/internal/service/network"
)

// 请求检查更新的github链接
const githubLatestReleaseAPI = "https://api.github.com/repos/sun977/TagMatrix/releases/latest"

// UpdateInfo 承载发送给前端的更新数据
type UpdateInfo struct {
	HasUpdate    bool   `json:"has_update"`
	CurrentVer   string `json:"current_ver"`
	LatestVer    string `json:"latest_ver"`
	ReleaseNotes string `json:"release_notes"`
	ReleaseUrl   string `json:"release_url"`
}

// CheckForUpdates 异步检测 GitHub 的最新 Release
func CheckForUpdates(ctx context.Context, currentVersion string) {
	// 稍微延迟一下，不要阻塞主程序的启动和页面渲染
	go func() {
		time.Sleep(2 * time.Second)

		info, err := FetchLatestRelease(currentVersion)
		if err != nil {
			logger.Log.Warn("Failed to check for updates", zap.Error(err))
			return
		}

		if info.HasUpdate {
			logger.Log.Info("New version detected", zap.String("latest", info.LatestVer))
			// 通知前端弹窗
			runtime.EventsEmit(ctx, "update_available", info)
		} else {
			logger.Log.Info("App is up to date", zap.String("current", currentVersion))
		}
	}()
}

// FetchLatestRelease 发起 HTTP 请求到 GitHub API 获取最新版本
func FetchLatestRelease(currentVersion string) (*UpdateInfo, error) {
	proxySvc := network.NewProxyService()
	client := proxySvc.GetHTTPClient()
	client.Timeout = 10 * time.Second // 设置 10 秒超时，并且已接管系统的全局代理配置

	req, err := http.NewRequest("GET", githubLatestReleaseAPI, nil)
	if err != nil {
		return nil, err
	}
	// 最好带上 Accept 规范，不过直接查最新的也是可以的
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	// GitHub API 强烈建议携带 User-Agent，否则有时候会直接被 403 拦截
	req.Header.Set("User-Agent", "TagMatrix-Updater")

	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Warn("HTTP request to GitHub failed", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned non-200 status: %d", resp.StatusCode)
	}

	var releaseData struct {
		TagName string `json:"tag_name"`
		Body    string `json:"body"`
		HtmlUrl string `json:"html_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&releaseData); err != nil {
		return nil, err
	}

	return &UpdateInfo{
		HasUpdate:    IsNewerVersion(currentVersion, releaseData.TagName),
		CurrentVer:   currentVersion,
		LatestVer:    releaseData.TagName,
		ReleaseNotes: releaseData.Body,
		ReleaseUrl:   releaseData.HtmlUrl,
	}, nil
}

// IsNewerVersion 严谨的版本比对逻辑：按点号切割并逐段转为数字比较（如 4.1.3 vs 4.2.0）
func IsNewerVersion(current string, latest string) bool {
	// 移除可能存在的 "v" 前缀或者 "TagMatrix-" 前缀
	curr := strings.TrimPrefix(strings.ToLower(current), "v")
	curr = strings.TrimPrefix(curr, "tagmatrix-v")

	lat := strings.TrimPrefix(strings.ToLower(latest), "v")
	lat = strings.TrimPrefix(lat, "tagmatrix-v")

	if lat == "" {
		return false
	}

	currParts := strings.Split(curr, ".")
	latParts := strings.Split(lat, ".")

	maxLen := len(currParts)
	if len(latParts) > maxLen {
		maxLen = len(latParts)
	}

	for i := 0; i < maxLen; i++ {
		currVal := 0
		latVal := 0

		if i < len(currParts) {
			if v, err := strconv.Atoi(currParts[i]); err == nil {
				currVal = v
			}
		}

		if i < len(latParts) {
			if v, err := strconv.Atoi(latParts[i]); err == nil {
				latVal = v
			}
		}

		if latVal > currVal {
			return true // 发现线上某一段的数字比本地大，确实是新版本
		} else if latVal < currVal {
			return false // 发现线上的数字比本地还小（或者降级发版了），不提示更新
		}
		// 如果相等，则继续比较下一段
	}

	return false // 完全一致
}
