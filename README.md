# xray-installer

在 VPS 本机执行的 Go 命令行工具。

启动后会交互式询问：
- VPS 绑定的域名
- Reality 伪装目标域名（直接回车默认 `www.bing.com`）

FlClash 节点名称固定为博客长模板中的 `[自建 1] 美国家宽-Reality`，然后自动继续安装。
安装始终按初始化流程重新生成节点参数并备份、覆盖已有配置；保留现有节点请使用 `update`。

## 一键安装并立即执行

> `install.sh` 会从 GitHub Releases 下载预编译二进制，因此仓库需要先发布一个 tag/release。

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/liuwentao/xray-installer/main/install.sh) install
```

仅安装，不立即运行：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/liuwentao/xray-installer/main/install.sh) install --no-run
```

安装后以 dry-run 方式立刻启动：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/liuwentao/xray-installer/main/install.sh) install -- --dry-run
```

一键卸载：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/liuwentao/xray-installer/main/install.sh) uninstall
```

```bash
xray-installer
```

它会自动：
- 检查运行环境（root / Linux / systemd / Debian/Ubuntu/Arch Linux）
- 检查 443 端口占用情况并给出提示（不再前置拦截安装）
- 安装 Xray
- 生成 VLESS + Reality + Vision 服务端配置
- 启动并启用 `xray` systemd 服务
- 生成文章同款长版 FlClash 配置 `proxy.yaml`
- 生成独立的 Mihomo 配置 `mihomo.yaml`，使用同一个 VLESS + Reality 节点
- 生成原生 Xray 客户端配置 `xray-client.json`，使用同一个 VLESS + Reality + Vision 节点
- 生成小火箭（Shadowrocket）分享链接和 PNG 二维码，安装完成后在终端显示导入二维码
- 支持 `uninstall`
- 支持 `update` 更新 Xray，保留现有配置
- 支持 `--dry-run`
- 支持 GitHub Release 二进制一键安装脚本 `install.sh`

生成的 FlClash 配置默认启用 `reality-opts.support-x25519mlkem768: true`，用于兼容新版 Xray 的 REALITY 握手要求。

## 当前边界

- 只支持 **Debian / Ubuntu / Arch Linux + systemd**
- 需要在 **VPS 本机**运行
- 域名 DNS 需要提前配好
- 如果用了 Cloudflare，需要保证是 **DNS Only**

## 构建

```bash
go build ./cmd/xray-installer
```

## 运行

```bash
sudo ./xray-installer
```

## 帮助

```bash
./xray-installer --help
./xray-installer --version
```

## 更新 Xray

将已安装的 Xray 更新到最新稳定版：

```bash
sudo xray-installer update
```

指定 Xray 版本（也可用于切换到旧版，请确保版本与现有配置兼容）：

```bash
sudo xray-installer update --xray-version v26.3.27
```

预览更新，不下载文件、不修改系统、不重启服务：

```bash
xray-installer update --dry-run
```

