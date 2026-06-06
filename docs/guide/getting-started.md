# 快速开始

## 系统要求

- 操作系统：macOS、Linux 或 Windows
- 架构：amd64 或 arm64
- 无需额外依赖（SQLite 已内嵌）

## 方式一：下载预编译二进制（推荐）

1. 前往 [Releases](https://codeberg.org/wsh233/golog/releases) 页面
2. 根据你的操作系统和架构下载对应的压缩包
3. 解压后得到 `golog`（或 Windows 下的 `golog.exe`）
4. 赋予执行权限（Linux/macOS）：

   ```bash
   chmod +x golog
   ```

## 方式二：从源码编译

### 前置要求

- [Go](https://go.dev/dl/) 1.25 或更高版本
- Make（可选，用于交叉编译）

### 编译步骤

```bash
# 克隆仓库
git clone https://github.com/giserlab/golog.git
cd golog

# 直接编译
go build -o golog main.go

# 或使用 Makefile 交叉编译（CGO_ENABLED=0）
make build
```

编译产物位于 `bin/` 目录下。

## 运行

### 基本启动

```bash
./golog
```

默认监听端口为 `5201`，启动后在浏览器中访问：

```
http://localhost:5201
```

### 首次启动

如果目录下没有 `config.json`，应用会自动重定向到初始化向导（`/wizard`），按提示完成：

1. 设置站点名称和描述
2. 创建管理员账号
3. 配置基本选项

### 命令行参数

```bash
# 指定端口
./golog --port 8080

# 启用 TLS
./golog --tls-crt server.crt --tls-key server.key

# 重置用户密码
./golog reset-password user@example.com

# 数据库迁移
./golog db:migrate        # 迁移到最新版本
./golog db:migrate 5      # 迁移到指定版本
```

### 常用选项

| 参数        | 简写 | 说明         | 默认值 |
| ----------- | ---- | ------------ | ------ |
| `--port`    | `-p` | 监听端口     | `5201` |
| `--tls-crt` | -    | TLS 证书路径 | -      |
| `--tls-key` | -    | TLS 私钥路径 | -      |

## 目录结构说明

首次运行后会自动生成以下目录和文件：

```
golog/
├── config.json          # 站点配置文件
├── db.sqlite            # SQLite 数据库
├── data/
│   └── uploads/
│       ├── covers/      # 文章封面图
│       └── images/      # 文章内图片
└── ...
```

## 下一步

- 访问管理后台撰写你的第一篇文章
- 在设置中切换主题和个性化选项
- 阅读 [功能特性](./features) 了解全部能力
