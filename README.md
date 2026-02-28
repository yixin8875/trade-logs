# 交易日志桌面程序（Wails + Go + React + Tailwind）

将 Excel 模板交易日志迁移为本地桌面应用，支持三张表导入维护、统计分析和导出。

## 当前功能

- 支持从 Excel 文件导入三张表（默认路径：`/Users/boohee/Downloads/交易日志模板菜真寒版.xlsx`）
  - `日志`
  - `错误类型`
  - `别瞎搞日记本`
- 本地 SQLite 持久化存储（macOS 默认在 `~/Library/Application Support/trade-logs/trades.db`）
- 交易记录增删改查
- 错误类型增删改查
- 日常执行日记增删改查
- 自动统计：总盈亏、胜率、平均盈利、平均亏损、盈亏比
- 按方向和关键字筛选列表
- 导出：
  - `CSV`（交易列表）
  - `Excel(.xlsx)`（三张表）
- 兼容迁移：首次启动会自动读取旧版 `trades.json` 并迁移入 SQLite（若数据库为空）

## 开发运行

```bash
wails dev
```

## 构建

```bash
wails build
```

构建产物：

- `build/bin/trade-logs.app`
