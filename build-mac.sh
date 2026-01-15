#!/usr/bin/env bash
set -e

APP_NAME="HAMLAB Bridge"
VERSION="0.1.0"

APP_DIR="dist/${APP_NAME}.app"
DMG_DIR="dmg"
BIN_NAME="hamlab-bridge"

# クリーン
rm -rf dist dmg *.dmg build hamlab-bridge icon-work/icon.icns

iconutil -c icns icon-work/icon.iconset -o icon-work/icon.icns

GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o build/hamlab-bridge-amd64
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o build/hamlab-bridge-arm64

lipo -create \
  build/hamlab-bridge-amd64 \
  build/hamlab-bridge-arm64 \
  -output build/hamlab-bridge

swiftc main.swift AppDelegate.swift -o hamlab-menubar

# .app 構成
mkdir -p "${APP_DIR}/Contents/MacOS"
mkdir -p "${APP_DIR}/Contents/Resources"

# バイナリ配置
cp "build/${BIN_NAME}" "${APP_DIR}/Contents/MacOS/${BIN_NAME}"
chmod +x "${APP_DIR}/Contents/MacOS/${BIN_NAME}"

# メニューバーアプリ配置
cp hamlab-menubar "${APP_DIR}/Contents/MacOS/hamlab-menubar"
chmod +x "${APP_DIR}/Contents/MacOS/hamlab-menubar"

# アイコン配置
cp icon-work/icon.icns "${APP_DIR}/Contents/Resources/icon.icns"

cp icon-work/icon.iconset/icon_16x16.png \
   dist/HAMLAB\ Bridge.app/Contents/Resources/StatusIcon.png

cp icon-work/icon.iconset/icon_32x32.png \
   dist/HAMLAB\ Bridge.app/Contents/Resources/StatusIcon@2x.png

# ★ LaunchAgent plist を app Resources に入れる
cp jp.hamlab.bridge.plist "${APP_DIR}/Contents/Resources/jp.hamlab.bridge.plist"

# Info.plist 作成
cat > "${APP_DIR}/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
 "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key>
  <string>${APP_NAME}</string>

  <key>CFBundleDisplayName</key>
  <string>${APP_NAME}</string>

  <key>CFBundleIdentifier</key>
  <string>jp.hamlab.bridge</string>

  <key>CFBundleVersion</key>
  <string>${VERSION}</string>

  <key>CFBundleShortVersionString</key>
  <string>${VERSION}</string>

  <key>CFBundleExecutable</key>
  <string>hamlab-menubar</string>

  <key>CFBundleIconFile</key>
  <string>icon</string>

  <!-- Dock に出さない（常駐用途） -->
  <key>LSUIElement</key>
  <true/>
</dict>
</plist>
EOF

# dmg 用フォルダ作成
mkdir -p "${DMG_DIR}"
cp -R "${APP_DIR}" "${DMG_DIR}/"

# ★ Applications フォルダへのエイリアス（重要）
ln -s /Applications "${DMG_DIR}/Applications"

# dmg 生成
hdiutil create \
  -volname "${APP_NAME}" \
  -srcfolder "${DMG_DIR}" \
  -ov \
  -format UDZO \
  "HAMLAB-Bridge-${VERSION}.dmg"

echo "✔ HAMLAB-Bridge-${VERSION}.dmg created"
