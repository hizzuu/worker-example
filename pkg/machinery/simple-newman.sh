#!/bin/bash

# Newman実行して結果のみ抽出
RESULT=$(newman run api-collection.json -d input.csv --reporters cli --reporter-cli-no-banner 2>&1 | grep "FILTERED_USERS:" | sed "s/.*'FILTERED_USERS://" | sed "s/'.*//")

# CSVファイルに出力
echo "user_id" > completed_user_ids.csv
echo "$RESULT" | tr ',' '\n' >> completed_user_ids.csv
