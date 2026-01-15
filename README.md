# HAMLAB Bridge

WSJT-X / JTDX から送信される UDP ADIF データを受信し、WebSocket を通じてブラウザへ配信するローカル常駐型ブリッジアプリケーションです。

[HAMLAB](https://hamlab.jp) の交信登録を自動化するための基盤として動作します。

## 概要

```
WSJT-X / JTDX
    ↓ UDP ADIF
HAMLAB Bridge（ローカル常駐）
    ↓ WebSocket / JSON
ブラウザ（HAMLAB）
    ↓
交信ログ自動登録
```

## 主な機能

- WSJT-X / JTDX の UDP ADIF 受信
- WebSocket によるリアルタイム配信
- QRZ.com 連携（QTH / Grid Locator / Operator 補完）
- Grid Locator から JCC/JCG 自動算出
- ポータブル局（/P 等）の判定
- QRZ キャッシュ（再起動後も保持）
- 設定用 Web UI
- メニューバー常駐（macOS）

## インストール

### macOS

1. [Releases](https://github.com/itcom/udp-bridge/releases) から `HAMLAB-Bridge-vX.X.X.dmg` をダウンロード
2. DMG を開き、`HAMLAB Bridge.app` を `Applications` フォルダへドラッグ
3. **初回起動前に**、ターミナルで以下を実行（署名がないため必要）
   ```bash
   xattr -cr /Applications/HAMLAB\ Bridge.app
   ```
4. `HAMLAB Bridge.app` を起動

> **Note**: macOS Ventura 以降、未署名アプリは Gatekeeper によりブロックされます。上記の `xattr` コマンドは quarantine 属性を削除し、アプリを開けるようにします。

### Windows

1. [Releases](https://github.com/itcom/udp-bridge/releases) から `hamlab-bridge-Setup-x64.exe` をダウンロード
2. インストーラを実行

### ソースからビルド

```bash
git clone https://github.com/itcom/udp-bridge.git
cd udp-bridge
go build
```

macOS 向け DMG 作成：
```bash
./build-mac.sh
```

## 起動後のサービス

| サービス | アドレス |
|---------|---------|
| UDP 受信 | 127.0.0.1:2333 |
| WebSocket | ws://127.0.0.1:17800/ws |
| 設定画面 | http://127.0.0.1:17801/settings |

## WSJT-X / JTDX の設定

1. Settings → Reporting タブを開く
2. 以下を設定
   - UDP Server: `127.0.0.1`
   - UDP Server port number: `2333`
3. 「Enable logged contact ADIF broadcast」にチェック

## QRZ.com 連携

設定画面（http://127.0.0.1:17801/settings）から QRZ.com のユーザー名・パスワードを設定すると、以下が自動補完されます。

- Operator（NAME）
- QTH
- Grid Locator（6桁以上のみ）

## 出力データ形式

WebSocket では以下の JSON を配信します。

```json
{
  "adif": "<call:6>JH9VIP<gridsquare:6>PM96AE...",
  "qrz": {
    "qth": "Fukui",
    "grid": "PM86CC",
    "operator": "Taro Yamada"
  },
  "geo": {
    "jcc": "2901"
  }
}
```

## トラブルシューティング

### アプリが開けない（macOS）

「破損している」「開けません」と表示される場合：
```bash
xattr -cr /Applications/HAMLAB\ Bridge.app
```

### QRZ がヒットしない

- CALL が正しく ADIF に含まれているか確認
- /P 等が付いた CALL の場合、QTH / Grid は意図的に無効化されます

### データが届かない

- HAMLAB Bridge が起動しているか確認
- WSJT-X / JTDX の UDP 設定を確認
- ファイアウォールで localhost 通信がブロックされていないか確認

## 動作環境

- macOS 12 以降（Intel / Apple Silicon 両対応）
- Windows 10 / 11
- WSJT-X または JTDX

## ライセンス

MIT License

## 作者

itcom  
https://github.com/itcom
