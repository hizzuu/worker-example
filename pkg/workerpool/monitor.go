package workerpool

import (
	"fmt"
	"sync"
	"time"
)

// PoolStats はワーカープールの統計情報
type PoolStats struct {
	// 基本統計
	TotalTasks     int64 `json:"total_tasks"`
	CompletedTasks int64 `json:"completed_tasks"`
	FailedTasks    int64 `json:"failed_tasks"`
	ActiveTasks    int64 `json:"active_tasks"`
	QueuedTasks    int64 `json:"queued_tasks"`
	RetryingTasks  int64 `json:"retrying_tasks"`

	// ワーカー統計
	TotalWorkers  int `json:"total_workers"`
	ActiveWorkers int `json:"active_workers"`
	IdleWorkers   int `json:"idle_workers"`

	// 処理時間統計
	AverageTime float64 `json:"average_time_ms"`
	MinTime     float64 `json:"min_time_ms"`
	MaxTime     float64 `json:"max_time_ms"`

	// タスクタイプ別統計
	TaskTypeStats map[TaskType]TaskTypeStats `json:"task_type_stats"`

	// システム情報
	Uptime      time.Duration `json:"uptime_ms"`
	LastUpdated time.Time     `json:"last_updated"`
}

// TaskTypeStats はタスクタイプ別の統計
type TaskTypeStats struct {
	Total     int64   `json:"total"`
	Succeeded int64   `json:"succeeded"`
	Failed    int64   `json:"failed"`
	Retried   int64   `json:"retried"`
	AvgTime   float64 `json:"avg_time_ms"`
}

// Monitor はリアルタイム監視機能
type Monitor struct {
	pool      *WorkerPool
	stats     PoolStats
	mutex     sync.RWMutex
	startTime time.Time

	// リアルタイム更新用
	updateCh chan TaskResult
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewMonitor は新しいモニターを作成
func NewMonitor(pool *WorkerPool) *Monitor {
	return &Monitor{
		pool:      pool,
		startTime: time.Now(),
		updateCh:  make(chan TaskResult, 100),
		stopCh:    make(chan struct{}),
		stats: PoolStats{
			TaskTypeStats: make(map[TaskType]TaskTypeStats),
		},
	}
}

// Start はモニタリングを開始
func (m *Monitor) Start() {
	m.wg.Add(1)
	go m.updateLoop()
}

// Stop はモニタリングを停止
func (m *Monitor) Stop() {
	close(m.stopCh)
	m.wg.Wait()
}

// OnTaskResult はタスク結果を受信
func (m *Monitor) OnTaskResult(result TaskResult) {
	select {
	case m.updateCh <- result:
	default:
		// チャネルが満杯の場合はスキップ
	}
}

// updateLoop は統計情報を定期的に更新
func (m *Monitor) updateLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case result := <-m.updateCh:
			m.updateStats(result)

		case <-ticker.C:
			m.updateSystemStats()

		case <-m.stopCh:
			return
		}
	}
}

// updateStats はタスク結果で統計を更新
func (m *Monitor) updateStats(result TaskResult) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 基本統計を更新
	m.stats.TotalTasks++
	if result.Success {
		m.stats.CompletedTasks++
	} else {
		m.stats.FailedTasks++
	}

	// 処理時間統計を更新
	timeMs := float64(result.TotalDuration.Nanoseconds()) / 1e6
	if m.stats.TotalTasks == 1 {
		m.stats.MinTime = timeMs
		m.stats.MaxTime = timeMs
		m.stats.AverageTime = timeMs
	} else {
		if timeMs < m.stats.MinTime {
			m.stats.MinTime = timeMs
		}
		if timeMs > m.stats.MaxTime {
			m.stats.MaxTime = timeMs
		}
		// 移動平均を計算
		m.stats.AverageTime = (m.stats.AverageTime*float64(m.stats.TotalTasks-1) + timeMs) / float64(m.stats.TotalTasks)
	}

	// タスクタイプ別統計を更新
	typeStats := m.stats.TaskTypeStats[result.TaskType]
	typeStats.Total++
	if result.Success {
		typeStats.Succeeded++
	} else {
		typeStats.Failed++
	}
	if result.WasRetried() {
		typeStats.Retried++
	}

	// タスクタイプ別平均時間を更新
	if typeStats.Total == 1 {
		typeStats.AvgTime = timeMs
	} else {
		typeStats.AvgTime = (typeStats.AvgTime*float64(typeStats.Total-1) + timeMs) / float64(typeStats.Total)
	}

	m.stats.TaskTypeStats[result.TaskType] = typeStats
	m.stats.LastUpdated = time.Now()
}

// updateSystemStats はシステム統計を更新
func (m *Monitor) updateSystemStats() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.stats.Uptime = time.Since(m.startTime)
	m.stats.TotalWorkers = m.pool.workers

	// キューの長さを取得（近似値）
	m.stats.QueuedTasks = int64(len(m.pool.tasks))
	m.stats.RetryingTasks = int64(len(m.pool.retryQueue))

	// アクティブワーカー数は実装により異なる（ここでは推定）
	m.stats.ActiveWorkers = m.stats.TotalWorkers
	m.stats.IdleWorkers = 0
}

// GetStats は現在の統計情報を取得
func (m *Monitor) GetStats() PoolStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// ディープコピーを返す
	stats := m.stats
	stats.TaskTypeStats = make(map[TaskType]TaskTypeStats)
	for k, v := range m.stats.TaskTypeStats {
		stats.TaskTypeStats[k] = v
	}

	return stats
}

// PrintStats はコンソールに統計情報を表示
func (m *Monitor) PrintStats() {
	stats := m.GetStats()

	fmt.Println("\n📊 === リアルタイム統計情報 ===")
	fmt.Printf("稼働時間: %v\n", stats.Uptime.Round(time.Second))
	fmt.Printf("総タスク数: %d | 完了: %d | 失敗: %d\n",
		stats.TotalTasks, stats.CompletedTasks, stats.FailedTasks)
	fmt.Printf("キュー: %d | リトライ中: %d\n",
		stats.QueuedTasks, stats.RetryingTasks)
	fmt.Printf("ワーカー: %d/%d アクティブ\n",
		stats.ActiveWorkers, stats.TotalWorkers)
	fmt.Printf("処理時間: 平均 %.1fms | 最小 %.1fms | 最大 %.1fms\n",
		stats.AverageTime, stats.MinTime, stats.MaxTime)

	if len(stats.TaskTypeStats) > 0 {
		fmt.Println("\n📋 タスクタイプ別統計:")
		for taskType, typeStats := range stats.TaskTypeStats {
			successRate := float64(typeStats.Succeeded) / float64(typeStats.Total) * 100
			fmt.Printf("  [%s] 総数:%d 成功:%d 失敗:%d リトライ:%d 成功率:%.1f%% 平均:%.1fms\n",
				taskType, typeStats.Total, typeStats.Succeeded, typeStats.Failed,
				typeStats.Retried, successRate, typeStats.AvgTime)
		}
	}
	fmt.Println("==================================================")
}
