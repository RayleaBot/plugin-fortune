# RayleaBot Fortune Plugin

`raylea.fortune` 提供每日运势命令与 Vue 管理页面。插件后端、静态管理页面和数据文件共同进入独立 artifact，生产包不随 RayleaBot 主程序打包。

## 本地联调

将本仓库路径写入 RayleaBot 根目录下被忽略的 `plugin-workspace.local.json`，并运行：

```powershell
$env:RAYLEA_PLUGIN_DEV = 'watch'
$env:RAYLEA_SERVER_RELOAD = 'watch'
node scripts/start-dev.mjs
```

启动脚本会把主仓库的 Go 与 Vue SDK 映射到 `.rayleabot/`，构建当前平台 artifact，再通过离线 `plugin dev-sync` 原子同步到 `plugins/installed/`。`.rayleabot/` 只属于本地开发，不进入提交或发布包。

## 发布

推送 `v*` 标签后，工作流使用固定 SDK 引用测试 Go 与 Vue 代码，构建 Windows x64、Linux x64 和 macOS arm64 ZIP，并创建 GitHub Release。插件目录仓库随后记录产物摘要并发布签名目录。

License: MIT
