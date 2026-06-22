# nogocommons

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

**nogocommons** is the common base library for the [NogoChain](https://github.com/nogochain) ecosystem. It provides the foundational building blocks used by all NogoChain sub-projects — from full nodes and wallets to mining pools and standalone miners.

## Role in the Ecosystem

```
┌──────────────────────────────────────────────────────────────┐
│                     NogoChain Ecosystem                       │
├───────────┬───────────┬───────────┬───────────┬──────────────┤
│ nogocore  │nogowallet │ nogopool  │nogominer  │  (your app)  │
│ Full Node │ HD Wallet │Mining Pool│Solo Miner │              │
└─────┬─────┴─────┬─────┴─────┬─────┴─────┬─────┴──────┬───────┘
      │           │           │           │            │
      └───────────┴───────────┴───────────┴────────────┘
                         │
              ┌──────────▼──────────┐
              │    nogocommons      │
              │  Common Base Library │
              └─────────────────────┘
```

Every project in the NogoChain ecosystem depends on **nogocommons** for:
- **Network wire protocol** — message serialization and deserialization
- **Consensus engine** — the NogoPow proof-of-work algorithm
- **Address encoding** — P2PKH, P2SH, P2WPKH, P2WSH, P2TR, P2A addresses
- **Elliptic curve cryptography** — secp256k1 ECDSA, Schnorr, key management
- **Database abstraction** — pluggable block and metadata storage backends
- **P2P networking** — peer management, connection handling, address discovery
- **Chain configuration** — network parameters for mainnet, testnet, regtest, signet, simnet

## Packages

| Package | Description |
|---------|-------------|
| **[address](./address/)** | NogoChain address encoding and decoding. Supports P2PKH, P2SH, P2WPKH, P2WSH, P2TR (Taproot), P2A (Pay-to-Anchor), and raw public key addresses. Includes base58 and bech32/bech32m sub-packages. |
| **[addrmgr](./addrmgr/)** | Concurrency-safe address manager for peer discovery. Maintains a cryptographically-randomized peer address pool with group segregation, routability awareness, Tor support, and quality-based selection bias. |
| **[chaincfg](./chaincfg/)** | Network parameter definitions for MainNet, TestNet3, TestNet4, RegressionNet, SigNet, and SimNet. Includes consensus rules (block time, max sizes), address encoding prefixes, BIP32 HD key magics, and the NogoPow economic model parameters. |
| **[chainhash](./chainhash/)** | Generic 32-byte hash type with hex encoding/decoding. Provides `Hash`, `DoubleHash`, tagged hashes (BIP-340/BIP-341), and convenience constructors (`NewHash`, `NewHashFromStr`). |
| **[connmgr](./connmgr/)** | Generic Bitcoin network connection manager. Maintains outbound connection targets, sources peers, handles banning, limits max connections, and supports Tor lookup. |
| **[database](./database/)** | Pluggable block and metadata storage database interface. Provides `DB` (Begin/View/Update/Close), `Tx` (StoreBlock/FetchBlock/FetchBlockRegion/PruneBlocks), `Bucket`, and `Cursor` interfaces. Default backend: ffldb (LevelDB metadata + flat files). |
| **[limits](./limits/)** | Cross-platform OS resource limit setting. No-op on Windows, `setrlimit`-based on Unix. |
| **[neutrino](./neutrino/)** | Light-client (SPV) protocol support. Includes header-first sync, filter database (GCS), block notifications, ban manager, and chain synchronization utilities. |
| **[nogoec](./nogoec/)** | Elliptic curve cryptography on secp256k1. ECDSA signing/verification, public key recovery, Schnorr signatures, private key management (WIF-compatible), and field arithmetic via libsecp256k1. |
| **[nogopow](./nogopow/)** | **NogoPow consensus engine** — the ASIC-resistant proof-of-work algorithm. Five-stage pipeline: Nonce → Salsa20/8 → Scrypt-like Smix → Matrix Multiply (256×256) → FNV Hash Reduce → Keccak256. Uses PI controller for difficulty adjustment with double exponential smoothing. |
| **[nogoutil](./nogoutil/)** | High-level utility functions and types. Re-exports address types, provides `Amount` (satoshi arithmetic), `WIF` (wallet import format), `Block`/`Tx` memoized wrappers, Bloom filters, HD keychain, GCS filters, and app data directory resolution. |
| **[ossec](./ossec/)** | OpenBSD security feature stubs. Provides no-op implementations of `Unveil` and `Pledge` for non-OpenBSD platforms. |
| **[peer](./peer/)** | Concurrent-safe Bitcoin peer management. Handles version negotiation, full-duplex message I/O, inventory trickling, keep-alive pings, and callback-based message dispatch for all wire protocol messages. |
| **[txscript](./txscript/)** | Transaction script compatibility stubs. Provides the `ScriptClass`, `SigHashType`, `MultiPrevOutFetcher`, and `SecretsSource` interfaces for integration with the UTXO engine in nogocore. |
| **[v2transport](./v2transport/)** | BIP324 v2 encrypted P2P transport protocol. Implements the ElligatorSwift-based handshake, ChaCha20-Poly1305 encryption, and garbage terminator exchange for encrypted peer communication. |
| **[wire](./wire/)** | Bitcoin wire protocol implementation. Supports all P2P message types (MsgVersion, MsgBlock, MsgTx, MsgInv, MsgGetData, etc.), message serialization/deserialization, service flags, and network magic numbers. |

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/nogochain/nogocommons/chaincfg"
    "github.com/nogochain/nogocommons/chainhash"
    "github.com/nogochain/nogocommons/nogoec"
    "github.com/nogochain/nogocommons/nogoutil"
)

