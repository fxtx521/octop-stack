#!/bin/bash
# octop-client v0.4.0 打包: macOS DMG + Windows exe
set -eu
cd /Users/liubang/dev/octop-stack/client
VER=0.4.1
DIST=~/dev/octop-stack/dist/v$VER
DMGROOT=/tmp/oc_dmgroot
rm -rf "$DIST" "$DMGROOT" && mkdir -p "$DIST" "$DMGROOT"
LDF="-X main.version=$VER"

echo "== 1. 回归: vet+测试 =="
go vet ./...
go test -count=1 ./...

echo "== 2. macOS arm64 + .app =="
GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "$LDF" -o "$DIST/octop-client-macos-arm64" .
APP="$DIST/octop-client.app"
mkdir -p "$APP/Contents/MacOS"
cp "$DIST/octop-client-macos-arm64" "$APP/Contents/MacOS/octop-client"
cat > "$APP/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key>              <string>Octop Client</string>
    <key>CFBundleDisplayName</key>       <string>Octop Client</string>
    <key>CFBundleIdentifier</key>        <string>com.zhenyun.octop-client</string>
    <key>CFBundleVersion</key>           <string>$VER</string>
    <key>CFBundleShortVersionString</key><string>$VER</string>
    <key>CFBundleExecutable</key>        <string>octop-client</string>
    <key>CFBundlePackageType</key>       <string>APPL</string>
    <key>LSMinimumSystemVersion</key>    <string>12.0</string>
    <key>NSHighResolutionCapable</key>   <true/>
</dict>
</plist>
PLIST
"$APP/Contents/MacOS/octop-client" --version

echo "== 3. DMG (拖拽安装布局) =="
cp -R "$APP" "$DMGROOT/"
ln -s /Applications "$DMGROOT/Applications"
hdiutil create -volname "Octop Client" -ov -format UDZO \
  -srcfolder "$DMGROOT" "$DIST/octop-client-$VER-macos.dmg" >/dev/null
echo "DMG 完成"

echo "== 4. Windows amd64 =="
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$LDF" -o "$DIST/octop-client-$VER-windows-amd64.exe" .
file "$DIST/octop-client-$VER-windows-amd64.exe" | head -1

rm -rf "$DMGROOT"
echo "== 5. 产物 =="
ls -la "$DIST"
echo "DIST=$DIST"
