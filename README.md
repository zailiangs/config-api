# config-api

一个使用 Go 标准库实现的轻量级服务器配置 API。首版提供 HTTP 接口，用于修改 Ubuntu/Debian 的 SSH 监听端口。

## 安全警告

> **该服务默认监听 `0.0.0.0:8080`，且不提供鉴权或 TLS。任何能够访问该端口的人都可以修改服务器 SSH 端口。**

仅应在可信网络中运行，并通过云安全组、防火墙或反向代理限制调用来源。修改前必须提前放行新的 SSH 端口，否则当前 SSH 会话断开后可能无法重新连接。

## 工作方式

请求：

```text
GET /config/ssh?port=18822
```

服务会：

1. 校验端口为 `1-65535` 的十进制整数。
2. 修改 `/etc/ssh/sshd_config` 中首个全局 `Port`，删除其他全局重复项。
3. 将原配置备份至 `/etc/ssh/sshd_config.config-api.bak`。
4. 使用同目录临时文件原子替换配置。
5. 执行 `sshd -t` 和 `sshd -T`，确认唯一有效端口等于请求值。
6. 执行 `systemctl restart ssh`；找不到该服务时回退至 `sshd`。
7. 确认服务处于 active 状态。验证或重启失败时恢复原配置并再次启动 SSH。

服务不会修改 UFW、其他防火墙或云安全组。它也不会递归修改 `Include` 文件；如果 Include 导致多个有效 SSH 端口，请求会失败并回滚。

## 要求

- Ubuntu 或 Debian，使用 systemd
- OpenSSH Server
- Go 1.24 或更高版本（仅构建时需要）
- 服务以 root 用户运行

## 构建与安装

```bash
git clone https://github.com/zailiangs/config-api.git
cd config-api
make check
sudo install -m 0755 bin/config-api /usr/local/sbin/config-api
sudo install -m 0644 deploy/config-api.service /etc/systemd/system/config-api.service
sudo systemctl daemon-reload
sudo systemctl enable --now config-api
```

检查状态：

```bash
systemctl status config-api
curl http://127.0.0.1:8080/healthz
```

## 使用

务必先放行新端口：

```bash
# 示例，仅在启用 UFW 时执行
sudo ufw allow 18822/tcp
```

调用接口：

```bash
curl "http://服务器地址:8080/config/ssh?port=18822"
```

成功响应：

```json
{"success":true,"port":18822,"changed":true,"rolled_back":false}
```

重复设置相同端口时不会修改文件或重启 SSH：

```json
{"success":true,"port":18822,"changed":false,"rolled_back":false}
```

## 配置

通过 systemd unit 或进程环境变量配置：

| 环境变量 | 默认值 | 说明 |
| --- | --- | --- |
| `LISTEN_ADDR` | `0.0.0.0:8080` | HTTP 监听地址 |
| `SSHD_CONFIG_PATH` | `/etc/ssh/sshd_config` | sshd 主配置文件 |
| `SSHD_BINARY` | 自动查找 | sshd 可执行文件路径 |

修改 unit 后执行：

```bash
sudo systemctl daemon-reload
sudo systemctl restart config-api
```

更安全的监听设置是将 `LISTEN_ADDR` 改为 `127.0.0.1:8080`，再通过 SSH 隧道或受控反向代理调用。

## 故障恢复

接口在验证或 SSH 重启失败时会自动恢复原配置。若自动恢复仍失败，可在服务器控制台执行：

```bash
sudo cp /etc/ssh/sshd_config.config-api.bak /etc/ssh/sshd_config
sudo sshd -t
sudo systemctl restart ssh
```

日志通过 journald 查看：

```bash
journalctl -u config-api
```

## 卸载

```bash
sudo systemctl disable --now config-api
sudo rm /etc/systemd/system/config-api.service
sudo rm /usr/local/sbin/config-api
sudo systemctl daemon-reload
```

## 开发

```bash
make test
make vet
make build
```

## License

[MIT](LICENSE)