func main() {
    // 1. Generate a secp256k1 key pair
    privKey, err := nogoec.NewPrivateKey()
    if err != nil {
        panic(err)
    }
    pubKey := privKey.PubKey()

    // 2. Create a P2PKH address (mainnet)
    pubKeyHash := nogoutil.Hash160(pubKey.SerializeCompressed())
    addr, err := nogoutil.NewAddressPubKeyHash(pubKeyHash, &chaincfg.MainNetParams)
    if err != nil {
        panic(err)
    }
    fmt.Println("Address:", addr.EncodeAddress())

    // 3. Sign and verify
    hash := chainhash.DoubleHashH([]byte("NogoChain"))
    sig := privKey.Sign(hash[:])
    fmt.Println("Signature valid:", sig.Verify(hash[:], pubKey))

    // 4. Work with amounts
    amount, _ := nogoutil.NewAmount(1.5)
    fmt.Println("Amount:", amount) // 1.50000000 BTC
}
```

## Detailed Documentation

- [Package Overview & Descriptions](./docs/01-nogocommons/README.md)
- [API Reference](./docs/01-nogocommons/API-Reference.md)
- [NogoPow Consensus Algorithm](./docs/01-nogocommons/Consensus-Algorithm.md)

## Dependencies

Key dependencies include:
- `github.com/decred/dcrd/dcrec/secp256k1/v4` — secp256k1 elliptic curve
- `golang.org/x/crypto` — Salsa20, Keccak256, ChaCha20, HKDF
- `github.com/syndtr/goleveldb` — LevelDB for metadata storage
- `golang.org/x/sync` — singleflight for cache deduplication

## License

nogocommons is licensed under the [ISC License](./LICENSE).

```
Copyright (c) 2026 NogoChain Contributors

