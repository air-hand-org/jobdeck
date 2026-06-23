# JobDeck 設計メモ

## 目的

転職活動向けのポートフォリオとして、バックエンド寄りの強みを見せられるWebサービスを作る。

フロントエンドの作り込みよりも、以下を重視する。

* Goによるバックエンド実装
* protobuf / Connect RPC による contract-first なAPI設計
* Workerによる非同期ジョブ実行
* PostgreSQL(MySQL)を使ったDB-backed queue
* retry / timeout / cancellation / graceful shutdown
* E2E test
* feature flag
* Aspireによる複数サービスのローカル orchestration
* Blazerでのフロントエンド実装
* User/Auth service
* 将来的なクラウドデプロイ
* Dapr pub/sub

題材は **HTTP Job Runner / Job Execution Platform** とする。

---

## サービス概要

ユーザーがHTTPジョブを登録し、手動または将来的にはスケジュールで実行できるサービス。

ジョブ実行は同期的に行わず、実行要求をqueueに積み、Workerが非同期に処理する。

### 主な機能

MVPでは以下を実装する。

* HTTP jobの作成
* HTTP jobの一覧取得
* HTTP jobの詳細取得
* 手動実行要求
* 実行履歴の取得
* WorkerによるHTTP request実行
* 成功 / 失敗 / timeout の記録
* retry
* 複数Worker起動時の二重実行防止

---

## 採用技術方針

### 言語

Goを主軸にする。

理由:

* C#より最近触っているため、最初の実装速度が出やすい
* Worker処理、goroutine、context、graceful shutdownとの相性が良い
* Connect RPC / protobuf と相性が良い
* バックエンド寄りの強みを見せやすい

C#は必要になれば後段でAspire AppHost側、または簡易UIで使う。

---

## API方針

### Connect RPCを採用する

外部APIも内部APIも、まずは protobuf + Connect RPC で統一する。

純粋なREST APIではなく、RPC over HTTP/JSON として扱う。

目的は「RESTらしいURL設計」ではなく、以下を満たすこと。

* HTTPで呼べる
* JSONで呼べる
* ブラウザやcurlから叩ける
* TypeScript clientを生成できる
* Go Workerからも同じproto contractで呼べる
* OpenAPI RESTとgRPCの二重管理を避ける

### APIイメージ

外部からは以下のようなConnect RPC endpointを呼ぶ。

```text
POST /jobdeck.v1.JobService/CreateJob
POST /jobdeck.v1.JobService/GetJob
POST /jobdeck.v1.JobService/ListJobs
POST /jobdeck.v1.JobService/RequestExecution
POST /jobdeck.v1.ExecutionQueryService/ListExecutions
```

Worker内部向けには以下を使う。

```text
POST /jobdeck.v1.ExecutionQueueService/ClaimExecution
POST /jobdeck.v1.ExecutionQueueService/ReportExecutionResult
```

---

## サービス構成

MVPではサーバーとWorkerの2プロセス構成にする。

```text
jobdeck-server
  - Connect RPC server
  - JobService
  - ExecutionQueryService
  - ExecutionQueueService
  - PostgreSQL access
  - DB-backed queueの整合性管理

jobdeck-worker
  - Connect RPC client
  - ClaimExecution
  - HTTP job実行
  - ReportExecutionResult
  - goroutineによる並列処理
  - timeout / cancellation / graceful shutdown

PostgreSQL
  - jobs
  - executions
  - execution_events
```

WorkerはDBを直接触らず、`ExecutionQueueService` 経由でqueueをclaim/reportする。

これにより、DB schemaの所有権をserver側に寄せる。

---

## protobuf設計案

```proto
syntax = "proto3";

package jobdeck.v1;

option go_package = "github.com/ryoshimada746/jobdeck/gen/jobdeck/v1;jobdeckv1";

service JobService {
  rpc CreateJob(CreateJobRequest) returns (CreateJobResponse);
  rpc GetJob(GetJobRequest) returns (GetJobResponse);
  rpc ListJobs(ListJobsRequest) returns (ListJobsResponse);
  rpc RequestExecution(RequestExecutionRequest) returns (RequestExecutionResponse);
}

service ExecutionQueryService {
  rpc GetExecution(GetExecutionRequest) returns (GetExecutionResponse);
  rpc ListExecutions(ListExecutionsRequest) returns (ListExecutionsResponse);
}

service ExecutionQueueService {
  rpc ClaimExecution(ClaimExecutionRequest) returns (ClaimExecutionResponse);
  rpc ReportExecutionResult(ReportExecutionResultRequest) returns (ReportExecutionResultResponse);
}
```

