# How to deploy to cloud

TODO: あとでmarmaidにすること

hosts:
- https://jobdeck.app.air-hand.com/
- https://demo-jobdeck.app.air-hand.com/

ui service (BFF) のみを外部公開する

GCP SecretManagerに保存し、それぞれ環境変数として注入する
- JWT生成鍵
- workerのclient_id, client_secret

コストがかかるためCloudSQLは使わず、job-api, auth Cloud Runのsidecarコンテナとして配置し、揮発することは許容する。

Cloud Runにはone-shot init containerがないため、migratorはapp containerのentrypointでserver起動前に実行する。

SSRFや脆弱性を回避するため、job-workerはforward proxyを通してリクエストを送信することとし、squidをsidecarとして構築し、allowlistで強制する

private/link-local IP、DNS rebindingを拒否する

proxy-bypassはsidecarでは完全に防ぐことはできないので注意

ui serviceはsessionをmemoryで管理しているため、スケールさせずに1 containerのみとする。(いずれにせよ、スケールさせるにはValkeyなどを外に出す必要がある)

リクエスト実行先として、簡単なDemo Appを用意し、様々なパターンに対応できるようにしておく。E2E testの検証先としても利用予定。

## Initial GCP deployment

初期の GCP project は `airhand-jobdeck`、region は `asia-northeast1` (Tokyo) とする。

### IaC layout and state

IaC はこの monorepo の `iac/` に置き、OpenTofu を使用する。Terragrunt は導入しない。

```text
iac/
  bootstrap/  # Artifact Registry, Secret Manager, IAM, GitHub OIDC, Cloud DNS, KMS
  app/        # Cloud Run, Scheduler, load balancer, DNS records
```

分割は依存関係と lifecycle を明示するためである。`app` は任意のタイミングで
destroy/apply できる一方、`bootstrap` は state、image repository、deploy identity、
secret の器を維持する。

OpenTofu state は手動で作成する GCS bucket を backend として利用する。backend は
`tofu init` より前に存在する必要があるため、この bucket 自体を同じ state で管理・
import しない。bucket の作成と初期操作者への権限付与だけを最小の手動 bootstrap とする。
bucket は object versioning と uniform bucket-level access を有効にする。state の prefix は
`bootstrap/` と `app/` に分ける。

`app` state と plan は GCP KMS を用いた OpenTofu encryption で暗号化する。KMS key は
`bootstrap` が管理し、削除保護と rotation を有効にする。

### Secret management

Secret Manager の secret resource と IAM は OpenTofu で管理する。secret value は
`google_secret_manager_secret_version.secret_data_wo` を使用して write-only で投入し、
state に保存しない。

secret source は SOPS + GCP KMS で暗号化した `iac/secrets/*.enc.yaml` とする。CI は
GitHub OIDC で deploy service account を impersonate し、一時ファイルへ復号した値を
OpenTofu の write-only input として渡す。一時ファイルは apply 後に削除する。OpenTofu
configuration 内の SOPS data source は使わない。復号済み data source が state に残ることを
避けるためである。

### Image and deployment flow

Terraform Docker provider は使わない。CI が各 image を build し、Artifact Registry へ
digest を push する。OpenTofu は digest を variable として受け取り、Cloud Run revision を
固定する。

```text
GitHub Actions
  -> build / push image digests to Artifact Registry
  -> decrypt SOPS secrets
  -> tofu apply iac/app with immutable image digests
```

destroy workflow は `iac/app` だけに対して実行する。bootstrap resource と state は削除しない。

### Runtime topology

```text
Internet
  -> HTTPS Load Balancer
     -> ui: jobdeck.app.air-hand.com
     -> demo: demo-jobdeck.app.air-hand.com

ui Cloud Run service
  -> job-api Cloud Run service + PostgreSQL sidecar
  -> auth Cloud Run service + PostgreSQL sidecar

Cloud Scheduler
  -> job-worker Cloud Run Job + Squid sidecar
     -> job-api / auth / allowlisted demo endpoint
```

`ui` と `demo` だけを外部公開する。`auth` と `job-api` は internal ingress とする。
PostgreSQL は各 service instance に閉じた揮発 DB であり、`max instances = 1` とする。
インスタンス再作成時にデータが失われることを明示的に許容する。

Cloud Run は one-shot init container を持たないため、migrator は sidecar にしない。PostgreSQL
sidecar の startup healthcheck 後に app container を開始し、entrypoint で `migrator up` を
実行してから server process を `exec` する。これにより listener を開く前に migration が完了する。

worker は初期段階では Cloud Scheduler から起動する Cloud Run Job とする。無限 polling の
代わりに `-once` で claim 済みの execution を処理・report して終了する。Pub/Sub 導入後は、
enqueue 時に publish し、Cloud Run service の push subscription で worker を message 数に
応じて scale させる。

### DNS

Route 53 の `air-hand.com` zone から `app.air-hand.com` を Cloud DNS managed zone へ
委任する。bootstrap apply が Cloud DNS nameserver を出力し、一度だけ Route 53 に NS record を
設定する。以後、`jobdeck.app.air-hand.com` と `demo-jobdeck.app.air-hand.com` の record は
OpenTofu が Cloud DNS で管理する。

Cloud Run domain mapping は Preview のため使わない。global external HTTPS Load Balancer、
serverless NEG、Google-managed certificate を IaC 管理し、2 host をそれぞれ `ui` と `demo` に
route する。
