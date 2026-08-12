# How to deploy to cloud

TODO: あとでmarmaidにすること

hosts:
- https://jobdeck.app.air-hand.com/
- https://demo.jobdeck.app.air-hand.com/

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

`app` state と plan は `tofu-state` GCP KMS key を用いた OpenTofu encryption で暗号化する。
KMS key は `bootstrap` が管理し、削除保護と rotation を有効にする。`bootstrap` state は、その
apply が作る KMS key で暗号化できないため OpenTofu encryption の対象外とし、secret value を
含めない。GCS bucket の Google 所有鍵による at-rest encryption、uniform bucket-level access、
public access prevention、object versioning を基礎保護とする。

### Secret management

Secret Manager の secret resource と IAM は OpenTofu で管理する。secret value は
`google_secret_manager_secret_version.secret_data_wo` を使用して write-only で投入し、
state に保存しない。

secret source は `sops` GCP KMS key で暗号化した `iac/secrets/*.enc.yaml` とする。CI は
GitHub OIDC で deploy service account を impersonate し、一時ファイルへ復号した値を
OpenTofu の write-only input として渡す。一時ファイルは apply 後に削除する。OpenTofu
configuration 内の SOPS data source は使わない。復号済み data source が state に残ることを
避けるためである。

Secret Manager の secret container と runtime service account ごとの accessor IAM は
`bootstrap` が管理する。`app` deployer は各既存 secret への version 追加だけを許可し、secret
container の作成や runtime IAM policy の変更は行えない。これは secret 名と consumer の対応を
長期的な基盤契約として固定するためである。

### Image and deployment flow

Terraform Docker provider は使わない。CI が各 image を build し、Artifact Registry へ
release tag (`vX.Y.Z`) を push する。repository は immutable tag を有効にし、OpenTofu は
release tag を variable として受け取って Cloud Run revision を固定する。

```text
main merge
  -> release-please creates a draft release and vX.Y.Z tag
  -> GitHub Release published
  -> bootstrap tofu apply
  -> build / push vX.Y.Z images to Artifact Registry
  -> decrypt SOPS secrets
  -> tofu apply iac/app with immutable image tags
```

`_tofu` は `root` と `command` を受け取る reusable workflow であり、release publish 時、PR CI
の infrastructure plan、必要時の手動 apply から呼ぶ。backend prefix は `root` (`bootstrap` または
`app`) とし、認証 service account は選択した GitHub Environment の
`GCP_SERVICE_ACCOUNT` をそのまま使う。workflow は誤用防止のため、
`plan/bootstrap/plan`、`apply/bootstrap/bootstrap`、`plan/app/plan`、
`apply/app/production` の command/root/environment 組み合わせだけを許可する。
destroy workflow は `iac/app` だけに対して実行する。bootstrap resource と state は削除しない。

PR の infrastructure plan は、手動で作成する専用の `jobdeck-plan` service account を GitHub
Environment `plan` の OIDC binding で利用する。同 SA は project/IAM policy の read、state bucket
の read、state lock object の create/delete、state KMS key の decrypt のみを持つ。`_ci` の子 job と
して実行し、fork PR ではスキップする。

### Runtime topology

```text
Internet
  -> HTTPS Load Balancer
     -> ui: jobdeck.app.air-hand.com
     -> demo: demo.jobdeck.app.air-hand.com

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

Route 53 の `air-hand.com` zone から `jobdeck.app.air-hand.com` だけを Cloud DNS managed
zone へ委任する。bootstrap apply が Cloud DNS nameserver を出力し、一度だけ Route 53 に NS
record を設定する。以後、`jobdeck.app.air-hand.com` と
`demo.jobdeck.app.air-hand.com` の record は OpenTofu が Cloud DNS で管理する。

Cloud Run domain mapping は Preview のため使わない。global external HTTPS Load Balancer、
serverless NEG、Google-managed certificate を IaC 管理し、2 host をそれぞれ `ui` と `demo` に
route する。

### Service boundaries and future polyrepo migration

現在は monorepo を維持するが、`auth`、`job-api`、`job-worker`、`ui` は将来それぞれ独立した
repository・image・release cadence へ分割できる境界として実装する。特に `job-worker` は
`job-api` の内部 package や DB を参照せず、生成済み protobuf client による gRPC だけで通信する。
必須コンポーネントであっても、後方互換な API を維持する限り `job-api` と `job-worker` の
release を同時に行う必要はない。

protobuf schema の lint と breaking change detection は契約の構造的互換性を保証するが、認可、
validation、gRPC status code、retry、timeout、状態遷移などの振る舞いは保証しない。そのため、
polyrepo 移行後は以下の層で検証する。

| Layer | Owner | Scope |
| --- | --- | --- |
| protobuf lint / breaking check | contract (`proto`) repository | wire と generated client の互換性 |
| consumer unit test with fake/stub | consumer service repository | client が期待する request、response、error の扱い |
| provider contract verification | provider service repository | 実 gRPC server が versioned contract scenario を満たすこと |
| browser E2E / deployed smoke | UI repository | user journey、auth、networking、公開環境の配線 |

contract scenario は protobuf とともに version 管理する。consumer の fake/stub だけを正とせず、
provider の CI で実 server と一時 DB に対して同じ scenario を検証する。これにより高速な
service 単体 test を主としつつ、protobuf だけでは表せない API の意味論も確認する。browser
E2E は全 PR で全経路を網羅するのではなく、UI の変更・release 時の代表的な user journey と、
公開後 smoke test に絞る。

#### Local composition after a polyrepo split

polyrepo 化後も UI repository が browser E2E と AppHost を持つ。AppHost は source checkout を
必須とせず、通常は Artifact Registry の versioned development image を `AddContainer` で起動する。
UI developer は UI repository だけを clone して、互換性検証済みの service image 群と E2E を
実行できる。

特定 service を開発するときだけ、その service repository で同じ Dockerfile から local image を
build し、AppHost の image tag を override する。例えば `job-api:dev-local` を起動対象にすれば、
UI/browser E2E は local API とその他の development image を組み合わせて実行できる。全 service
source を常に checkout して `AddProject` する構成にはしない。

```text
default local integration
  ui source + browser E2E
  -> auth/job-api/job-worker/demo development images

job-api development override
  ui source + browser E2E
  -> job-api:dev-local + other development images

production smoke
  browser E2E -> published public URL
```

全 service の source を同時に変更する頻度が高くなった場合だけ、複数 checkout を compose する
`jobdeck-dev` repository を追加する。これは開発補助であり、CI の正規 contract verification や
browser E2E の唯一の実行基盤にはしない。候補として、AppHost から repository を clone して
`AddProjectFromRepository` などで起動できる
[Aspire.PolyRepo](https://github.com/Dutchskull/Aspire.PolyRepo) を評価する。