Permission to use, copy, modify, and/or distribute this software for any
purpose with or without fee is hereby granted, provided that the above
copyright notice and this permission notice appear in all copies.
```

---

# nogocommons

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

**nogocommons** 是 [NogoChain](https://github.com/nogochain) 生态系统的公共基础库。它为所有 NogoChain 子项目提供基础构建模块——从全节点、钱包到矿池和独立矿工。

## 在生态系统中的角色

```
┌──────────────────────────────────────────────────────────────┐
│                     NogoChain 生态系统                        │
├───────────┬───────────┬───────────┬───────────┬──────────────┤
│ nogocore  │nogowallet │ nogopool  │nogominer  │  (你的应用)   │
│  全节点   │  HD 钱包  │   矿池    │ 独立矿工  │              │
└─────┬─────┴─────┬─────┴─────┬─────┴─────┬─────┴──────┬───────┘
      │           │           │           │            │
      └───────────┴───────────┴───────────┴────────────┘
                         │
              ┌──────────▼──────────┐
              │    nogocommons      │
              │      公共基础库      │
              └─────────────────────┘
```

NogoChain 生态中的每个项目都依赖 **nogocommons** 提供：
- **网络有线协议** — 消息序列化与反序列化
- **共识引擎** — NogoPow 工作量证明算法
- **地址编码** — P2PKH、P2SH、P2WPKH、P2WSH、P2TR、P2A 地址
- **椭圆曲线密码学** — secp256k1 ECDSA、Schnorr、密钥管理
- **数据库抽象** — 可插拔的区块和元数据存储后端
- **P2P 网络** — 节点管理、连接处理、地址发现
- **链配置** — 主网、测试网、回归测试网、签名网、模拟网的网络参数

## 子包

| 包 | 描述 |
|---------|-------------|
| **[address](./address/)** | NogoChain 地址编码与解码。支持 P2PKH、P2SH、P2WPKH、P2WSH、P2TR（Taproot）、P2A（Pay-to-Anchor）和原始公钥地址。包含 base58 和 bech32/bech32m 子包。 |
| **[addrmgr](./addrmgr/)** | 并发安全的地址管理器，用于节点发现。维护一个加密随机化的节点地址池，具有分组隔离、可路由性感知、Tor 支持和基于质量的偏向选择。 |
| **[chaincfg](./chaincfg/)** | MainNet、TestNet3、TestNet4、RegressionNet、SigNet 和 SimNet 的网络参数定义。包括共识规则（出块时间、最大区块大小）、地址编码前缀、BIP32 HD 密钥魔术数和 NogoPow 经济模型参数。 |
| **[chainhash](./chainhash/)** | 通用 32 字节哈希类型，支持十六进制编解码。提供 `Hash`、`DoubleHash`、标签哈希（BIP-340/BIP-341）和便捷构造函数（`NewHash`、`NewHashFromStr`）。 |
| **[connmgr](./connmgr/)** | 通用比特币网络连接管理器。维护出站连接目标、获取节点来源、处理封禁、限制最大连接数，支持 Tor 查找。 |
| **[database](./database/)** | 可插拔的区块和元数据存储数据库接口。提供 `DB`（Begin/View/Update/Close）、`Tx`（StoreBlock/FetchBlock/FetchBlockRegion/PruneBlocks）、`Bucket` 和 `Cursor` 接口。默认后端：ffldb（LevelDB 元数据 + 扁平文件）。 |
| **[limits](./limits/)** | 跨平台操作系统资源限制设置。Windows 上为空操作，Unix 上基于 `setrlimit`。 |
| **[neutrino](./neutrino/)** | 轻客户端（SPV）协议支持。包括头部优先同步、过滤器数据库（GCS）、区块通知、封禁管理器和链同步工具。 |
| **[nogoec](./nogoec/)** | secp256k1 椭圆曲线密码学。ECDSA 签名/验证、公钥恢复、Schnorr 签名、私钥管理（兼容 WIF）和基于 libsecp256k1 的域运算。 |
| **[nogopow](./nogopow/)** | **NogoPow 共识引擎** — 抗 ASIC 工作量证明算法。五阶段流水线：Nonce → Salsa20/8 → 类 Scrypt Smix → 矩阵乘法（256×256）→ FNV 哈希归约 → Keccak256。使用 PI 控制器进行难度调整，结合双指数平滑。 |
| **[nogoutil](./nogoutil/)** | 高级工具函数和类型。重新导出地址类型，提供 `Amount`（satoshi 算术）、`WIF`（钱包导入格式）、`Block`/`Tx` 记忆化包装器、布隆过滤器、HD 密钥链、GCS 过滤器和应用数据目录解析。 |
| **[ossec](./ossec/)** | OpenBSD 安全特性桩。为非 OpenBSD 平台提供 `Unveil` 和 `Pledge` 的空操作实现。 |
| **[peer](./peer/)** | 并发安全的比特币节点管理。处理版本协商、全双工消息 I/O、库存滴漏、保活 ping 和基于回调的消息分发，支持所有有线协议消息。 |
| **[txscript](./txscript/)** | 交易脚本兼容性桩。提供 `ScriptClass`、`SigHashType`、`MultiPrevOutFetcher` 和 `SecretsSource` 接口，用于与 nogocore 中的 UTXO 引擎集成。 |
| **[v2transport](./v2transport/)** | BIP324 v2 加密 P2P 传输协议。实现基于 ElligatorSwift 的握手、ChaCha20-Poly1305 加密和垃圾终止符交换，用于加密节点通信。 |
| **[wire](./wire/)** | 比特币有线协议实现。支持所有 P2P 消息类型（MsgVersion、MsgBlock、MsgTx、MsgInv、MsgGetData 等）、消息序列化/反序列化、服务标志和网络魔术数。 |

## 快速开始

```go
package main