### ExecutionStatus

```proto
enum ExecutionStatus {
  EXECUTION_STATUS_UNSPECIFIED = 0;
  EXECUTION_STATUS_QUEUED = 1;
  EXECUTION_STATUS_RUNNING = 2;
  EXECUTION_STATUS_SUCCEEDED = 3;
  EXECUTION_STATUS_FAILED = 4;
  EXECUTION_STATUS_CANCELLED = 5;
  EXECUTION_STATUS_TIMED_OUT = 6;
}
```

---

## ジョブ種別

最初は任意コマンド実行を避ける。

MVPでは **HTTP_REQUEST** のみ扱う。

任意shell command、任意Docker container、任意スクリプト実行はセキュリティ・サンドボックス設計が重くなるため対象外。

### HTTP_REQUEST設定

```text
HTTP_REQUEST
  - method
  - url
  - headers
  - body
  - timeout_ms
  - expected_status_codes
  - max_response_body_bytes
```

### 成功条件

最初は以下で判定する。

* timeoutしない
* HTTP requestが完了する
* status codeが expected_status_codes に含まれる

後段で以下を追加できる。

* response body contains
* JSONPathによる検証
* max duration
* redirect制限
* SSRF対策

---

## PostgreSQL / DB-backed queue

ジョブ実行要求はPostgreSQLに保存する。

専用のQueueミドルウェアを最初から導入せず、`executions` テーブルをqueueとして扱う。

### 主要テーブル案

```text
jobs
  id
  name
  type
  enabled
  config_json
  created_at
  updated_at

executions
  id
  job_id
  status
  attempt
  requested_at
  started_at
  finished_at
  next_run_at
  locked_by
  locked_until
  duration_ms
  error_message
  result_json

execution_events
  id
  execution_id
  event_type
  payload_json
  occurred_at
```

### Claim処理

複数Workerが同時に動いても同じexecutionを処理しないようにする。

PostgreSQLの `SELECT ... FOR UPDATE SKIP LOCKED` を使う想定。

概念的には以下。

```sql
SELECT id
FROM executions
WHERE status = 'queued'
  AND next_run_at <= now()
ORDER BY requested_at
FOR UPDATE SKIP LOCKED
LIMIT $1;
```

claimしたexecutionは同一トランザクション内で `RUNNING` に更新する。

```text
status = RUNNING
locked_by = worker_id
locked_until = now + lock_timeout
started_at = now
```

---

## Worker設計

Workerは常駐プロセスとして動く。

### 基本フロー

```text
1. ClaimExecution を呼ぶ
2. 実行対象がなければsleep
3. executionを受け取ったらgoroutineで処理
4. HTTP requestを実行
5. 成功 / 失敗 / timeoutを判定
6. ReportExecutionResult を呼ぶ
7. retry対象ならserver側で再queue
```

### Workerで重視すること

* context.Contextによるtimeout / cancellation
* goroutineによる並列処理
* worker concurrencyの設定
* graceful shutdown
* SIGTERM受信時に新規claimを止める
* 実行中jobを一定時間待つ、またはcontext cancel
* lock timeoutで再実行可能にする
* structured logging
* OpenTelemetry対応

---

## Retry設計

初期は固定間隔retryでもよい。

後段でfeature flagにより retry_v2 を追加する。

```text
retry_v1:
  fixed backoff

retry_v2:
  exponential backoff + jitter
```

retry判断はWorkerではなくserver側に寄せる。

Workerは実行結果を報告するだけにする。

---

## Event Sourcing / CQRSについて

理想的にはEvent Sourcing / CQRSまで発展できると良い。

ただしMVPでは過剰なので、最初は以下の折衷にする。

```text
executions:
  現在状態の正本

execution_events:
  状態遷移のappend-only履歴
```

MVPでは `executions` を正とし、`execution_events` は監査ログ・履歴として扱う。

