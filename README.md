# Reverse Tunnel

极简反向 TCP 隧道（Reverse Tunnel）MVP：
- `server` 监听两个端口：
  - A：外部流量入口
  - B：client 连接入口
- `client` 主动连接 B，并将流量转发到本地目标服务

## 环境
- Go 1.25
- macOS/Linux（脚本使用 `bash`、`python3`、`curl`）

## 手动运行

1) 启动 server

```bash
go run ./cmd/server --listen-a :18080 --listen-b :19090 --pair-timeout 10s
```

2) 启动 client

```bash
go run ./cmd/client --server 127.0.0.1:19090 --target 127.0.0.1:8080
```

3) 访问 A 端口（示例）

```bash
curl http://127.0.0.1:18080
```

## 说明
- 当前版本是 MVP：单 client、无鉴权、纯 TCP 透传。
- 详细规划见 [plan.md](plan.md)。
