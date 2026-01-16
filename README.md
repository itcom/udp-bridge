# HAMLAB Bridge

WSJT-X / JTDX から送信される UDP ADIF データを受信し、WebSocket を通じてブラウザへ配信するローカル常駐型ブリッジアプリケーションです。

[HAMLAB](https://hamlab.jp) の交信登録を自動化するための基盤として動作します。

## 概要

```
WSJT-X / JTDX
    ↓ UDP ADIF
HAMLAB Bridge（ローカル常駐）← 無線機（CAT / CI-V）
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
- **無線機連携（CAT / CI-V）**
  - 周波数・モード取得
  - YAESU CAT / ICOM CI-V 自動判別
- **PTY ルーター**（macOS / Linux）
  - 無線機ポートを WSJT-X 等と共有
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

## 設定

設定画面（http://127.0.0.1:17801/settings）で各種設定を変更できます。

### 再起動が必要な設定

以下の設定を変更した場合は、アプリの再起動が必要です：

- 無線機連携の ON/OFF
- CAT / CI-V ポート
- ボーレート
- PTY ルーターの ON/OFF

### 即時反映される設定

以下の設定は保存後すぐに反映されます：

- QRZ.com ユーザー名 / パスワード
- QRZ.com 連携の ON/OFF
- JCC / 住所補完の ON/OFF

## WSJT-X / JTDX の設定

1. Settings → Reporting タブを開く
2. 以下を設定
   - UDP Server: `127.0.0.1`
   - UDP Server port number: `2333`
3. 「Enable logged contact ADIF broadcast」にチェック

## 無線機連携

設定画面から無線機の CAT / CI-V 接続を有効にすると、周波数とモードをリアルタイムで取得できます。

### 対応プロトコル

| メーカー | プロトコル | 確認済み機種 |
|---------|-----------|-------------|
| ICOM | CI-V | IC-705, IC-7300 等 |
| YAESU | CAT | FT-991A, FT-710 等 |
| KENWOOD | CAT | TS-590 等 |

### PTY ルーター（macOS / Linux）

通常、シリアルポートは1つのアプリケーションしか開けませんが、PTY ルーターを有効にすると仮想ポートが作成され、WSJT-X 等と同時に使用できます。

1. 設定画面で「PTYルーター」にチェック
2. 表示される PTY パス（例: `/dev/ttys003`）を WSJT-X の CAT 設定に入力
3. HAMLAB Bridge と WSJT-X で同時に周波数・モードを取得可能

> **Note**: PTY ルーターは macOS / Linux でのみ利用可能です。

## QRZ.com 連携

設定画面（http://127.0.0.1:17801/settings）から QRZ.com のユーザー名・パスワードを設定すると、以下が自動補完されます。

- Operator（NAME）
- QTH
- Grid Locator（6桁以上のみ）

> **Note**: QRZ.com の API を利用するには「**XML Logbook Data Subscription**」以上のプランが必要です。無料プランでは利用できません。

## 出力データ形式

WebSocket では以下の JSON を配信します。

### ADIF 受信時

```json
{
  "type": "adif",
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

### 無線機状態

```json
{
  "type": "rig",
  "rig": "ICOM",
  "freq": 14074000,
  "mode": "USB",
  "data": true
}
```

### PTY パス通知

```json
{
  "type": "pty",
  "path": "/dev/ttys003"
}
```

## トラブルシューティング

### アプリが開けない（macOS）

「破損している」「開けません」と表示される場合：
```bash
xattr -cr /Applications/HAMLAB\ Bridge.app
```

### 無線機が認識されない

- 正しいポートとボーレートを選択しているか確認
- 無線機の CAT / CI-V 設定が有効か確認
- USB ドライバがインストールされているか確認

### CI-V が止まる / 更新されない

- 他のアプリが同じポートを使用していないか確認
- PTY ルーターを有効にすると複数アプリで共有可能

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
- Linux（GTK3 対応環境）
- WSJT-X または JTDX

## ライセンス

MIT License

## 作者

itcom  
https://github.com/itcom