将来的に、eventsからread modelを構築する方向に発展可能にする。

---

## Aspire方針

Aspireは、複数サービスのローカル orchestration と、将来的なクラウドデプロイの検証に使う。

Kubernetesを主戦場にはしない。

### Aspireで起動したいもの

```text
Aspire AppHost
  ├─ jobdeck-server
  ├─ jobdeck-worker
  ├─ notification-service
  ...
  ├─ MySQL(PostgreSQL)
  ├─ flagd later
  └─ Dapr components later
```

### Aspireの位置づけ

* Docker Composeの代替ではなく、アプリモデルとして使う
* ローカルで複数サービスをまとめて起動する
* 接続情報や依存関係を管理する
* Aspire dashboardで状態を確認する
* 将来的にクラウドデプロイへつなげる

Kubernetes運用を主目的にしないため、Tilt / Argo CD / Spinnaker はMVPでは使わない。

---

## Kubernetes / k3dについて

k3d / k3s にはデプロイできるとよいが、主戦場にはしない。

理由:

* Kubernetes前提にすると運用負荷が増える
* 今回の主目的はGoバックエンド、Worker、DB-backed queue、Aspire
* Tiltを覚えるくらいなら kubectl / docker / nerdctl / k3s/k3d の基礎に慣れたほうが汎用性が高い

k3dは後段のデプロイ検証用とする。

---

## クラウドデプロイ方針

最終的には独自ドメインで公開できるとよい。

クラウド候補は以下。

### Azure Container Apps

Aspireとの相性がよい。

構成イメージ:

```text
Azure Container Apps
  - jobdeck-server
  - jobdeck-worker
  - managed PostgreSQL
  - custom domain
  - managed TLS
```

Aspireのデプロイ体験を検証するなら第一候補。

### AWS

AWSでも可能。

ただし、Aspireの主戦場ではないため、AWSでは通常のECS構成に寄せるのが現実的。

構成イメージ:

```text
Route 53
  ↓
ACM certificate
  ↓
ALB
  ↓
ECS Fargate service: jobdeck-server
  ↓
RDS PostgreSQL

ECS Fargate service: jobdeck-worker
```

AWSでやるなら以下を使う。

* ECR
* ECS Fargate
* RDS PostgreSQL
* ALB
* ACM
* Route 53
* Secrets Manager or SSM Parameter Store
* CloudWatch Logs

Aspireはローカル orchestration、AWSデプロイはCDK/Terraform/ECSで堅く作る方針もあり。

---

## Feature Flag

後段で feature flag を入れる。

flagd + OpenFeature を使う想定。

### 候補flag

```text
retry_v2
  false: fixed backoff
  true: exponential backoff + jitter

execution_events
  false: executions table only
  true: execution_events にappend

worker_batch_claim
  false: 1件ずつclaim
  true: max_countでまとめてclaim

ssrf_protection
  false: basic validation
  true: private IP / localhost / redirect制限
```

### 方針

アプリコードはOpenFeature APIに依存し、flagd固有実装には直接依存しない。

```text
Go app
  ↓ OpenFeature API
flagd provider
  ↓
flagd
```

ローカルではAspireでflagdを起動する。

---

## Dapr

Daprは後段で検討する。

使うなら、pub/sub用途に限定する。

### Daprを使う場所

ジョブ実行の本線には使わない。

```text
ジョブ実行の正本:
  PostgreSQL executions / execution_events

ジョブ実行のclaim:
  Connect RPC ClaimExecution / ReportExecutionResult

周辺イベント配送:
  Dapr pub/sub
```

### 例

```text
JobService
  ↓ PostgreSQL
execution state / execution_events

JobService
  ↓ Dapr pub/sub
ExecutionSucceeded / ExecutionFailed

NotificationWorker
  ↓
webhook通知 / audit / log
```

Daprでやらないこと:

* DB-backed queueの置き換え
* Connect RPCの置き換え
* state storeでPostgreSQL設計を隠す
* service invocationを全面採用する

---

## E2E Test

E2E testは後段で必ず入れたい。

### 最初に検証するシナリオ

