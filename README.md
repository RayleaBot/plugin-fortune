# 运势

RayleaBot 官方插件 · `raylea.fortune`

每天给每位用户抽一次运势：运势名、星级、签文、解签，以及今日宜忌。同一天重复查询会拿到同一张结果，并累计查看天数和运势分布。

## 功能

- 按用户、按自然日抽取今日运势，重复查询返回当日原结果
- 发送运势卡片图；图片渲染不可用时回退为文字
- 查看个人运势统计：各签次数、占比、连续查看天数、最长连续大吉 / 大凶
- 在插件管理页自定义触发词、时区、运势库、特殊日期和宜忌池
- 当日结果和统计始终以当前运势库、特殊日期和宜忌为准

## 安装

本插件独立发布，不随 RayleaBot 主程序打包。安装后默认停用，需要在插件列表里**启用**才会响应命令。运势卡片依赖 RayleaBot 的图片渲染环境。

### 插件商店

1. 打开 Web 管理面，进入 [插件商店](https://github.com/RayleaBot/RayleaBot/blob/main/docs/user/management-surface.md)（`/plugins/store`）。
2. 找到 **运势**，安装与当前系统匹配的版本。
3. 安装前确认：插件会作为本机原生程序运行。
4. 到 **插件列表**（`/plugins`）启用 `raylea.fortune`。

### 本地安装包

也可以在插件列表中安装本仓库 [GitHub Release](https://github.com/RayleaBot/plugin-fortune/releases) 里对应平台的 ZIP：

| 平台 | 资源 |
| --- | --- |
| Windows x64 | `windows-x64` |
| Linux x64 | `linux-x64` |
| macOS arm64 | `macos-arm64` |

## 使用方法

命令前缀以管理面 **插件设置** 为准，下面按默认前缀 `/` 和默认触发词书写。聊天命令使用管理页中的当前触发词。

| 默认命令 | 权限 | 说明 |
| --- | --- | --- |
| `/我的运势` | 所有人 | 抽取或查看今日运势 |
| `/运势统计` | 所有人 | 查看个人运势统计 |

群聊和私聊都可以用。

```text
你：/我的运势
机器人：（运势卡片，含运势名、星级、签文、解签、今日宜忌、连续天数）

你：/我的运势
机器人：（同一张当日结果，并提示今日已经抽过）

你：/运势统计
机器人：（统计卡片，含各签次数、占比和连续记录）
```

还没抽过运势时，统计命令会提示先发送「我的运势」。

### 运势怎么抽

- 每位用户每天一份结果，按管理页里的**时区**切日，默认 `Asia/Shanghai`。
- 签种为：大吉、吉、中吉、小吉、末吉、凶、大凶。日常抽取不会抽到「吉凶未定」，该签留给特殊日期指定。
- 今日宜、今日忌各抽两条；忌项不会与当天的宜项重复。
- 特殊日期可写成 `05-04`（每年重复）或 `2026-05-04`（只生效一次），当天固定抽中指定签种。

## 管理页

打开 **插件列表 → 运势 → 运势设置**。保存内容即时生效。

| 分区 | 配置内容 |
| --- | --- |
| 触发词 | 运势查询词、统计查询词，可添加多个 |
| 时区 | 刷新每日运势所用的参考时区 |
| 运势库 | 运势名、星级、签文、解签；可筛选、分页、新增、复制、删除 |
| 特殊日期 | 指定日期固定抽取的签种 |
| 每日宜忌 | 「今日宜」「今日忌」候选活动 |

页脚可以重新载入、恢复默认或保存。星级由运势名决定，需要与运势名匹配才能保存。

例如，运势查询词配置为「今日运势」时，对应的聊天命令为 `/今日运势`。

## 说明

- 统计按用户累计，不按群隔离。同一人在不同群查询，用的是同一份运势和统计。
- 当日结果和统计以当前运势库、特殊日期及宜忌为准。触发词和时区不参与结果内容判定。
- 「吉凶未定」出现在统计里，是因为特殊日期指定过该签，或运势库在极端情况下只剩这一签。
- 图片发不出来时检查渲染环境（Chrome / Chromium / Edge）；插件仍会尝试发送文字版运势。

## 开发

插件后端、Vue 管理页、默认运势数据和渲染模板共同进入独立 artifact。生产包不随 RayleaBot 主程序打包。`.rayleabot/` 只属于本地开发，不进入提交或发布包。

### 目录结构

```text
plugin-fortune/
  cmd/fortune/                 进程入口
  internal/plugin/             事件处理、运势逻辑和测试
  internal/assets/fortunes.json 默认运势、触发词、宜忌和时区
  templates/                   运势卡片与统计卡片模板
  ui/                          Vue 管理页
  info.json                    manifest v3、权限、默认触发词与发布元数据
```

`fortunes.json` 编译进后端并由管理页作为完整默认运势库使用；manifest 内联默认触发词与时区。`templates/card` 与 `templates/stats` 由统一构建器自动发现并随插件包发布。

### 本地联调

1. 将本仓库路径写入 RayleaBot 根目录下被 Git 忽略的 `plugin-workspace.local.json`：

```json
{
  "workspace_version": "2",
  "plugins": [
    {
      "path": "../RayleaBotPlugins/plugin-fortune",
      "enabled": true
    }
  ]
}
```

2. 在 **RayleaBot 主仓库根目录** 启动：

```powershell
$env:RAYLEA_PLUGIN_DEV = "watch"
$env:RAYLEA_SERVER_RELOAD = "watch"
.\start.bat
```

启动器会把主仓库的 Go 与 Vue SDK 映射到 `.rayleabot/`，构建当前平台 artifact，再通过离线 `plugin dev-sync` 同步到 `plugins/installed/`。

### 测试与构建

```powershell
go test -race ./...
pnpm --dir ui install --frozen-lockfile
pnpm --dir ui typecheck
pnpm --dir ui test
pnpm --dir ui build
$env:RAYLEA_PLUGIN_BUILD_USE_WORKSPACE = "1"
go run github.com/RayleaBot/RayleaBot/sdk/go/cmd/raylea-plugin build-go --plugin . --backend ./cmd/fortune --target windows-x64 --out dist
```

### 发布

`v*` 标签对应的发布工作流使用固定 SDK 引用测试 Go 与 Vue 代码，构建 Windows x64、Linux x64 和 macOS arm64 ZIP，并创建 GitHub Release。[plugin-catalog](https://github.com/RayleaBot/plugin-catalog) 定时读取当前 Release，校验 artifact 并记录各平台 ZIP 的 SHA-256。

本地联调与商店分发说明见 [插件商店与独立开发](https://github.com/RayleaBot/RayleaBot/blob/main/docs/plugin/store-and-development.md)。

## License

[MIT](./LICENSE)
