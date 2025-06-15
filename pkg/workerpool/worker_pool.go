package workerpool

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type WorkerPool struct {
	tasks         chan Task
	retryQueue    chan Task
	results       chan TaskResult
	workers       int
	wg            sync.WaitGroup
	retryWg       sync.WaitGroup
	processors    map[TaskType]TaskProcessor
	retryPolicies map[TaskType]RetryPolicy
	taskTimeout   time.Duration
	shutdownCh    chan struct{} // 🆕 シャットダウン用チャネル
}

func NewWorkerPool(workers int) *WorkerPool {
	return &WorkerPool{
		tasks:         make(chan Task, 10),
		retryQueue:    make(chan Task, 50), // リトライキューは大きめに
		results:       make(chan TaskResult, 10),
		workers:       workers,
		processors:    make(map[TaskType]TaskProcessor),
		retryPolicies: TaskTypeRetryPolicies(), // デフォルトポリシーを設定
		taskTimeout:   30 * time.Second,
		shutdownCh:    make(chan struct{}),
	}
}

func (wp *WorkerPool) RegisterProcessor(taskType TaskType, processor TaskProcessor) {
	wp.processors[taskType] = processor
}

func (wp *WorkerPool) SetTaskTimeout(timeout time.Duration) {
	wp.taskTimeout = timeout
}

func (wp *WorkerPool) SetRetryPolicy(taskType TaskType, policy RetryPolicy) {
	wp.retryPolicies[taskType] = policy
}

func (wp *WorkerPool) Start() {
	fmt.Printf("🚀 %d個のワーカーを開始します\n", wp.workers)

	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}

	wp.retryWg.Add(1)
	go wp.retryHandler()
}

func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	fmt.Printf("👷 ワーカー %d が開始されました\n", id)

	for task := range wp.tasks {
		wp.executeTask(task, id)
	}

	fmt.Printf("🛑 ワーカー %d が終了しました\n", id)
}

// リトライハンドラー
func (wp *WorkerPool) retryHandler() {
	defer wp.retryWg.Done()

	fmt.Println("🔄 リトライハンドラーが開始されました")

	for {
		select {
		case task := <-wp.retryQueue:
			policy, exists := wp.retryPolicies[task.Type]
			if !exists {
				policy = DefaultRetryPolicy()
			}

			// リトライ遅延を計算
			delay := policy.CalculateRetryDelay(task.AttemptCount)
			fmt.Printf("⏰ タスク %d を %v 後にリトライします (試行回数: %d/%d)\n",
				task.ID, delay, task.AttemptCount+1, policy.MaxRetries+1)

			// 遅延後にメインキューに戻す
			time.Sleep(delay)

			select {
			case wp.tasks <- task:
				fmt.Printf("🔄 タスク %d をリトライキューから戻しました\n", task.ID)
			case <-wp.shutdownCh:
				return
			}

		case <-wp.shutdownCh:
			fmt.Println("🛑 リトライハンドラーが終了しました")
			return
		}
	}
}

func (wp *WorkerPool) executeTask(task Task, workerID int) {
	startTime := time.Now()
	if task.FirstAttempt.IsZero() {
		task.FirstAttempt = startTime // 最初の試行日時を設定
	}

	attemptInfo := ""
	if task.AttemptCount > 0 {
		attemptInfo = fmt.Sprintf(" (リトライ %d回目)", task.AttemptCount)
	}

	fmt.Printf("⚡ ワーカー %d がタスク %d (%s:%s) を処理中...%s\n", workerID, task.ID, task.Type, task.Name, attemptInfo)

	// タスクを実行
	var err error
	processor, exists := wp.processors[task.Type]
	if !exists {
		err = fmt.Errorf("タスクタイプ %s のプロセッサが登録されていません", task.Type)
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), wp.taskTimeout)
		err = processor(ctx, task)
		cancel()
	}

	endTime := time.Now()
	duration := endTime.Sub(startTime)
	totalDuration := endTime.Sub(task.FirstAttempt)

	if err != nil {
		// リトライ判定
		policy, exists := wp.retryPolicies[task.Type]
		if !exists {
			policy = DefaultRetryPolicy()
		}

		if policy.ShouldRetry(err, task.AttemptCount) {
			fmt.Printf("🔄 ワーカー %d: タスク %d が失敗、リトライします (エラー: %v)\n",
				workerID, task.ID, err)

			// リトライ用にタスクを更新
			task.AttemptCount++
			task.LastError = err

			// リトライキューに送信
			select {
			case wp.retryQueue <- task:
			default:
				// リトライキューが満杯の場合は失敗として処理
				fmt.Printf("⚠️ リトライキューが満杯のため、タスク %d を失敗として処理します\n", task.ID)
				wp.sendResult(task, err, duration, totalDuration, workerID, false)
			}
			return
		} else {
			fmt.Printf("❌ ワーカー %d: タスク %d が最終的に失敗 (試行回数: %d, エラー: %v)\n",
				workerID, task.ID, task.AttemptCount+1, err)
		}
	} else {
		successInfo := ""
		if task.AttemptCount > 0 {
			successInfo = fmt.Sprintf(" (%d回目で成功)", task.AttemptCount+1)
		}
		fmt.Printf("✅ ワーカー %d がタスク %d を完了%s (処理時間: %v, 総時間: %v)\n",
			workerID, task.ID, successInfo, duration, totalDuration)
	}

	wp.sendResult(task, err, duration, totalDuration, workerID, true)
}

func (wp *WorkerPool) sendResult(task Task, err error, duration, totalDuration time.Duration, workerID int, isFinal bool) {
	result := TaskResult{
		TaskID:        task.ID,
		TaskName:      task.Name,
		TaskType:      task.Type,
		Success:       err == nil,
		Error:         err,
		Duration:      duration,
		TotalDuration: totalDuration, // 🆕 リトライ含む総処理時間
		WorkerID:      workerID,
		StartTime:     task.FirstAttempt,
		EndTime:       time.Now(),
		AttemptCount:  task.AttemptCount + 1, // 🆕 試行回数
		IsFinal:       isFinal,               // 🆕 最終結果かどうか
	}

	wp.results <- result
}

func (wp *WorkerPool) AddTask(task Task) {
	wp.tasks <- task
	fmt.Printf("📥 タスク %d (%s) がキューに追加されました\n", task.ID, task.Name)
}

// 🆕 結果を取得する関数
func (wp *WorkerPool) GetResult() TaskResult {
	return <-wp.results
}

// 🆕 指定した数の結果を取得する関数
func (wp *WorkerPool) GetResults(count int) []TaskResult {
	results := make([]TaskResult, 0, count)
	for i := 0; i < count; i++ {
		result := <-wp.results
		results = append(results, result)
	}
	return results
}

func (wp *WorkerPool) Stop() {
	fmt.Println("🔄 ワーカープールを停止中...")

	// シャットダウンシグナルを送信
	close(wp.shutdownCh)

	close(wp.tasks) // タスクチャネルを閉じる
	wp.wg.Wait()    // すべてのワーカーの完了を待つ

	close(wp.retryQueue) // リトライキューを閉じる
	wp.retryWg.Wait()    // リトライハンドラーの完了を待つ

	close(wp.results) // 結果チャネルも閉じる
	fmt.Println("✋ ワーカープールが停止しました")
}