```text
1. HTTP jobを作成する
2. 実行要求を出す
3. Workerがclaimする
4. テスト用HTTP serverにrequestが届く
5. executionがSUCCEEDEDになる
6. execution_eventsに履歴が残る
```

### 追加シナリオ

```text
- timeout時にTIMED_OUTになる
- 5xx時にretryされる
- retry上限でFAILEDになる
- Workerを2つ起動しても二重実行されない
- Worker停止後、lock timeoutで再claimされる
```

### 実装候補

Goでは testcontainers-go を使う。

```text
E2E test
  ├─ PostgreSQL container
  ├─ jobdeck-server process/container
  ├─ jobdeck-worker process/container
  └─ fake target HTTP server
```

CIではtagを分ける。

```bash
go test ./...
go test -tags=e2e ./test/e2e/...
```

---

## セキュリティ考慮

HTTP jobであってもSSRF対策を考える。

MVPでは最低限、後段で強化する。

### 後段で入れたい対策

```text
- localhost宛て拒否
- private IP宛て拒否
- link-local宛て拒否
- redirect先の再検証
- timeout必須
- response body size limit
- 許可するHTTP methodの制限
- header denylist
```

任意コマンド実行は扱わない。

---

## ディレクトリ構成案

```text
jobdeck/
  apphost/
    JobDeck.AppHost/              # Aspire AppHost

  proto/
    jobdeck/v1/*.proto

  gen/
    jobdeck/v1/                   # generated code

  services/
    server/                       # Go Connect server
    worker/                       # Go worker

  internal/
    domain/
    store/
    queue/
    executor/
    flags/
    telemetry/

  db/
    migrations/

  deploy/
    docker/
    aws/
    azure/
    k8s/                          # optional / validation only

  test/
    e2e/

  docs/
    architecture.md
    deployment.md
```

---

## フェーズ分け

### Phase 1: Core

* Go server
* Connect RPC
* PostgreSQL
* HTTP_REQUEST job
* Go worker
* DB-backed queue
* 手動実行
* 実行履歴

### Phase 2: Reliability

* retry
* timeout
* lock timeout
* graceful shutdown
* execution_events

### Phase 3: Aspire

* AppHost
* PostgreSQL resource
* server / worker起動
* dashboard確認

### Phase 4: Test

* unit test
* integration test
* E2E test
* fake target HTTP server
* multiple worker no-duplicate test

### Phase 5: Deploy

* Dockerfile
* cloud deploy
* custom domain
* TLS
* READMEに構成図とデプロイ手順

### Phase 6: Could

* flagd + OpenFeature
* Dapr pub/sub
* 簡易UI
* k3d検証
* AWS/Azureの比較

---

## 今回は採用しないもの

### Kafka

MVPには過剰。

event log / stream processing / replay まで本格的にやるなら候補。

### Spinnaker

CD platformとしては強いが、今回のMVPでは重すぎる。

Aspireとはレイヤが違う。

### Argo CD

Kubernetes GitOpsを主戦場にするなら候補。

ただし、今回はk8s運用負荷を増やしたくないため後回し。

### Tilt

ローカルKubernetes開発ループを最適化する道具。

今回はk8s主戦場ではないため不要。

Tiltを覚えるより、kubectl / docker / nerdctl / k3s/k3d の基礎に寄せる。

---

## READMEで強調したいこと

* Go + Connect RPC による contract-first API
* ブラウザ/HTTP/JSONと内部RPCをprotoで統一
* WorkerはDBを直接触らず、Queue API越しにclaim/report
* PostgreSQL DB-backed queueで二重実行を防止
* goroutine/contextで並列実行、timeout、graceful shutdownを扱う
* Aspireで複数サービスと依存リソースをまとめて起動
* E2E testで分散構成を検証
* feature flagで段階的リリースを想定
* Dapr pub/subで周辺イベント配送を疎結合化可能
* 独自ドメインでクラウド公開まで目指す

---

## 最初に着手すること

1. repository作成
2. proto設計
3. Connect RPC serverの最小実装
4. PostgreSQL migration
5. `CreateJob`
6. `RequestExecution`
7. `ClaimExecution`
8. WorkerでHTTP request実行
9. `ReportExecutionResult`
10. 実行履歴取得
11. Aspire AppHostでserver / worker / postgresを起動