import (
    "fmt"
    "github.com/nogochain/nogocommons/chaincfg"
    "github.com/nogochain/nogocommons/chainhash"
    "github.com/nogochain/nogocommons/nogoec"
    "github.com/nogochain/nogocommons/nogoutil"
)

func main() {
    // 1. 生成 secp256k1 密钥对
    privKey, err := nogoec.NewPrivateKey()
    if err != nil {
        panic(err)
    }
    pubKey := privKey.PubKey()

    // 2. 创建 P2PKH 地址（主网）
    pubKeyHash := nogoutil.Hash160(pubKey.SerializeCompressed())
    addr, err := nogoutil.NewAddressPubKeyHash(pubKeyHash, &chaincfg.MainNetParams)
    if err != nil {
        panic(err)
    }
    fmt.Println("地址:", addr.EncodeAddress())

    // 3. 签名和验证
    hash := chainhash.DoubleHashH([]byte("NogoChain"))
    sig := privKey.Sign(hash[:])
    fmt.Println("签名有效:", sig.Verify(hash[:], pubKey))

    // 4. 金额操作
    amount, _ := nogoutil.NewAmount(1.5)
    fmt.Println("金额:", amount) // 1.50000000 BTC
}
```

## 详细文档

- [包概览与描述](./docs/01-nogocommons/README.md)
- [API 参考](./docs/01-nogocommons/API-Reference.md)
- [NogoPow 共识算法](./docs/01-nogocommons/Consensus-Algorithm.md)

## 依赖

关键依赖包括：
- `github.com/decred/dcrd/dcrec/secp256k1/v4` — secp256k1 椭圆曲线
- `golang.org/x/crypto` — Salsa20、Keccak256、ChaCha20、HKDF
- `github.com/syndtr/goleveldb` — 用于元数据存储的 LevelDB
- `golang.org/x/sync` — 用于缓存去重的 singleflight

## 许可证

nogocommons 基于 [ISC 许可证](./LICENSE) 授权。

```
Copyright (c) 2026 NogoChain Contributors

特此免费授予任何获得本软件及相关文档文件副本的人不受限制地处理本软件的权限，
包括但不限于使用、复制、修改、合并、发布、分发、再许可和/或销售本软件副本的权利，
并允许获得本软件的人这样做，但须满足上述版权声明和本许可声明出现在所有副本中。
```
