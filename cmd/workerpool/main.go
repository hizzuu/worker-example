package main

import (
	"fmt"
	"time"

	"github.com/hizzuu/worker-example/pkg/workerpool"
)

func main() {
	// 3つのワーカーを持つプールを作成
	pool := workerpool.NewWorkerPool(3)

	// プロセッサを登録
	pool.RegisterProcessor(workerpool.TaskTypeEmail, workerpool.EmailProcessor)
	pool.RegisterProcessor(workerpool.TaskTypeImage, workerpool.ImageProcessor)
	pool.RegisterProcessor(workerpool.TaskTypeDatabase, workerpool.DatabaseProcessor)
	pool.RegisterProcessor(workerpool.TaskTypeReport, workerpool.ReportProcessor)

	// タスクタイムアウトを設定
	pool.SetTaskTimeout(10 * time.Second)

	// 🆕 監視機能を追加
	monitor := workerpool.NewMonitor(pool)
	monitor.Start()
	defer monitor.Stop()

	// 🆕 Web監視画面を開始
	monitor.StartWebServer(8080)

	// ワーカープールを開始
	pool.Start()

	// 大量のタスクを準備（監視機能のテスト用）
	fmt.Println("📝 大量タスクを投入してリアルタイム監視をテストします...")
	fmt.Println("🌐 Web監視画面: http://localhost:8080")

	// タスクを段階的に投入
	go func() {
		for batch := 1; batch <= 5; batch++ {
			fmt.Printf("\n📦 バッチ %d を投入中...\n", batch)

			for i := 1; i <= 4; i++ {
				taskID := (batch-1)*4 + i
				taskTypes := []workerpool.TaskType{
					workerpool.TaskTypeEmail,
					workerpool.TaskTypeImage,
					workerpool.TaskTypeDatabase,
					workerpool.TaskTypeReport,
				}

				task := workerpool.Task{
					ID:   taskID,
					Name: fmt.Sprintf("バッチ%d-タスク%d", batch, i),
					Type: taskTypes[(i-1)%len(taskTypes)],
				}

				pool.AddTask(task)
				time.Sleep(500 * time.Millisecond) // 0.5秒間隔で投入
			}

			time.Sleep(2 * time.Second) // バッチ間の待機
		}
	}()

	// 🆕 定期的に統計情報を表示
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				monitor.PrintStats()
			}
		}
	}()

	// 結果を取得（タスク完了を監視しながら）
	fmt.Println("\n📊 結果を取得中...")
	totalTasks := 20
	results := make([]workerpool.TaskResult, 0, totalTasks)

	for i := 0; i < totalTasks; i++ {
		result := pool.GetResult()
		results = append(results, result)

		// 🆕 監視機能にタスク結果を通知
		monitor.OnTaskResult(result)

		// 進捗表示
		fmt.Printf("📈 進捗: %d/%d 完了\n", len(results), totalTasks)
	}

	// 最終統計を表示
	fmt.Println("\n🎯 最終結果:")
	var (
		successCount  int
		failureCount  int
		retryCount    int
		totalDuration time.Duration
	)

	for _, result := range results {
		totalDuration += result.TotalDuration
		if result.Success {
			successCount++
			if result.WasRetried() {
				retryCount++
			}
		} else {
			failureCount++
		}
	}

	avgDuration := totalDuration / time.Duration(len(results))
	successRate := float64(successCount) / float64(len(results)) * 100

	fmt.Printf("📊 最終統計:\n")
	fmt.Printf("   総タスク数: %d\n", len(results))
	fmt.Printf("   成功: %d (%.1f%%)\n", successCount, successRate)
	fmt.Printf("   失敗: %d (%.1f%%)\n", failureCount, 100-successRate)
	fmt.Printf("   リトライ成功: %d (%.1f%%)\n", retryCount, float64(retryCount)/float64(len(results))*100)
	fmt.Printf("   平均処理時間: %v\n", avgDuration)

	// 🆕 最終監視統計を表示
	monitor.PrintStats()

	fmt.Println("\n🌐 Web監視画面は http://localhost:8080 で確認できます")
	fmt.Println("📊 JSONデータは http://localhost:8080/stats で取得できます")
	fmt.Println("⏰ 10秒後にシステムを停止します...")

	time.Sleep(10 * time.Second)

	// ワーカープールを停止
	pool.Stop()

	fmt.Println("🎉 すべての処理が完了しました！")
}
