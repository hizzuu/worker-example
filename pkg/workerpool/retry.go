package workerpool

import (
	"time"
)

type RetryPolicy struct {
	MaxRetries      int           // 最大リトライ回数
	InitialDelay    time.Duration // 初回リトライまでの遅延
	MaxDelay        time.Duration // 最大遅延時間
	BackoffFactor   float64       // バックオフ係数
	RetryableErrors []string      // リトライ対象のエラーパターン
}

func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		RetryableErrors: []string{
			"SMTP接続エラー",
			"データベース接続エラー",
			"context deadline exceeded", // タイムアウト
		},
	}
}

func TaskTypeRetryPolicies() map[TaskType]RetryPolicy {
	return map[TaskType]RetryPolicy{
		TaskTypeEmail: {
			MaxRetries:      5, // メールは重要なので多めにリトライ
			InitialDelay:    2 * time.Second,
			MaxDelay:        60 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"SMTP接続エラー"},
		},
		TaskTypeImage: {
			MaxRetries:      2, // 画像処理は重くないのでリトライ少なめ
			InitialDelay:    5 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   1.5,
			RetryableErrors: []string{}, // 形式エラーは基本的にリトライしない
		},
		TaskTypeDatabase: {
			MaxRetries:      4, // データベースは接続エラーが多いので多めに
			InitialDelay:    1 * time.Second,
			MaxDelay:        20 * time.Second,
			BackoffFactor:   2.5,
			RetryableErrors: []string{"データベース接続エラー", "context deadline exceeded"},
		},
		TaskTypeReport: {
			MaxRetries:      3,
			InitialDelay:    10 * time.Second, // レポートは重い処理なので待機時間長め
			MaxDelay:        120 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"データ不整合エラー"},
		},
	}
}

func (rp *RetryPolicy) CalculateRetryDelay(attemptCount int) time.Duration {
	if attemptCount <= 0 {
		return rp.InitialDelay
	}

	// 指数バックオフ計算
	delay := float64(rp.InitialDelay) * (rp.BackoffFactor * float64(attemptCount))
	delayDuration := time.Duration(delay)

	// 最大遅延時間を超えないように制限
	if delayDuration > rp.MaxDelay {
		return rp.MaxDelay
	}

	return delayDuration
}

// ShouldRetry はエラーがリトライ対象かどうかを判定
func (rp *RetryPolicy) ShouldRetry(err error, attemptCount int) bool {
	if err == nil {
		return false
	}

	if attemptCount >= rp.MaxRetries {
		return false
	}

	errorMsg := err.Error()
	for _, retryableError := range rp.RetryableErrors {
		if len(retryableError) > 0 && len(errorMsg) >= len(retryableError) {
			if errorMsg[:len(retryableError)] == retryableError {
				return true
			}
		}
	}

	return false
}
