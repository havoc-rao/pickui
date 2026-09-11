# picktui 文档

两类读者，两套入口：

## 开发（docs/dev/）— 面向 picktui 仓库维护者/贡献者

| 文档 | 内容 |
|---|---|
| [plan.md](dev/plan.md) | 抽取计划与架构：M1 Go 引擎 / M2 TS 绑定、仓库布局、泛化点、shr 改造步骤、发布节奏 |

## 接入（docs/integration/）— 面向使用 picktui 的宿主项目

| 文档 | 内容 |
|---|---|
| [protocol.md](integration/protocol.md) | 引擎协议 v1（唯一契约）：子命令、JSON 字段、退出码 0/1/2/130、数据/错误约定；脚本与各语言绑定共同遵守 |
| [ts.md](integration/ts.md) | TS/JS 接入指南（npm 包 @havocrao/picktui）：安装、用法、关键语义 |

> Go 宿主无需单独指南：引擎即库，API 与嵌入方式见 [go/README.md](../go/README.md)。
> 仓库级快速开始与按需食用矩阵见根 [README.md](../README.md)。