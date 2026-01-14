#!/usr/bin/env bash
set -e

APP_NAME="HAMLAB Bridge"
VERSION="0.1.0"

APP_DIR="dist/${APP_NAME}.app"
DMG_DIR="dmg"
BIN_NAME="hamlab-bridge"

# クリーン
rm -rf dist dmg *.dmg

# .app 構成
mkdir -p "${APP_DIR}/Contents/MacOS"
mkdir -p "${APP_DIR}/Contents/Resources"

# バイナリ配置
cp "${BIN_NAME}" "${APP_DIR}/Contents/MacOS/${BIN_NAME}"
chmod +x "${APP_DIR}/Contents/MacOS/${BIN_NAME}"

# アイコン配置
cp icon-work/icon.icns "${APP_DIR}/Contents/Resources/icon.icns"

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
  <string>${BIN_NAME}</string>

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
