# Windows NSSM 自启动

NSSM 可以把 CCX 可执行文件注册为 Windows 服务，适合非 Docker 部署。

## 准备

1. 从 Release 下载 Windows 可执行文件，例如 `ccx-windows-amd64.exe`。
2. 创建目录 `C:\ccx`。
3. 将可执行文件放到 `C:\ccx\ccx-windows-amd64.exe`。
4. 下载并安装 NSSM。

## 安装服务

以管理员身份打开 PowerShell：

```powershell
nssm install ccx C:\ccx\ccx-windows-amd64.exe
nssm set ccx AppDirectory C:\ccx
nssm set ccx AppEnvironmentExtra PROXY_ACCESS_KEY=your-proxy-access-key PORT=3000 ENABLE_WEB_UI=true APP_UI_LANGUAGE=zh-CN
nssm set ccx Start SERVICE_AUTO_START
nssm start ccx
```

## 常用命令

```powershell
nssm status ccx
nssm restart ccx
nssm stop ccx
nssm remove ccx confirm
```

## 日志

如需输出日志，可以配置 NSSM stdout/stderr：

```powershell
mkdir C:\ccx\logs
nssm set ccx AppStdout C:\ccx\logs\stdout.log
nssm set ccx AppStderr C:\ccx\logs\stderr.log
nssm set ccx AppRotateFiles 1
nssm set ccx AppRotateOnline 1
nssm set ccx AppRotateBytes 10485760
```
