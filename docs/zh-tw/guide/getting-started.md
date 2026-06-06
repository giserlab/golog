# 快速開始

## 系統要求

- 作業系統：macOS、Linux 或 Windows
- 架構：amd64 或 arm64
- 無需額外依賴（SQLite 已內嵌）

## 方式一：下載預編譯二進位檔案（推薦）

1. 前往 [Releases](https://github.com/WShihan/golog/releases) 頁面
2. 根據你的作業系統和架構下載對應的壓縮包
3. 解壓後得到 `golog`（或 Windows 下的 `golog.exe`）
4. 賦予執行權限（Linux/macOS）：

   ```bash
   chmod +x golog
   ```

## 方式二：從原始碼編譯

### 前置要求

- [Go](https://go.dev/dl/) 1.25 或更高版本
- Make（可選，用於交叉編譯）

### 編譯步驟

```bash
# 複製倉庫
git clone https://github.com/WShihan/golog.git
cd golog

# 直接編譯
go build -o golog main.go

# 或使用 Makefile 交叉編譯（CGO_ENABLED=0）
make build
```

編譯產物位於 `bin/` 目錄下。

## 執行

### 基本啟動

```bash
./golog
```

預設監聽連接埠為 `5201`，啟動後在瀏覽器中存取：

```
http://localhost:5201
```

### 首次啟動

如果目錄下沒有 `config.json`，應用會自動重新導向到初始化精靈（`/wizard`），按提示完成：

1. 設定站點名稱和描述
2. 建立管理員帳號
3. 設定基本選項

### 命令列參數

```bash
# 指定連接埠
./golog --port 8080

# 啟用 TLS
./golog --tls-crt server.crt --tls-key server.key

# 重設使用者密碼
./golog reset-password user@example.com

# 資料庫遷移
./golog db:migrate        # 遷移到最新版本
./golog db:migrate 5      # 遷移到指定版本
```

### 常用選項

| 參數        | 簡寫 | 說明         | 預設值 |
| ----------- | ---- | ------------ | ------ |
| `--port`    | `-p` | 監聽連接埠   | `5201` |
| `--tls-crt` | -    | TLS 憑證路徑 | -      |
| `--tls-key` | -    | TLS 私鑰路徑 | -      |

## 目錄結構說明

首次執行後會自動生成以下目錄和檔案：

```
golog/
├── config.json          # 站點設定檔案
├── db.sqlite            # SQLite 資料庫
├── data/
│   └── uploads/
│       ├── covers/      # 文章封面圖
│       └── images/      # 文章內圖片
└── ...
```

## 下一步

- 存取管理後台撰寫你的第一篇文章
- 在設定中切換主題和個人化選項
- 閱讀 [功能特性](./features) 瞭解全部能力
