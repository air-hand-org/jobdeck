# IaC 図のローカル試行

TerraVision は `mise.toml` の `pipx:terravision` で 0.56.1 に固定しています。
Python 依存は mise の独立したツール環境に入り、アプリの依存には追加しません。

## 準備

- mise と uv（または pipx）、OpenTofu（既存の Aqua 設定）が必要です。
- OS 側に Graphviz と `dot` / `neato` のレイアウトプラグインが必要です。
  macOS は `brew install graphviz`、Ubuntu 26.04 は
  `sudo apt-get install graphviz libgvplugin-neato-layout8`。
- リポジトリの mise 設定を確認してから trust してください。既存の環境設定は
  `.env.local` と GitHub 認証を参照します。図のために新しい資格情報は不要です。

```sh
mise install pipx:terravision
mise exec -- terravision --version
mise exec -- terravision draw --help
```

通常の `make setup` にもこのツールのインストールが含まれます。

## 保存済み入力から描画

`make tofu-diagram` は **init / plan / apply を実行しません**。
同じ root・同じ構成から取得済みの plan JSON、graph DOT、ローカル HCL ディレクトリを
明示的に渡してください。入力が足りなければ描画前に失敗し、source-mode にフォールバックしません。
app と bootstrap は別々に実行します。

```sh
mise exec -- make tofu-diagram \
  TOFU_DIAGRAM_SOURCE=/absolute/private/app-source \
  TOFU_DIAGRAM_PLAN=/absolute/private/app-plan.json \
  TOFU_DIAGRAM_GRAPH=/absolute/private/app-graph.dot \
  TOFU_DIAGRAM_OUT=/absolute/private/app-architecture
```

出力先の親ディレクトリは先に作成してください。GCP の出力名は
`app-architecture-gcp.dot.png` です。TerraVision のバージョンによっては作業ファイルも
生成されるため、実行後に作業ツリーを確認し、生成物を commit しないでください。

plan JSON はバイナリ `tfplan` そのものではなく、保存済み plan を
`tofu show -json tfplan` で変換したものです。graph は既存の初期化済み root の
`tofu graph` 出力です。暗号化された plan の読み取りには既存の GCP KMS 認証が
必要な場合があります。この入口は plan を自動取得しません。既存の認証済み plan 手順で
作った入力がなければ、まずその手順・権限を確認してください。
バックエンドを local に置換して再 plan したり、apply したりしないでください。

## 取り扱いと制限

**ローカル出力は CI の安全化済み artifact と同じではありません。** 元の HCL・plan・graph は
秘密情報を含み得ます。入力、ログ、中間ファイル、PNG を private な場所に保管し、commit や
アップロードをしないでください。公開する場合は CI と同様に属性値・outputs・state・instance key を
除いた構造入力と空の resource 宣言を使い、画像のラベルとメタデータを検査してください。
既存 CI は従来どおり安全化・再エンコードした **PNG 単一ファイルのみ**を allowlist しています。
このローカル入口は CI やアップロード条件を変更しません。

矢印は Terraform の依存関係であり通信の実証ではありません。state 間接続も補完しません。
現在の安全化出力は横に非常に長くなるため、導入確定ではなくローカルで比較・試行するための入口です。
