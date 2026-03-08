# git-handoff

`git-handoff` は、AI に作業中のブランチを渡して別の作業を進め、あとでその続きと変更内容を手元に戻すための Git subcommand です。git の worktree を使って AI の作業を別 worktree で進めつつ、手元の `local` での確認や追加作業をやりやすくします。OpenAI Codex app の handoff 機能にインスパイアされています。

## 目的

AI に実装を任せているあいだも、人は手元で別のブランチの作業を続けたいことがあります。

ただし Git には、同じ branch を複数の checkout で同時に持てない制約があります。  
`git-handoff` はこの制約に沿って、作業中のブランチを worktree へ安全に渡し、あとで `local` へ戻せるようにします。

このツールで解きたいことは次です。

- git の worktree を使って AI の作業を別 worktree で進めながら、`local` での確認をやりやすくする
- AI は専用ディレクトリで実装を進める
- 人は `local` のメインディレクトリで別ブランチの作業を続ける
- 必要になったらブランチを AI 側へ渡し、あとで手元へ戻せる
- staged / unstaged changes と untracked files も handoff 対象に含める
- `.gitignore` 対象ファイルは handoff しない

## インストール方法

```console
$ go install github.com/hanenao/git-handoff@latest
```

```console
$ brew install hanenao/homebrew-tap/git-handoff
```

direnv を使う場合は、この repo で `direnv allow` すると `bin/` が `PATH` に追加され、`GOCACHE` も repo ローカルへ向きます。

## Shell Integration

`git ho switch` / `git ho go <branch>` で移動先ディレクトリへ自動で入りたい場合は、shell integration を読み込みます。

```console
$ eval "$(git ho --init zsh)"
$ eval "$(git ho --init bash)"
$ git-ho --init fish | source
```

`git()` wrapper は `git ho ...` だけを横取りし、成功時の stdout 最終行が実在ディレクトリならそこへ `cd` します。  
他のツールも `git()` を上書きしている場合は競合する可能性があります。

自動で移動したくない場合は、初期化時に `--nocd` を付けるか、実行時に `git config ho.nocd true` または `git ho --nocd switch` を使います。

## コマンド

主なコマンドは次のとおりです。

```console
$ git ho worktree create
$ git ho worktree list
$ git ho worktree remove <worktree-id>

$ git ho switch [<worktree-id>]
$ git ho go <branch>
```

### `git ho worktree create`

AI 作業用の worktree を新規作成します。

### `git ho worktree list`

`local` と各 worktree の状態を一覧表示します。  
どの branch がどこにあるかを一覧だけで把握できます。

### `git ho worktree remove <worktree-id>`

不要になった worktree を削除します。

### `git ho switch [<worktree-id>]`

- `local` で実行した場合:
  - 現在の branch を空いている worktree へ handoff する
  - handoff 後、`local` では `ho.basebranch` の checkout を試み、使えなければ detached のままにする
- worktree で実行した場合:
  - 現在の branch を `local` へ戻す

同じ branch を `local` と worktree に同時 checkout しないよう、移動元からは branch を外します。
`worktree-id` を省略した場合は、空いている worktree が自動で選ばれます。
shell integration を有効にしていれば、成功後にカレントディレクトリは移動先ディレクトリへ切り替わります。

### `git ho go <branch>`

指定 branch を現在 checkout しているディレクトリへ移動するための補助コマンドです。  
shell integration が無い場合でも、stdout 最終行に移動先 path を返します。

## 使う用語

- `local`
  - 手元で普段作業するメインディレクトリ
  - 人が確認や追加作業をする場所
- `worktree`
  - AI 用の worktree
  - AI が別ディレクトリで作業する場所
- `handoff`
  - branch と作業中の差分を `local` と worktree のあいだで移す操作

## 使い方の流れ

### 1. worktree を作る

```console
$ git ho worktree create
created worktree: a8k2m9
```

AI に渡す作業場所を先に作ります。

### 2. `local` で作業 branch を作る

```console
$ git switch -c feature/order-cache
```

人が `local` で対象 branch を作成または checkout します。

### 3. branch を worktree に handoff する

```console
$ git ho switch
/path/to/.ho/a8k2m9
```

この時点で、branch と handoff 対象の差分が空いている worktree 側へ移ります。
コマンド成功後は、その worktree ディレクトリにいる状態になります。

### 4. AI が worktree で作業する

AI は worktree 側で実装、テスト、生成物の確認を進めます。  
そのあいだ人は `local` で別作業を続けられます。

### 5. branch を `local` に戻す

```console
$ git ho switch
/path/to/repo
```

AI 作業が終わったら `local` へ handoff back して、人がレビューや追加修正を行います。
コマンド成功後は、`local` のメインディレクトリに戻ります。

## 設定

設定は `git config` の `ho.*` 名前空間を使います。

1. command line flag
2. local config
3. global config
4. builtin default

主な設定:

- `ho.basedir`
  - worktree の作成先ディレクトリ
  - default: `.ho`
- `ho.basebranch`
  - `local -> worktree` の handoff 後に `local` で checkout を試みる branch
  - branch が存在しない、または他の worktree で checkout 済みなら `local` は detached のまま
  - default: `main`
- `ho.copyignored`
  - worktree 作成時に `.gitignore` 対象ファイルをコピーするか
  - default: `false`
- `ho.hook`
  - worktree 作成時に実行する初期化コマンド
  - default: 未設定
- `ho.nocd`
  - shell integration 使用時に自動でディレクトリ移動しない
  - default: `false`

## 参考

- [OpenAI Codex: Worktrees and Handoff](https://developers.openai.com/codex/app/worktrees/)
