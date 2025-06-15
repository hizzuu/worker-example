package workerpool

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// StartWebServer は統計情報をHTTPで公開
func (m *Monitor) StartWebServer(port int) {
	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		stats := m.GetStats()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(stats)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, getHTMLTemplate())
	})

	fmt.Printf("🌐 Web監視画面: http://localhost:%d\n", port)
	fmt.Printf("📊 JSON API: http://localhost:%d/stats\n", port)
	go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

// getHTMLTemplate はHTMLテンプレートを返す
func getHTMLTemplate() string {
	return `<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Worker Pool Monitor</title>
    <style>
        body { 
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; 
            margin: 20px; 
            background-color: #f5f5f5;
        }
        .header {
            background: linear-gradient(135deg, #007acc, #0099ff);
            color: white;
            padding: 20px;
            border-radius: 10px;
            margin-bottom: 20px;
            text-align: center;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
        }
        .stats { 
            display: grid; 
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); 
            gap: 20px; 
            margin-bottom: 30px;
        }
        .card { 
            border: 1px solid #ddd; 
            padding: 20px; 
            border-radius: 10px; 
            background: white;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            transition: transform 0.2s, box-shadow 0.2s;
        }
        .card:hover {
            transform: translateY(-2px);
            box-shadow: 0 4px 8px rgba(0,0,0,0.15);
        }
        .metric { 
            font-size: 28px; 
            font-weight: bold; 
            color: #007acc; 
            margin: 10px 0;
        }
        .label { 
            color: #666; 
            font-size: 14px; 
            text-transform: uppercase;
            font-weight: bold;
            letter-spacing: 0.5px;
        }
        .success { color: #28a745; }
        .failure { color: #dc3545; }
        .warning { color: #ffc107; }
        .info { color: #17a2b8; }
        .refresh { 
            margin: 10px 0; 
            text-align: center;
            background: white;
            padding: 15px;
            border-radius: 8px;
            border: 1px solid #ddd;
            box-shadow: 0 2px 4px rgba(0,0,0,0.05);
        }
        .refresh-flex {
            display: flex; 
            justify-content: space-between; 
            align-items: center;
        }
        .task-types {
            background: white;
            padding: 20px;
            border-radius: 10px;
            border: 1px solid #ddd;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .task-type-row {
            display: grid;
            grid-template-columns: 1fr 1fr 1fr 1fr 1fr 1fr;
            gap: 15px;
            padding: 12px 10px;
            border-bottom: 1px solid #eee;
            align-items: center;
        }
        .task-type-header {
            font-weight: bold;
            background: #f8f9fa;
            padding: 15px 10px;
            color: #495057;
        }
        .pulse {
            animation: pulse 1.5s ease-in-out;
        }
        @keyframes pulse {
            0% { opacity: 1; transform: scale(1); }
            50% { opacity: 0.8; transform: scale(1.02); }
            100% { opacity: 1; transform: scale(1); }
        }
        .status-indicator {
            display: inline-block;
            width: 12px;
            height: 12px;
            border-radius: 50%;
            margin-right: 8px;
        }
        .status-running { background-color: #28a745; }
        .status-warning { background-color: #ffc107; }
        .status-error { background-color: #dc3545; }
        
        .loading {
            text-align: center;
            color: #666;
            font-style: italic;
        }
        
        @media (max-width: 768px) {
            .stats {
                grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
                gap: 15px;
            }
            .task-type-row {
                grid-template-columns: 1fr 60px 60px 60px 70px 80px;
                gap: 8px;
                font-size: 14px;
            }
            .refresh-flex {
                flex-direction: column;
                gap: 10px;
            }
        }
    </style>
    <script>
        let lastUpdateTime = 0;
        
        function updateStats() {
            fetch('/stats')
                .then(response => response.json())
                .then(data => {
                    console.log('Stats received:', data); // デバッグ用
                    
                    // 基本統計の更新
                    updateElement('total-tasks', data.total_tasks || 0);
                    updateElement('completed-tasks', data.completed_tasks || 0);
                    updateElement('failed-tasks', data.failed_tasks || 0);
                    updateElement('queued-tasks', data.queued_tasks || 0);
                    updateElement('retrying-tasks', data.retrying_tasks || 0);
                    updateElement('active-workers', (data.active_workers || 0) + '/' + (data.total_workers || 0));
                    updateElement('avg-time', (data.average_time_ms || 0).toFixed(1) + 'ms');
                    updateElement('min-time', (data.min_time_ms || 0).toFixed(1) + 'ms');
                    updateElement('max-time', (data.max_time_ms || 0).toFixed(1) + 'ms');
                    updateElement('uptime', formatUptime(data.uptime_ms || 0));
                    
                    const successRate = data.total_tasks > 0 ? (data.completed_tasks / data.total_tasks * 100).toFixed(1) : 0;
                    updateElement('success-rate', successRate + '%');
                    
                    // 最終更新時刻の処理
                    const currentTime = new Date(data.last_updated).getTime();
                    if (currentTime > lastUpdateTime && data.last_updated) {
                        const updateTimeElement = document.getElementById('last-updated');
                        updateTimeElement.textContent = new Date(data.last_updated).toLocaleTimeString('ja-JP');
                        updateTimeElement.className = 'pulse';
                        updateTimeElement.style.color = '';
                        setTimeout(() => {
                            updateTimeElement.className = '';
                        }, 1500);
                        lastUpdateTime = currentTime;
                    }
                    
                    // タスクタイプ別統計の更新
                    updateTaskTypeStats(data.task_type_stats);
                    
                    // システム状態インジケーターの更新
                    updateSystemStatus(data);
                })
                .catch(error => {
                    console.error('Error fetching stats:', error);
                    const updateTimeElement = document.getElementById('last-updated');
                    updateTimeElement.textContent = 'エラー';
                    updateTimeElement.style.color = '#dc3545';
                });
        }
        
        function updateElement(id, value) {
            const element = document.getElementById(id);
            if (element && element.textContent !== String(value)) {
                element.textContent = value;
                element.classList.add('pulse');
                setTimeout(() => element.classList.remove('pulse'), 1500);
            }
        }
        
        function formatUptime(uptimeMs) {
            const seconds = Math.floor(uptimeMs / 1000000 / 1000);
            const hours = Math.floor(seconds / 3600);
            const minutes = Math.floor((seconds % 3600) / 60);
            const secs = seconds % 60;
            
            if (hours > 0) {
                return hours + 'h ' + minutes + 'm ' + secs + 's';
            } else if (minutes > 0) {
                return minutes + 'm ' + secs + 's';
            } else {
                return secs + 's';
            }
        }
        
        function updateTaskTypeStats(taskTypeStats) {
            const container = document.getElementById('task-types-container');
            if (!taskTypeStats || Object.keys(taskTypeStats).length === 0) {
                container.innerHTML = '<div class="loading">タスクタイプ別統計はまだありません</div>';
                return;
            }
            
            let html = '<div class="task-type-header task-type-row">';
            html += '<div>タスクタイプ</div>';
            html += '<div>総数</div>';
            html += '<div>成功</div>';
            html += '<div>失敗</div>';
            html += '<div>成功率</div>';
            html += '<div>平均時間</div>';
            html += '</div>';
            
            Object.keys(taskTypeStats).sort().forEach(taskType => {
                const stats = taskTypeStats[taskType];
                const successRate = stats.total > 0 ? (stats.succeeded / stats.total * 100).toFixed(1) : 0;
                const statusColor = successRate >= 90 ? 'success' : successRate >= 70 ? 'warning' : 'failure';
                
                html += '<div class="task-type-row">';
                html += '<div><strong>' + taskType + '</strong></div>';
                html += '<div>' + stats.total + '</div>';
                html += '<div class="success">' + stats.succeeded + '</div>';
                html += '<div class="failure">' + stats.failed + '</div>';
                html += '<div class="' + statusColor + '">' + successRate + '%</div>';
                html += '<div>' + stats.avg_time_ms.toFixed(1) + 'ms</div>';
                html += '</div>';
            });
            
            container.innerHTML = html;
        }
        
        function updateSystemStatus(data) {
            const statusElement = document.getElementById('system-status');
            let statusClass = 'status-running';
            let statusText = '正常稼働中';
            
            if (data.failed_tasks > 0 && data.total_tasks > 0) {
                const failureRate = (data.failed_tasks / data.total_tasks) * 100;
                if (failureRate > 20) {
                    statusClass = 'status-error';
                    statusText = '高エラー率';
                } else if (failureRate > 10) {
                    statusClass = 'status-warning';
                    statusText = '注意が必要';
                }
            }
            
            if (data.retrying_tasks > 5) {
                statusClass = 'status-warning';
                statusText = 'リトライ多数';
            }
            
            statusElement.innerHTML = '<span class="status-indicator ' + statusClass + '"></span>' + statusText;
        }
        
        // 1秒ごとに更新
        setInterval(updateStats, 1000);
        
        // 初回読み込み
        document.addEventListener('DOMContentLoaded', function() {
            updateStats();
        });
    </script>
</head>
<body>
    <div class="header">
        <h1>🚀 Worker Pool Monitor</h1>
        <div>リアルタイム監視ダッシュボード</div>
    </div>
    
    <div class="refresh">
        <div class="refresh-flex">
            <div>最終更新: <span id="last-updated">読み込み中...</span></div>
            <div>システム状態: <span id="system-status">起動中...</span></div>
        </div>
    </div>
    
    <div class="stats">
        <div class="card">
            <div class="label">総タスク数</div>
            <div class="metric info" id="total-tasks">0</div>
        </div>
        <div class="card">
            <div class="label">完了タスク</div>
            <div class="metric success" id="completed-tasks">0</div>
        </div>
        <div class="card">
            <div class="label">失敗タスク</div>
            <div class="metric failure" id="failed-tasks">0</div>
        </div>
        <div class="card">
            <div class="label">成功率</div>
            <div class="metric" id="success-rate">0%</div>
        </div>
        <div class="card">
            <div class="label">キューイング中</div>
            <div class="metric warning" id="queued-tasks">0</div>
        </div>
        <div class="card">
            <div class="label">リトライ中</div>
            <div class="metric warning" id="retrying-tasks">0</div>
        </div>
        <div class="card">
            <div class="label">ワーカー数</div>
            <div class="metric info" id="active-workers">0/0</div>
        </div>
        <div class="card">
            <div class="label">平均処理時間</div>
            <div class="metric" id="avg-time">0ms</div>
        </div>
        <div class="card">
            <div class="label">最小処理時間</div>
            <div class="metric" id="min-time">0ms</div>
        </div>
        <div class="card">
            <div class="label">最大処理時間</div>
            <div class="metric" id="max-time">0ms</div>
        </div>
        <div class="card">
            <div class="label">稼働時間</div>
            <div class="metric info" id="uptime">0s</div>
        </div>
    </div>
    
    <div class="task-types">
        <h3>📋 タスクタイプ別統計</h3>
        <div id="task-types-container" class="loading">
            データを読み込み中...
        </div>
    </div>
</body>
</html>`
}