也可通过脚本调用已安装的安装器：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/liuwentao/xray-installer/main/install.sh) update
```

更新要求已有完整的 Xray 安装和配置，不会重新询问域名或生成 UUID、Reality 密钥及 FlClash 配置。如果缺少 `xray-client.json`，会读取原有 `install-result.json` 中的节点参数补生成；已有客户端文件保持原样。安装记录缺失或参数不完整时，会在更新前报错。
通过 [Xray 官方安装脚本](https://github.com/XTLS/Xray-install/blob/main/install-release.sh) 更新，保留现有 systemd 服务设置，更新后校验原有服务端配置，再重启并检查服务状态。
如果校验失败，会返回错误并跳过重启；已替换的二进制不会自动回滚。`--version` 仍用于显示安装器版本。

## Mihomo 配置

安装完成后，除 FlClash 长版配置外，还会生成 `/usr/local/etc/xray-installer/mihomo.yaml`：

```bash
sudo cat /usr/local/etc/xray-installer/mihomo.yaml
```

将该文件复制到运行 Mihomo 的客户端，保存为 `mihomo.yaml`，然后执行：

```bash
mihomo -t -f ./mihomo.yaml
mihomo -f ./mihomo.yaml
```

配置提供本地 HTTP / SOCKS 混合端口 `127.0.0.1:7890`，在浏览器或系统中设置该代理后使用。
局域网 IPv4 地址直连，其余流量默认通过自建节点；`PROXY` 组也可选择 `DIRECT`。不自动启用 TUN，不依赖在线规则集或 Geo 数据下载。
使用支持 `reality-opts.support-x25519mlkem768` 的 Mihomo 版本，字段说明见 [Mihomo TLS 文档](https://wiki.metacubex.one/config/proxies/tls/)。

`update` 不补生成 Mihomo 配置。已有安装可继续使用原来的 `proxy.yaml`（同样使用 Mihomo 配置格式）。

## 原生 Xray 客户端配置

安装或更新补生成后的文件位于 `/usr/local/etc/xray-installer/xray-client.json`，权限为 `0600`。将它复制到客户端，执行：

```bash
xray run -test -config ./xray-client.json
xray run -config ./xray-client.json
```

客户端提供 SOCKS5 `127.0.0.1:1080`（支持 UDP）和 HTTP `127.0.0.1:8080`，流量通过自建节点转发，不启用 TUN。Reality 使用官方文档中的 [`password` 客户端字段](https://xtls.github.io/config/transports/reality.html)，内容为服务端公钥；客户端文件不包含服务端私钥。请使用支持该字段的 Xray 版本。

`update` 即使无需升级 Xray 版本，也会补生成缺失的原生客户端配置；`update --dry-run` 只预览，不写文件。

## 小火箭（Shadowrocket）导入

安装完成后，打开小火箭的扫描功能，扫描终端显示的二维码即可添加同一个 VLESS + Reality + Vision 节点。
分享链接按照 [VLESS 分享链接标准](https://github.com/XTLS/Xray-core/discussions/716)生成，包含服务器地址、UUID、Reality 公钥、Short ID、SNI 和 Vision flow。

也可以复制分享链接，再在小火箭中从剪贴板导入：

```bash
sudo cat /usr/local/etc/xray-installer/shadowrocket.txt
```

如果终端太窄导致二维码换行，将 `/usr/local/etc/xray-installer/shadowrocket.png` 复制到本地打开后扫描，或保存到手机相册后在小火箭中选择图片识别。
二维码在本地生成，内容与 `shadowrocket.txt` 中的链接一致；这两个文件包含节点访问凭据，权限为 `0600`，请妥善保管。

`--dry-run` 只预览文件路径，不保存文件或显示可导入的二维码。`update` 不会为已有安装补生成这些文件。
连接还需小火箭版本支持服务端当前的 Reality 握手要求，需在实际设备上验证。

## 卸载

```bash
sudo ./xray-installer uninstall
```

## Dry run

```bash
./xray-installer --dry-run
```

生成文件：
- Xray 配置：`/usr/local/etc/xray/config.json`
- FlClash 配置：`/usr/local/etc/xray-installer/proxy.yaml`
- Mihomo 配置：`/usr/local/etc/xray-installer/mihomo.yaml`
- 原生 Xray 客户端配置：`/usr/local/etc/xray-installer/xray-client.json`
- 小火箭分享链接：`/usr/local/etc/xray-installer/shadowrocket.txt`
- 小火箭二维码：`/usr/local/etc/xray-installer/shadowrocket.png`
- 安装记录：`/usr/local/etc/xray-installer/install-result.json`

安装完成后，可用下面命令直接获取 FlClash 配置：

```bash
sudo cat /usr/local/etc/xray-installer/proxy.yaml
```

## 验证

```bash
go test ./...
go build ./cmd/xray-installer
```

## 参考

- 文章：<https://manateelazycat.github.io/2026/04/09/best-proxy/>
- Xray 安装脚本：<https://github.com/XTLS/Xray-install>
- Xray 命令文档：<https://xtls.github.io/document/command>

## License

GPL-3.0
