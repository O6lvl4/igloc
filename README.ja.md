# igloc

[English](README.md) | 日本語

`.gitignore` で無視されているファイルをスキャンして発見するCLIツール。

**igloc** は、バージョン管理にコミットされない秘密情報やローカル設定を見つけるのに役立ちます。プロジェクトの監査や新しいマシンのセットアップに便利です。

## インストール

```bash
go install github.com/O6lvl4/igloc/cmd/igloc@latest
```

またはソースからビルド：

```bash
git clone https://github.com/O6lvl4/igloc.git
cd igloc
make build
```

## 使い方

### シークレットをスキャン

```bash
# カレントディレクトリをスキャン
igloc scan

# 特定のディレクトリをスキャン
igloc scan ~/projects/my-app

# .env ファイルのみ表示
igloc scan --category env

# 全ての無視ファイルを表示（シークレット以外も）
igloc scan --all

# 依存ディレクトリも含める（node_modules など）
igloc scan --include-deps

# 再帰的に全リポジトリをスキャン
igloc scan -r ~/projects
```

### GitHub からパターンを同期

```bash
# github/gitignore から最新パターンを取得
igloc sync

# 対応言語一覧を表示
igloc sync --list
```

### シークレットのエクスポート/インポート

マシン間でシークレットを移行：

```bash
# シークレットを zip にエクスポート
igloc export backup.zip

# 特定ディレクトリからエクスポート
igloc export --path ~/projects/my-app backup.zip

# 再帰的に全リポジトリをエクスポート
igloc export -r ~/projects backup.zip

# 別のマシンでインポート
igloc import backup.zip

# インポート内容をプレビュー
igloc import --dry-run backup.zip

# 確認なしでインポート
igloc import --yes backup.zip
```

アーカイブ構造：
```
backup.zip
├── manifest.yaml      # メタデータとファイルマッピング
├── patterns.yaml      # 同期済みパターン設定
└── files/
    └── my-app/
        ├── .env
        └── config/.env.local
```

## 出力例

```
📂 /Users/you/projects/my-app

   🔑 ENV (3)
      .env 🔐
      .env.local 🔐
      config/.env.production 🔐

   Total: 3 files (🔐 3 secrets)
```

## 仕組み

1. **Scan**: `git status --ignored` で `.gitignore` に無視されているファイルを検出
2. **Categorize**: ファイルをタイプ別に分類（env, key, config, build, cache, ide）
3. **Filter**: デフォルトで依存ディレクトリを除外し、シークレットのみ表示

### 依存ディレクトリの除外

デフォルトで以下の依存ディレクトリを除外します：

| 言語 | ディレクトリ |
|------|-------------|
| Node.js | `node_modules/` |
| Python | `.venv/`, `venv/`, `__pycache__/`, `.tox/` |
| Ruby | `vendor/bundle/`, `.bundle/` |
| Go | `vendor/`, `pkg/mod/` |
| Rust | `target/` |
| Java | `.gradle/`, `.m2/`, `build/` |
| .NET | `packages/`, `bin/`, `obj/` |
| iOS | `Pods/`, `Carthage/` |
| その他 | ... |

`igloc sync` で [github/gitignore](https://github.com/github/gitignore) から最新パターンを取得できます。

## 設定

`igloc sync` 実行後、パターンは `~/.config/igloc/patterns.yaml` に保存されます。

## ユースケース

- **シークレット監査**: プロジェクト内の全 `.env` ファイルを発見
- **新規マシンセットアップ**: リポジトリをクローンした後に必要な秘密ファイルを一覧
- **セキュリティレビュー**: 誤ってコミットされる可能性のある認証情報を発見

## ライセンス

MIT
