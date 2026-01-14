# udp-bridge

udp-bridge は、WSJT-X / JTDX から送信される UDP ADIF データを受信し、  
WebSocket を通じてブラウザ側（HAMLAB 用拡張・スクリプト等）へ配信する  
**ローカル常駐型ブリッジアプリケーション**です。

[HAMLAB](https://hamlab.jp) の交信登録を自動化するための基盤として動作します。

---

## 概要

WSJT-X / JTDX  
↓（UDP ADIF）  
udp-bridge（Go / ローカル常駐）  
↓（WebSocket / JSON）  
ブラウザ側クライアント  
↓  
HAMLAB 管理画面（自動入力・登録）


## 主な機能

- WSJT-X / JTDX の UDP ADIF 受信
- WebSocket によるリアルタイム配信
- QRZ.com 連携
  - QTH 補完
  - Grid Locator 補完（6桁以上のみ使用）
  - Operator（NAME）補完
- Grid Locator から JCC/JCG 自動算出
- ポータブル局（/P 等）の判定
  - /P 付き CALL の場合は QTH / Grid を採用しない
- QRZ キャッシュ
  - 同一 CALL に対して API を再度呼び出さない
  - 再起動後も保持
- 設定用 Web UI 搭載


## 動作環境

- Windows / macOS / Linux
- WSJT-X または JTDX
- Go 1.22 以降（ビルド時）


## インストール
### バイナリ利用（Windows）

GitHub Releases から Windows 用バイナリ、またはインストーラをダウンロードしてください。

https://github.com/itcom/udp-bridge/releases

### ソースからビルド
```bash
git clone https://github.com/itcom/udp-bridge.git
cd udp-bridge
go build
```


## 起動方法
```bash
./udp-bridge
```

起動すると以下のサービスが立ち上がります。
- UDP 受信  
  127.0.0.1:2333
- WebSocket  
  ws://127.0.0.1:17800/ws
- 設定画面  
  http://127.0.0.1:17801/settings


## WSJT-X / JTDX 側の設定

- UDP Server  
  127.0.0.1
- UDP Port  
  2333
- ADIF UDP 出力を有効化してください


## QRZ.com 連携

設定画面から QRZ.com の以下を設定できます。
- ユーザー名
- パスワード

有効化すると以下が自動補完されます。
- Operator（NAME）
- QTH
- Grid Locator（6桁以上のみ）


## 出力データ形式

WebSocket では JSON 形式で配信されます。
- adif（生 ADIF 文字列）
- qrz
  - qth
  - grid
  - operator
- geo
  - jcc


## トラブルシュート
### QRZ がヒットしない
- CALL が正しく ADIF に含まれているか確認してください
- /P 等が付いた CALL の場合、QTH / Grid は意図的に無効化されます

### データが届かない
- udp-bridge が起動しているか確認してください
- WSJT-X / JTDX の UDP 設定を確認してください
- localhost 通信がファイアウォールで遮断されていないか確認してください


## ライセンス

MIT License


## 作者

itcom  
https://github.com/itcom