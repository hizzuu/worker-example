package workerpool

import (
	"context"
	"errors"
	"math/rand"
	"time"
)

type Task struct {
	ID           int
	Name         string
	Type         TaskType
	Payload      interface{}
	AttemptCount int       // リトライ回数
	MaxRetries   int       // 最大リトライ回数
	LastError    error     // 最後のエラー
	CreatedAt    time.Time // タスクの作成日時
	FirstAttempt time.Time // 最初の試行日時
}

type TaskType string

const (
	TaskTypeEmail    TaskType = "email"
	TaskTypeImage    TaskType = "image"
	TaskTypeDatabase TaskType = "database"
	TaskTypeReport   TaskType = "report"
)

type TaskProcessor func(ctx context.Context, task Task) error

func EmailProcessor(ctx context.Context, task Task) error {
	processingTime := time.Duration(1+rand.Intn(2)) * time.Second

	select {
	case <-time.After(processingTime):
		// 最初の試行では20%失敗、リトライでは10%失敗（改善される想定）
		failureRate := 20
		if task.AttemptCount > 0 {
			failureRate = 10
		}

		if rand.Intn(100) < failureRate {
			return errors.New("SMTP接続エラー: メール送信に失敗しました")
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func ImageProcessor(ctx context.Context, task Task) error {
	processingTime := time.Duration(2+rand.Intn(4)) * time.Second

	select {
	case <-time.After(processingTime):
		// 画像形式エラーはリトライしても改善されないことが多い
		if rand.Intn(10) < 2 {
			return errors.New("画像形式エラー: サポートされていない形式です")
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func DatabaseProcessor(ctx context.Context, task Task) error {
	processingTime := time.Duration(1+rand.Intn(3)) * time.Second

	select {
	case <-time.After(processingTime):
		// データベース接続は時間が経つと改善されることが多い
		failureRate := 10
		if task.AttemptCount > 1 {
			failureRate = 3 // リトライで大幅改善
		}

		if rand.Intn(100) < failureRate {
			return errors.New("データベース接続エラー: タイムアウトしました")
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func ReportProcessor(ctx context.Context, task Task) error {
	processingTime := time.Duration(3+rand.Intn(3)) * time.Second

	select {
	case <-time.After(processingTime):
		// データ不整合は時間が経つと解決される場合がある
		failureRate := 15
		if task.AttemptCount > 0 {
			failureRate = 8
		}

		if rand.Intn(100) < failureRate {
			return errors.New("データ不整合エラー: レポート生成に必要なデータが不足しています")
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
