# detour2-fyne

`detour2-fyne` 是 detour2 的本地客户端。当前版本不再部署或删除阿里云函数，只在本机启动 socks5/http 代理，然后连接你指定的上游 `server` 或 `relay`。

## Usage

首次启动不会内置任何上游线路，也不会预填密码；请在“配置”页新建自己的上游配置。

默认运行参数：

- socks5 本地端口：`127.0.0.1:7776`
- HTTP 本地端口：`127.0.0.1:7777`
- WebSocket 连接池：`64`

上游地址可以填 `tcp://host:port`、`ws://host:port/ws`、`http://host:port` 或 `https://host:port`。客户端会把 `tcp://host:port` 自动转换为 detour2 local 需要的 `ws://host:port/ws`。请在配置页填写自己的上游地址和密码。

上游需要先运行 detour2 `server` 或 `relay`，例如：

```bash
detour2 server -l tcp://0.0.0.0:7777 -p <your-password>
```

或连接到一条 relay 链的入口：

```bash
detour2 relay -l tcp://0.0.0.0:7777 -r ws://next-relay:7777/ws -p <your-password>
```

客户端现在直接依赖 `github.com/observerss/detour2 v0.4.1`。

## Build

先安装 Fyne CLI：

```bash
go install fyne.io/fyne/v2/cmd/fyne@latest
```

桌面当前平台：

```bash
./build.sh
```

五个平台一起打包：

```bash
./build.sh all
```

在 Linux/WSL 上需要跨平台打包时，可以使用 Docker + `fyne-cross` 的封装脚本。它会创建临时 workspace、vendor 当前模块依赖，并把产物复制到 `dist/`：

```bash
scripts/package-cross.sh windows
scripts/package-cross.sh linux
scripts/package-cross.sh android
```

macOS 交叉包需要提供 `MACOSX_SDK_PATH`，iOS 仍需要 darwin host。

Linux：

```bash
./build.sh linux
```

Windows：

```bash
./build.sh windows
```

macOS：

```bash
./build.sh macos
```

macOS 运行页的“启用全局代理”现在会通过系统 `networksetup` 为所有启用的网络服务打开 HTTP、HTTPS 和 SOCKS 代理；如果当前权限不足，系统会弹出管理员授权窗口。启用前会保存一次系统代理快照，关闭时优先恢复原来的 Web、Secure Web、SOCKS 和 bypass domains 设置；没有快照时才回退为关闭这些代理状态。

macOS Developer ID 签名和公证流程见 [docs/apple-clients.md](docs/apple-clients.md)，脚本入口是 [scripts/Package-macOS.sh](scripts/Package-macOS.sh)。云桌面/Apple 构建环境准备见 [docs/apple-build-environment.md](docs/apple-build-environment.md)。

iPhone/iOS：

```bash
./build.sh ios
```

iOS 打包和真机分发需要 macOS + Xcode + Apple Developer 证书。iOS 全局代理不能由普通 App 直接修改系统设置，必须使用 Network Extension 的 Packet Tunnel Provider entitlement；当前仓库已加入 Packet Tunnel overlay、gomobile/tun2socks 后端和主 App 到 NetworkExtension 的桥接代码，具体接入要求见 [ios/README.md](ios/README.md) 和 [docs/apple-clients.md](docs/apple-clients.md)。在 macOS 构建机上可以运行 [scripts/Prepare-iOSPacketTunnel.sh](scripts/Prepare-iOSPacketTunnel.sh) 生成 gomobile framework、暂存 Swift overlay，并在安装 Ruby `xcodeproj` 后自动补 Xcode Packet Tunnel target；推荐用 `bash scripts/Prepare-iOSPacketTunnel.sh` 调用。

Android：

```bash
# Docker/fyne-cross 路径：构建后自动调用 scripts/Inject-AndroidVpnService.ps1
scripts/package-cross.sh windows
scripts/package-cross.sh linux
scripts/package-cross.sh android
scripts/package-cross.sh android-amd64

export ANDROID_NDK_HOME=/PATH/TO/android-ndk
./build.sh android
```

Android 已经接入真实 VPN 模式：运行页的“启用全局代理”会请求 Android VPN 授权，启动 `DetourVpnService`，建立系统 TUN，并通过 Go/JNI 把 TUN 流量交给 `tun2socks`，再转发到本机 SOCKS5 `127.0.0.1:7776`。本机 HTTP 代理仍监听 `127.0.0.1:7777`。

Android 默认按 Android 14 / API 34 打包。Fyne 先生成基础 APK；随后 `scripts/package-cross.sh android*` 会自动执行 VPN Service 注入：优先调用 Docker 版 [scripts/Inject-AndroidVpnService.ps1](scripts/Inject-AndroidVpnService.ps1)，如果当前 Windows/Docker Desktop 出现卷挂载异常，会自动回退到本机版 [scripts/Inject-AndroidVpnServiceLocal.ps1](scripts/Inject-AndroidVpnServiceLocal.ps1)。本机版复用 `D:\AndroidTools\sdk` 和 JDK，自动设置 `JAVA_HOME`，完成 `classes2.dex` 注入、最终 Manifest 编译、`targetSdkVersion=34`、`foregroundServiceType=systemExempted`、`extractNativeLibs=false`、16KB native lib 对齐和重新签名。

Fyne Android 包不会自动编译仓库里的 Java Service，所以不要直接使用未注入的基础 APK。推荐始终使用封装脚本：

```bash
# 真机 arm64 包
scripts/package-cross.sh android

# x86_64 模拟器包
scripts/package-cross.sh android-amd64
```

