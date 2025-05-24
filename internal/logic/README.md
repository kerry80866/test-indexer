# logic/ 目录结构说明

该目录是核心逻辑层，负责处理从 Solana gRPC 流接收到的区块数据，进行结构适配、事件解析、封装和分发。
结构清晰，遵循 Clean Architecture 分层设计，职责明确。

## 📦 模块总览

| 包名         | 职责说明        | 核心函数/结构                    |
|--------------|-------------|----------------------------|
| `grpc/`       | gRPC 流接收与驱动流程 | `BlockProcessor`           |
| `txadapter/`  | 交易结构适配      | `AdaptGrpcTx`              |
| `core/`     | 内部通用数据结构    | `AdaptedTx`, `TxContext`, `Event` |
| `eventparser/`| 提取业务事件      | `ExtractTxEvents`          |

---

## 🔒 模块依赖

- `core` 是最低层，不依赖其他任何逻辑包
- `txadapter` → 只依赖 `core`
- `eventparser` → 依赖 `core`, `events`
- `grpc` → 调用所有处理模块，但不参与内部细节

---

## ✅ 模块调用流程

```
gRPC block stream
    ↓
grpc.BlockProcessor
    ↓
txadapter.AdaptGrpcTx(rawTx)
    ↓
core.AdaptedTx
    ↓
eventparser.ExtractTxEvents(adaptedTx)
    ↓
events.TxEvents
    ↓
dispatcher → Kafka / DB
```

---

## 📁 grpc/
> **gRPC 连接与消费层** —— 接收区块推送流，并驱动主流程

- `grpc_stream.go`：gRPC 连接、重连、订阅处理
- `block_processor.go`：Block 级别处理，内部并发调用 `txadapter` 与 `eventparser`

**作用：** 主控器，协调各个处理模块的调用

---

## 📁 core/
> **内部领域模型层** —— 定义系统内部通用数据结构，不依赖外部协议，不包含业务逻辑。

- `tx.go`：定义 `AdaptedTx`, `TxContext`，代表结构化交易数据。
- `instruction.go`：定义 `AdaptedInstruction` 结构体。
- `balance.go`：定义 `TokenBalance`, `TokenDecimals` 等结构。

**被谁依赖：** `txadapter`, `eventparser`, `events`, `dispatcher`  
**不应依赖：** 任意外部模块（保持纯结构）

---

## 📁 txadapter/
> **交易适配器层** —— 将 gRPC 原始交易数据转为内部结构 `AdaptedTx`

- `grpc_tx.go`：核心函数 `AdaptGrpcTx()`
- `mint_resolver.go`：Token mint 精度缓存查询辅助工具

**输入：** `*pb.SubscribeUpdateTransactionInfo`  
**输出：** `*core.AdaptedTx`

---

## 📁 eventparser/
> **事件解析器层** —— 从 `AdaptedTx` 中提取有意义的业务事件

- `extract.go`：核心函数 `ExtractTxEvents()`，用于交易事件识别与封装

**输入：** `*core.AdaptedTx`  
**输出：** `*events.TxEvents`（包含事件列表）

---

## 📁 events/
> **事件结构层** —— 定义所有事件结构与 `Event` 接口

- `event.go`：定义 `Event` 接口、`TxEvents`、各类事件结构（如 `TradeEvent`, `TransferEvent`）

**依赖：** `core`  
**被谁依赖：** `eventparser`, `dispatcher`

---

