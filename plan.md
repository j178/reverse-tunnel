# 极简 Server/Client 反向 TCP 隧道代理实施计划（MVP）

一句话定义：本项目实现一种常见的“反向隧道（Reverse Tunnel）”程序——由内网 `client` 主动连接公网 `server`，让外部流量经 `server` 转发到 `client` 所在网络内的目标 TCP 服务。

## 1) 程序通常叫什么

- 常见名称：**反向隧道**（Reverse Tunnel）
- 也常见于：**反向代理隧道**、**TCP Reverse Tunnel**
- 你的需求最贴近：**单 Client 反向 TCP 隧道**

## 2) 目标与非目标（MVP 边界）

### 目标（必须实现）
- 使用 Go 1.25
- `server` 监听两个端口：
  - A：外部流量入口
  - B：`client` 连接入口
- `client` 主动连接 `server:B`
- `server` 将 A 收到的连接，转发给当前 B 上在线的 `client`
- `client` 将收到的流量转发到本地目标服务（启动参数指定单目标 `host:port`）
- 纯 TCP 透传
- 单 `client`（MVP）
- 无鉴权（按当前需求）

### 非目标（本期不做）
- 多 client 路由/负载均衡
- TLS/mTLS、Token 鉴权
- HTTP 七层代理能力
- 管理后台、配置热更新

## 3) 架构设计

### 3.1 双连接模型（推荐）
采用“**控制连接 + 数据连接**”模式：

1. `client` 启动后，先连到 `server:B`，建立常驻控制连接。
2. 外部连接进入 `server:A` 后，`server` 生成 `connID`。
3. `server` 通过控制连接通知 `client`：有一个 `connID` 需要处理。
4. `client` 新建一条到 `server:B` 的数据连接，并携带 `connID`。
5. `server` 用 `connID` 将这条数据连接和 A 侧外部连接配对。
6. 配对成功后，`client` 连接本地目标服务，双方进行双向字节转发。

### 3.2 端口职责
- A 端口：只负责外部连接接入。
- B 端口：负责 client 控制连接与数据连接接入。

### 3.3 生命周期
- `A accept` -> 生成 `connID` -> 通知 client -> 等待配对（超时）-> 转发 -> 清理
- 任一侧断开：双向关闭并回收状态，防止 goroutine/连接泄漏。

## 4) 模块设计（目录建议）

```text
cmd/
  reverse-tunnel/
    main.go          # 单一 binary，通过 subcommand 启动 client/server
internal/
  server/
    listener_a.go      # A 端口接入
    listener_b.go      # B 端口接入（控制/数据）
    session.go         # 单 client 会话状态
    matcher.go         # connID 配对表
  client/
    control.go         # 控制连接、接收 NEW_CONN
    dialer.go          # 数据连接回连 + 本地目标连接
  protocol/
    message.go         # 控制消息定义与编解码
  transport/
    relay.go           # 双向 io.Copy 转发与关闭策略
  common/
    log.go             # 统一日志包装
    config.go          # 配置与参数结构
```

### 模块职责
- `cmd/reverse-tunnel`：参数解析与进程启动（`server` / `client` 子命令）。
- `internal/server`：监听、连接管理、配对与调度。
- `internal/client`：控制面处理、数据面回连、连接本地目标。
- `internal/protocol`：最小协议定义（消息、握手）。
- `internal/transport`：纯转发与资源回收。

## 5) 功能规划

### P0（本次必须）
- 双端口监听（A/B）
- 单 client 控制连接
- A 收到连接后通知 client
- client 建立数据连接并回传 `connID`
- server 完成配对
- client 连接本地目标并做双向 TCP 透传
- 超时与异常关闭
- 基础日志

### P1（下一阶段）
- client 自动重连（指数退避）
- 心跳保活（PING/PONG）
- 简单统计指标（活动连接数/失败数）

### P2（演进）
- 多 client（按 clientID/标签路由）
- TLS + Token/mTLS
- HTTP/HTTPS 七层代理扩展

## 6) 协议设计（MVP 最小）

### 6.1 控制消息
建议最小消息：
- `NEW_CONN {connID}`：server -> client，通知处理新连接。
- `CLOSE {connID}`（可选）：任一侧通知提前结束。

消息格式可先用行文本或轻量 JSON（保持简单即可）。

### 6.2 数据连接握手
- client 建立到 B 的数据连接后，首包发送 `connID`。
- server 校验 `connID` 并与 A 侧连接配对。
- 未命中或超时：拒绝该数据连接并记录日志。

### 6.3 超时策略
- A 侧等待数据连接配对超时（如 10s，可配置）。
- 超时后关闭 A 连接并清理 `connID`。

## 7) 实现步骤（按里程碑）

1. **工程初始化**
   - 初始化 Go 模块（Go 1.25）
   - 创建 `cmd` 与 `internal` 目录结构
   - 定义配置参数（A/B 端口、target、timeout）

2. **server 基础能力**
   - 实现 A/B 两个监听器
   - 实现单 client 会话注册
   - 实现 `connID` 分配与待配对表

3. **client 基础能力**
   - 主动连接 B（控制连接）
   - 处理 `NEW_CONN`
   - 建立数据连接并发送 `connID`

4. **转发链路打通**
   - server 侧完成 A 连接与数据连接配对
   - client 拨号本地 target
   - 双向转发（两路 `io.Copy`）

5. **稳定性与清理**
   - 超时、断链、异常回收
   - 完善日志与错误分类

6. **联调验证**
   - 本地服务压测与异常场景验证
   - 输出运行说明

## 8) 验证方案

### 8.1 本地联调
- 本地先起一个目标服务，例如 `127.0.0.1:8080`。
- 启动 server：监听 A/B。
- 启动 client：连接 B，target 指向 `127.0.0.1:8080`。
- 访问 A 端口，确认能透传到 target。

### 8.2 异常验证
- client 不在线时访问 A：应超时失败。
- client 中途退出：新连接应失败，已有连接应收敛关闭。
- target 不可达：连接应快速报错并清理。

## 9) 风险与后续演进

### 当前风险（按需求接受）
- 无鉴权：B 端口若暴露，存在被伪造连接风险。
- 无加密：明文传输，存在被窃听风险。
- 单 client：单点故障，无横向扩展。

### 后续建议
- 最先补齐：Token 鉴权 + TLS。
- 再演进：多 client 路由与健康检查。
- 最后扩展：HTTP 层能力与可观测体系。

## 10) 运行参数建议（示例）

### server
- 子命令：`server`
- `--listen-a :18080`
- `--listen-b :19090`
- `--pair-timeout 10s`

### client
- 子命令：`client`
- `--server 127.0.0.1:19090`
- `--target 127.0.0.1:8080`

### 启动示例
- `go run ./cmd/reverse-tunnel server --listen-a :18080 --listen-b :19090 --pair-timeout 10s`
- `go run ./cmd/reverse-tunnel client --server 127.0.0.1:19090 --target 127.0.0.1:8080`

## 11) 完成标准（Definition of Done）

- 外部流量经 A 可稳定到达 client 本地 target。
- 单 client 模式稳定运行，连接可正常创建/关闭。
- 超时与异常断链都可正确回收。
- 文档明确无鉴权/无加密风险与后续路线。