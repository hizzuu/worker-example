{
  "info": {
    "name": "Subscription Cancel Check API",
    "description": "APIでキャンセル日をチェックしてCSVに出力",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "Check Subscription Cancel Date",
      "request": {
        "method": "POST",
        "header": [
          {
            "key": "Content-Type",
            "value": "application/json"
          },
          {
            "key": "Authorization",
            "value": "Bearer YOUR_API_TOKEN"
          }
        ],
        "body": {
          "mode": "raw",
          "raw": "{\n  \"member_id\": \"{{member_id}}\",\n  \"receipt_id\": \"{{receipt_id}}\",\n  \"amazon_user_id\": \"{{amazon_user_id}}\"\n}"
        },
        "url": {
          "raw": "https://api.subscription-service.com/v1/subscription/check",
          "protocol": "https",
          "host": [
            "api",
            "subscription-service",
            "com"
          ],
          "path": [
            "v1",
            "subscription",
            "check"
          ]
        }
      },
      "event": [
        {
          "listen": "test",
          "script": {
            "exec": [
              "// レスポンスをJSONとして取得",
              "const response = pm.response.json();",
              "",
              "// CSVの結果を蓄積するための配列を初期化",
              "if (!pm.globals.has(\"canceledUsers\")) {",
              "    pm.globals.set(\"canceledUsers\", JSON.stringify([]));",
              "}",
              "",
              "// cancel_dateが存在し、null/空でない場合のみ追加",
              "if (response.cancel_date && response.cancel_date !== null && response.cancel_date !== '') {",
              "    let canceledUsers = JSON.parse(pm.globals.get(\"canceledUsers\"));",
              "    const userId = pm.iterationData.get(\"user_id\");",
              "    ",
              "    // user_idを配列に追加（重複チェック）",
              "    if (!canceledUsers.includes(userId)) {",
              "        canceledUsers.push(userId);",
              "        pm.globals.set(\"canceledUsers\", JSON.stringify(canceledUsers));",
              "        console.log(`キャンセル済みユーザー発見: ${userId} (cancel_date: ${response.cancel_date})`);",
              "    }",
              "} else {",
              "    console.log(`アクティブユーザー: ${pm.iterationData.get(\"user_id\")}`);",
              "}",
              "",
              "// 最後のイテレーションで結果をCSV形式で出力",
              "if (pm.info.iteration === pm.info.iterationCount - 1) {",
              "    const canceledUsers = JSON.parse(pm.globals.get(\"canceledUsers\"));",
              "    console.log('\\n=== キャンセル済みユーザーのCSV ===');",
              "    console.log('user_id');",
              "    canceledUsers.forEach(userId => {",
              "        console.log(userId);",
              "    });",
              "    console.log(`\\n合計 ${canceledUsers.length} 件のキャンセル済みユーザーを検出`);",
              "    ",
              "    // グローバル変数をクリア",
              "    pm.globals.unset(\"canceledUsers\");",
              "}"
            ]
          }
        }
      ]
    }
  ]
}
newman run subscription-check-collection.json -d input.csv --reporters csv --reporter-csv-export results.csv