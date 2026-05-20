// ============================================================================
//  ___________              _____          __         .__        
//  \__    ___/____    ____ /     \ _____ _/  |________|__|__  ___
//    |    |  \__  \  / ___/  \ /  \\__  \\   __\_  __ \  \  \/  /
//    |    |   / __ \/ /_/  >  Y    \/ __ \|  |  |  | \/  |>    < 
//    |____|  (____  /\___  /____|__  (____  /__|  |__|  |__/__/\_ \
//                 \//_____/        \/     \/                     \/
// ============================================================================
// ⚡️ TagMatrix :: Context & State Counters
//
// 👤 SYSTEM_ARCHITECT : sun977 (SunHaobo)
// 🌐 GITHUB_REF       : https://github.com/sun977
// 📧 CONTACT_MAIL     : jiuwei977@foxmail.com
// 📅 INIT_YEAR        : 2026
//
// 📝 [DESC] 定义基于 context.Context 的状态注入与副作用记录容器，支持行级计数与全局并发计数。
//
// 💡 "A somewhat obsessive developer in cybersecurity & AI scenarios."
// ============================================================================

package matcher

import (
	"context"
	"sync"
	"sync/atomic"
)

// ctxKey 自定义 Context 键类型，避免不同包间的键名冲突
type ctxKey string

const (
	globalCounterKey ctxKey = "globalCounter"
	rowCounterKey    ctxKey = "rowCounter"
	currentTagKey    ctxKey = "currentTag"
)

// ============================================================================
// Global Counter (全局计数器)
// ============================================================================

// GlobalCounter 全局计数器，贯穿整个打标任务。
// 使用 sync.Map 结合 atomic.Int64 保障极高并发下的原子累加安全。
type GlobalCounter struct {
	data sync.Map // key: string (tagName), value: *int64
}

// NewGlobalCounter 实例化一个新的全局并发安全计数器
func NewGlobalCounter() *GlobalCounter {
	return &GlobalCounter{}
}

// Inc 对指定标签的计数安全累加 delta
func (gc *GlobalCounter) Inc(tagName string, delta int) {
	// 如果 key 不存在，则初始化一个指针并存入
	val, _ := gc.data.LoadOrStore(tagName, new(int64))
	
	// 原子累加，绝对避免数据竞争
	atomic.AddInt64(val.(*int64), int64(delta))
}

// GetAll 获取当前所有的计数结果
func (gc *GlobalCounter) GetAll() map[string]int {
	res := make(map[string]int)
	gc.data.Range(func(key, value interface{}) bool {
		tagName := key.(string)
		count := atomic.LoadInt64(value.(*int64))
		res[tagName] = int(count)
		return true // 继续遍历
	})
	return res
}

// ============================================================================
// Row Counter (行级计数器)
// ============================================================================

// RowCounter 行级计数器，生命周期仅限于处理单行数据，不跨协程，无需并发控制。
type RowCounter struct {
	data map[string]int
}

// NewRowCounter 实例化一个新的行级计数器
func NewRowCounter() *RowCounter {
	return &RowCounter{
		data: make(map[string]int),
	}
}

// Inc 对指定标签的计数累加 delta
func (rc *RowCounter) Inc(tagName string, delta int) {
	rc.data[tagName] += delta
}

// GetAll 获取行级的所有计数值
func (rc *RowCounter) GetAll() map[string]int {
	// 拷贝一份副本返回，防止外部意外污染容器
	res := make(map[string]int, len(rc.data))
	for k, v := range rc.data {
		res[k] = v
	}
	return res
}

// ============================================================================
// Context Inject & Extract Helper Functions (上下文辅助函数)
// ============================================================================

// WithGlobalCounter 将全局计数器注入到 Context 中
func WithGlobalCounter(ctx context.Context, gc *GlobalCounter) context.Context {
	return context.WithValue(ctx, globalCounterKey, gc)
}

// GetGlobalCounter 从 Context 中提取全局计数器
func GetGlobalCounter(ctx context.Context) *GlobalCounter {
	if val, ok := ctx.Value(globalCounterKey).(*GlobalCounter); ok {
		return val
	}
	return nil
}

// WithRowCounter 将行级计数器注入到 Context 中
func WithRowCounter(ctx context.Context, rc *RowCounter) context.Context {
	return context.WithValue(ctx, rowCounterKey, rc)
}

// GetRowCounter 从 Context 中提取行级计数器
func GetRowCounter(ctx context.Context) *RowCounter {
	if val, ok := ctx.Value(rowCounterKey).(*RowCounter); ok {
		return val
	}
	return nil
}

// WithCurrentTag 将当前正在处理的标签名称注入到 Context 中
func WithCurrentTag(ctx context.Context, tagName string) context.Context {
	return context.WithValue(ctx, currentTagKey, tagName)
}

// GetCurrentTag 从 Context 中提取当前处理的标签名称
func GetCurrentTag(ctx context.Context) string {
	if val, ok := ctx.Value(currentTagKey).(string); ok {
		return val
	}
	return ""
}
