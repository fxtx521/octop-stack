#!/bin/bash
# octop-client v0.3.0 打包: macOS .app + Windows zip
set -eu
cd /Users/liubang/dev/octop-stack/client
VER=0.3.0
DIST=~/dev/octop-stack/dist/v$VER
rm -rf "$DIST" && mkdir -p "$DIST"
LDF="-X main.version=$VER"

echo "== 1. 回归: 测试+vet =="
go vet ./...
go test -count=1 ./...

echo "== 2. macOS arm64 二进制 =="
GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "$LDF" -o "$DIST/octop-client-macos-arm64" .
"$DIST/octop-client-macos-arm64" --version

echo "== 3. macOS .app (原生无签名, 右键打开) =="
APP="$DIST/octop-client.app"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
cp "$DIST/octop-client-macos-arm64" "$APP/Contents/MacOS/octop-client"
cat > "$APP/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key>              <string>Octop Client</string>
    <key>CFBundleDisplayName</key>       <string>Octop Client</string>
    <key>CFBundleIdentifier</key>        <string>com.zhenyun.octop-client</string>
    <key>CFBundleVersion</key>           <string>0.3.0</string>
    <key>CFBundleShortVersionString</key><string>0.3.0</string>
    <key>CFBundleExecutable</key>        <string>octop-client</string>
    <key>CFBundlePackageType</key>       <string>APPL</string>
    <key>LSMinimumSystemVersion</key>    <string>12.0</string>
    <key>NSHighResolutionCapable</key>   <true/>
</dict>
</plist>
PLIST
ditto -c -k --sequesterRsrc --keepParent "$DIST/octop-client.app" "$DIST/octop-client-$VER-macos.zip"
echo ".app 打包完成"

echo "== 4. Windows amd64 =="
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$LDF" -o "$DIST/octop-client-$VER-windows-amd64.exe" .
file "$DIST/octop-client-$VER-windows-amd64.exe" | head -1

echo "== 5. 产物清单 =="
ls -la "$DIST"
echo "DIST=$DIST"