构建完成后，最终 APK 应满足：`targetSdkVersion=34`、包含 `classes2.dex`、APK 签名验证通过，并且未压缩 native library 通过 16KB 对齐检查。只有当 Docker 注入和本机 fallback 都失败时，才需要单独运行本机注入脚本排查环境：

```powershell
PowerShell -ExecutionPolicy Bypass -File .\scripts\Inject-AndroidVpnServiceLocal.ps1 -ApkPath dist/android-arm64/Detour2.apk
PowerShell -ExecutionPolicy Bypass -File .\scripts\Inject-AndroidVpnServiceLocal.ps1 -ApkPath dist/android-amd64/Detour2.apk
```

注入脚本是 [scripts/Inject-AndroidVpnService.ps1](scripts/Inject-AndroidVpnService.ps1)，会把 [android/src/main/java/github/com/observerss/detour2fyne/DetourVpnService.java](android/src/main/java/github.com/observerss/detour2fyne/DetourVpnService.java) 编译成 `classes2.dex` 并合并进 APK。Windows 本机 fallback 是 [scripts/Inject-AndroidVpnServiceLocal.ps1](scripts/Inject-AndroidVpnServiceLocal.ps1)，用于 Docker Desktop 卷挂载异常时复用本机 `D:\AndroidTools\sdk` 和 JDK 完成同样的注入、16KB 对齐和签名。

模拟器验证要点：安装 `dist/android-amd64/Detour2.apk`，启动本地代理后应看到 `127.0.0.1:7776` 和 `127.0.0.1:7777` 监听；启用全局代理并授权后，`adb shell ip addr show tun0` 应出现 `10.21.0.2/32`，`dumpsys connectivity` 中应出现 `VPN CONNECTED` 和 `VALIDATED`。当前测试机上已验证 `DetourVpnService` 前台服务运行、`tun0` 建立、VPN 网络校验通过。

iOS 版本需要单独接入 Apple Network Extension Packet Tunnel；移动端开机自启仍不支持。

### Windows

推荐使用统一的跨平台打包脚本生成 Windows 发布产物。Windows 本机不需要 WSL，直接通过 PowerShell 包装脚本调用 Git Bash：

```powershell
PowerShell -ExecutionPolicy Bypass -File .\scripts\package-cross.ps1 windows
```

也可以用 CMD 入口：

```cmd
scripts\package-cross.cmd windows
```

这两个入口最终都会调用 `scripts/package-cross.sh windows`。如果 Git Bash 不在默认安装路径，可以先设置 `GIT_BASH` 指向 `bash.exe`。

产物会输出到 `dist/windows-amd64/`，包括 `Detour2.exe` 和 `Detour2.zip`。这条路径会创建临时 workspace、vendor 依赖并使用 `fyne-cross windows -arch=amd64` 构建，是当前推荐的 Windows 打包方式。

Windows zip 会附带 `README-Windows.md` 和 `tools\Sign-WindowsLocal.ps1`。这个签名脚本只用于本机开发/测试，它会把 self-signed 证书导入当前用户证书库；不要要求普通用户执行它，也不要把 self-signed 签名当正式发行签名。

如果只是在 Windows 本机快速编译当前平台，可以运行：

```bash
./build.sh windows
```

本机快速编译通常会在仓库根目录生成 `detour2-fyne.exe`。需要把它整理成 `dist/windows-amd64/Detour2.exe` 和 zip 时，再运行：

```powershell
PowerShell -ExecutionPolicy Bypass -File .\scripts\Package-WindowsLocal.ps1
```

如果 Windows 的 Smart App Control 或 SmartScreen 拦截当前测试包，原因通常是 exe 没有 Authenticode 签名，或者 zip/exe 带有 Mark-of-the-Web。正式发布应该使用 CA 签发的代码签名证书，并给 exe/installer 加 timestamp；本机临时测试可以在 PowerShell 中先解锁 zip，再用本仓库的本机测试证书脚本签名：

```powershell
Unblock-File .\dist\windows-amd64\Detour2.zip
Expand-Archive .\dist\windows-amd64\Detour2.zip .\dist\windows-amd64\Detour2 -Force
PowerShell -ExecutionPolicy Bypass -File .\scripts\Sign-WindowsLocal.ps1 -ExePath .\dist\windows-amd64\Detour2\Detour2.exe -Rezip -ZipPath .\dist\windows-amd64\Detour2-local-signed.zip
```

这个脚本只适合自己的测试机：它会创建一个当前用户的 self-signed code signing cert，并导入到当前用户的 `Root` 和 `TrustedPublisher`。不要把这个 self-signed 产物当正式 release 发给别人。

不要直接从 `\\wsl.localhost\...` 路径启动 exe。Smart App Control 会把这种来源当作不熟悉的网络/WSL 应用，更容易直接拦截且没有“仍要运行”按钮。测试 Windows 包时建议在 Windows PowerShell 里运行下面的准备脚本，它会把 zip 解到 `%LOCALAPPDATA%\Programs\Detour2`，移除来源标记，执行本机测试签名，然后启动：

```powershell
PowerShell -ExecutionPolicy Bypass -File \\wsl.localhost\Ubuntu-22.04\home\jingc\Projects\Personal\detour2-fyne\scripts\Prepare-WindowsLocal.ps1 -Run
```

如果 Smart App Control 仍然拦截 self-signed 测试签名，说明当前系统策略只接受有信誉的正式签名或 Microsoft Store 分发；这不是程序逻辑错误，需要走正式代码签名证书、SignPath 开源签名或 Store 分发。

Windows 开机自启写当前用户 `Run` 注册表，不再通过 PowerShell + `WScript.Shell` 动态创建快捷方式。这只能减少误报触发面，不能替代正式签名和发行信誉。

更完整的 Windows 运行排查步骤见 [docs/windows-testing.md](docs/windows-testing.md)。
