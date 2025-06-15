package workerpool

import "time"

type TaskResult struct {
	TaskID        int
	TaskName      string
	TaskType      TaskType
	Success       bool
	Error         error
	Duration      time.Duration
	TotalDuration time.Duration // リトライ含む総処理時間
	WorkerID      int
	StartTime     time.Time
	EndTime       time.Time
	AttemptCount  int  // 試行回数
	IsFinal       bool // 最終結果かどうか
}

func (tr *TaskResult) IsTimeout() bool {
	if tr.Error == nil {
		return false
	}

	return tr.Error.Error() == "context deadline exceeded"
}

func (tr *TaskResult) GetErrorType() string {
	if tr.Error == nil {
		return ""
	}

	errorMsg := tr.Error.Error()
	switch {
	case tr.IsTimeout():
		return "TIMEOUT"
	case len(errorMsg) > 0:
		if len(errorMsg) > 20 {
			return errorMsg[:20] // エラーメッセージが長い場合は先頭20文字を返す
		}
		return errorMsg
	default:
		return "UNKNOWN"
	}
}

func (tr *TaskResult) WasRetried() bool {
	return tr.AttemptCount > 1
}
