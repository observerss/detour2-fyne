# Detour2 Windows 便携版说明

这是 Detour2 的 Windows 便携版。请先把 zip 解压到本机目录，再运行 `Detour2.exe`。

## 推荐用法

- 先完整解压 zip，再运行程序。
- 不要直接在 zip 压缩包内运行程序。
- 不要从 `\\wsl.localhost\...`、网络共享目录或其它不可信来源路径直接启动程序。
- 如果要正式公开发布，应该使用可信 CA 签发的 Authenticode 代码签名证书，并给签名加 timestamp。

## 本机测试签名工具

压缩包里的 `tools\Sign-WindowsLocal.ps1` 只适合开发者自己的测试机使用。

它会在当前 Windows 用户下创建或复用一个 self-signed 代码签名证书，把证书导入当前用户的 `Root` 和 `TrustedPublisher` 证书库，然后给指定的 exe 签名。

示例：

```powershell
PowerShell -ExecutionPolicy Bypass -File .\tools\Sign-WindowsLocal.ps1 -ExePath .\Detour2.exe
```

签名后重新打 zip：

```powershell
PowerShell -ExecutionPolicy Bypass -File .\tools\Sign-WindowsLocal.ps1 -ExePath .\Detour2.exe -Rezip -ZipPath .\Detour2-local-signed.zip
```

## 重要提醒

不要要求普通用户运行这个本机测试签名脚本。self-signed 证书不能建立公开信任，也不能当作正式发行签名。

如果 Windows SmartScreen 或 Smart App Control 拦截当前构建，应该改用正式代码签名、Microsoft Store 分发，或其它可信发布渠道。
