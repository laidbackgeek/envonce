package i18n

// Message-key constants. Each catalog entry needs both ZH and EN values; keys with %s/%d/%v support fmt.Sprintf.
const (
	// —— command Short/Long (command descriptions shown in --help, translated at runtime by applyI18n) ——
	RootShort        = "root_short"
	RootLong         = "root_long"
	EnvShort         = "env_short"
	ShellInitShort   = "shellinit_short"
	ShellInitLong    = "shellinit_long"
	InitShort        = "init_short"
	DoctorShort      = "doctor_short"
	ServiceShort     = "service_short"
	GroupShort       = "group_short"
	SvcAddShort      = "svc_add_short"
	SvcStartShort    = "svc_start_short"
	SvcStopShort     = "svc_stop_short"
	SvcRestartShort  = "svc_restart_short"
	SvcSyncShort     = "svc_sync_short"
	SvcStatusShort   = "svc_status_short"
	SvcListShort     = "svc_list_short"
	SvcTakeShort     = "svc_take_short"
	SvcDropShort     = "svc_drop_short"
	GroupCreateShort = "group_create_short"
	GroupListShort   = "group_list_short"
	GroupRenameShort = "group_rename_short"
	GroupDeleteShort = "group_delete_short"

	// —— common ——
	MsgWroteEnv = "wrote_env"

	// —— runtime operation summaries (✓ what it did / next step / rollback / related) ——
	// env
	SummaryEnvSetNext     = "summary_env_set_next"
	SummaryEnvSetRollback = "summary_env_set_rollback"
	SummaryEnvUnsetDone   = "summary_env_unset_done"
	SummaryEnvListRelated = "summary_env_list_related"
	// service
	SummarySvcAddDone       = "summary_svc_add_done"
	SummarySvcStartDone     = "summary_svc_start_done"
	SummarySvcStopDone      = "summary_svc_stop_done"
	SummarySvcRestartDone   = "summary_svc_restart_done"
	SummarySvcSyncDone      = "summary_svc_sync_done"
	SummarySvcStatusRelated = "summary_svc_status_related"
	SummarySvcListRelated   = "summary_svc_list_related"
	// take / drop
	SummaryTakeDone        = "summary_take_done"
	SummaryTakeNextRoll    = "summary_take_next_roll"
	SummaryTakeMigrated    = "summary_take_migrated"
	SummaryDropDone        = "summary_drop_done"
	SummaryDropRestoreBrew = "summary_drop_restore_brew"
	// group
	SummaryGroupCreateDone  = "summary_group_create_done"
	SummaryGroupListRelated = "summary_group_list_related"
	SummaryGroupRenameDone  = "summary_group_rename_done"
	SummaryGroupDeleteDone  = "summary_group_delete_done"
	// init
	SummaryInitDir      = "summary_init_dir"
	SummaryInitFiles    = "summary_init_files"
	SummaryInitShell    = "summary_init_shell"
	SummaryInitMarker   = "summary_init_marker"
	SummaryInitNextRoll = "summary_init_next_roll"
	// doctor
	DoctorMsgNoConfigDir        = "doctor_msg_no_config_dir"
	DoctorMsgNotInit            = "doctor_msg_not_init"
	DoctorMsgBrewMissing        = "doctor_msg_brew_missing"
	DoctorMsgSecurityMissing    = "doctor_msg_security_missing"
	DoctorMsgBrewPlistCollision = "doctor_msg_brew_plist_collision"
	DoctorMsgXDGDrift           = "doctor_msg_xdg_drift"

	// —— status labels (service status / list) ——
	StatusRunning          = "status_running"           // %d (pid)
	StatusNotLoaded        = "status_not_loaded"
	StatusLoadedNotRunning = "status_loaded_not_running"
	StatusCrashed          = "status_crashed" // %s exit code, %d runs, %s service name
	EnvParseWarn           = "env_parse_warn" // %v
	EnvParseOK             = "env_parse_ok"

	// —— cli error messages ——
	ErrUnknownService  = "err_unknown_service" // %s
	ErrGroupExists     = "err_group_exists"    // %s
	ErrInvalidKV       = "err_invalid_kv"
	ErrNotFound        = "err_not_found"         // %s
	ErrBrewStopFailed  = "err_brew_stop_failed"  // %v
	ErrBrewStartFailed = "err_brew_start_failed" // %s %v

	// —— first-run banner ——
	BannerNotInit = "banner_not_init"

	// —— flag descriptions (each flag's text in --help) ——
	FlagLang        = "flag_lang"
	FlagVersion     = "flag_version"
	FlagUninstall   = "flag_uninstall"
	FlagGroup       = "flag_group"
	FlagGroups      = "flag_groups"
	FlagService     = "flag_service"
	FlagBinary      = "flag_binary"
	FlagKeepAlive   = "flag_keep_alive"
	FlagRunAtLoad   = "flag_run_at_load"
	FlagRestoreBrew = "flag_restore_brew"
)

// catalog maps each message key to its localized ZH and EN text.
var catalog = map[string]map[Lang]string{
	RootShort:        {ZH: "统一管理 shell 与 launchd 服务的环境变量", EN: "Unified env var management for shell and launchd services"},
	RootLong:         {ZH: "envonce 把环境变量从 launchd plist / brew wrapper 里解放出来，集中到 env.d/*.env 分组管理，shell 与服务共享同一份配置。", EN: "envonce frees env vars from launchd plists and brew wrappers, centralizing them into env.d/*.env groups shared by shell and services."},
	EnvShort:         {ZH: "管理环境变量", EN: "Manage environment variables"},
	ShellInitShort:   {ZH: "输出 shell 应加载的 export 语句（由 init 自动接入）", EN: "Print export lines for the shell (auto-wired by init)"},
	ShellInitLong:    {ZH: "用途：输出当前 shell 应加载的 export 语句（运行时解析，含 KeyChain）。\n\n通常由 `envonce init` 自动接入到 ~/.zshrc；手动用于预览/排查/非标准 shell 配置。", EN: "Prints export lines for the shell (runtime-resolved, incl. KeyChain).\n\nUsually auto-wired into ~/.zshrc by `envonce init`; run manually to preview/debug or for non-standard shells."},
	InitShort:        {ZH: "创建配置目录并接入 shell", EN: "Create config directories and wire into shell"},
	DoctorShort:      {ZH: "环境自检", EN: "Run environment health checks"},
	ServiceShort:     {ZH: "管理后台服务", EN: "Manage background services"},
	GroupShort:       {ZH: "管理 env 分组", EN: "Manage env groups"},
	SvcAddShort:      {ZH: "添加并启动一个后台服务", EN: "Add and start a background service"},
	SvcStartShort:    {ZH: "启动（已添加的）服务", EN: "Start a registered service"},
	SvcStopShort:     {ZH: "停止服务（bootout）", EN: "Stop a service (launchd bootout)"},
	SvcRestartShort:  {ZH: "重启服务（bootout 后重新 bootstrap）", EN: "Restart a service (bootout then bootstrap)"},
	SvcSyncShort:     {ZH: "重生 wrapper 与 plist（不重启服务）", EN: "Regenerate wrapper and plist (no restart)"},
	SvcStatusShort:   {ZH: "查看服务状态（加载状态 + env 健康检查）", EN: "Show service status (load state + env health)"},
	SvcListShort:     {ZH: "列出全部已注册服务及加载状态", EN: "List all registered services and load state"},
	SvcTakeShort:     {ZH: "从 brew 接管服务（导入 + 停 brew + 启动 envonce 管理）", EN: "Take over a brew service (import, stop brew, start under envonce)"},
	SvcDropShort:     {ZH: "卸载并移除 envonce 接管的服务", EN: "Unload and remove an envonce-managed service"},
	GroupCreateShort: {ZH: "创建一个空 env 分组", EN: "Create an empty env group"},
	GroupListShort:   {ZH: "列出全部 env 分组", EN: "List all env groups"},
	GroupRenameShort: {ZH: "重命名 env 分组", EN: "Rename an env group"},
	GroupDeleteShort: {ZH: "删除 env 分组（连同其中的变量）", EN: "Delete an env group (and its variables)"},

	MsgWroteEnv: {ZH: "已写入 %s: %s", EN: "Written to %s: %s"},

	// —— env ——
	SummaryEnvSetNext:     {ZH: "下一步：envonce env list --group %s 查看分组内容", EN: "Next: envonce env list --group %s to view group contents"},
	SummaryEnvSetRollback: {ZH: "回滚：envonce env unset %s --group %s", EN: "Rollback: envonce env unset %s --group %s"},
	SummaryEnvUnsetDone:   {ZH: "✓ 已移除 %s（组 %s）\n下一步：envonce env list --group %s 查看分组内容\n回滚：envonce env set %s=... --group %s 重新写入", EN: "✓ Removed %s (group %s)\nNext: envonce env list --group %s to view group contents\nRollback: envonce env set %s=... --group %s to rewrite"},
	SummaryEnvListRelated: {ZH: "相关命令：envonce env set KEY=VALUE --group %s | envonce group list", EN: "Related: envonce env set KEY=VALUE --group %s | envonce group list"},

	// —— service ——
	SummarySvcAddDone:       {ZH: "✓ 已添加并启动服务 %s\n下一步：envonce service status %s\n回滚：envonce service drop %s", EN: "✓ Added and started service %s\nNext: envonce service status %s\nRollback: envonce service drop %s"},
	SummarySvcStartDone:     {ZH: "✓ 已启动 %s\n下一步：envonce service status %s\n回滚：envonce service stop %s", EN: "✓ Started %s\nNext: envonce service status %s\nRollback: envonce service stop %s"},
	SummarySvcStopDone:      {ZH: "✓ 已停止 %s\n下一步：envonce service start %s 重新启动\n回滚：envonce service start %s", EN: "✓ Stopped %s\nNext: envonce service start %s to restart\nRollback: envonce service start %s"},
	SummarySvcRestartDone:   {ZH: "✓ 已重启 %s\n下一步：envonce service status %s 查看状态\n回滚：envonce service stop %s", EN: "✓ Restarted %s\nNext: envonce service status %s to view status\nRollback: envonce service stop %s"},
	SummarySvcSyncDone:      {ZH: "✓ 已重生 %s 的 plist+wrapper（未重启）\n下一步：envonce service restart %s 让改动生效\n回滚：envonce service sync %s 再次重生（基于最新 config.toml）", EN: "✓ Regenerated plist+wrapper for %s (not restarted)\nNext: envonce service restart %s to apply changes\nRollback: envonce service sync %s to regenerate (based on latest config.toml)"},
	SummarySvcStatusRelated: {ZH: "相关命令：envonce service restart %s | envonce env export --service %s", EN: "Related: envonce service restart %s | envonce env export --service %s"},
	SummarySvcListRelated:   {ZH: "相关命令：envonce service status <NAME> | envonce service add <NAME>", EN: "Related: envonce service status <NAME> | envonce service add <NAME>"},

	// —— take / drop ——
	SummaryTakeDone:        {ZH: "✓ 已接管 %s（brew services stop，生成 com.envonce.%s 并启动）", EN: "✓ Took over %s (brew services stop, generated com.envonce.%s and started)"},
	SummaryTakeNextRoll:    {ZH: "下一步：envonce service status %s\n回滚：envonce service drop %s --restore-brew", EN: "Next: envonce service status %s\nRollback: envonce service drop %s --restore-brew"},
	SummaryTakeMigrated:    {ZH: "✓ 从 brew plist 迁移 %d 个环境变量到组 %s", EN: "✓ Migrated %d env vars from brew plist to group %s"},
	SummaryDropDone:        {ZH: "✓ 已移除 %s（卸载 plist+wrapper）", EN: "✓ Removed %s (unloaded plist+wrapper)"},
	SummaryDropRestoreBrew: {ZH: "✓ 已交还 brew 管理（brew services start %s）", EN: "✓ Handed back to brew (ran brew services start %s)"},

	// —— group ——
	SummaryGroupCreateDone:  {ZH: "✓ 创建组 %s（%s）\n下一步：envonce env set KEY=VALUE --group %s\n回滚：envonce group delete %s", EN: "✓ Created group %s (%s)\nNext: envonce env set KEY=VALUE --group %s\nRollback: envonce group delete %s"},
	SummaryGroupListRelated: {ZH: "相关命令：envonce env list --group <NAME> | envonce group create <NAME>", EN: "Related: envonce env list --group <NAME> | envonce group create <NAME>"},
	SummaryGroupRenameDone:  {ZH: "✓ 重命名 %s → %s\n下一步：如配置/服务引用了旧组名，请改用 %s\n回滚：envonce group rename %s %s", EN: "✓ Renamed %s → %s\nNext: if configs/services reference the old group name, use %s instead\nRollback: envonce group rename %s %s"},
	SummaryGroupDeleteDone:  {ZH: "✓ 删除组 %s（%s）\n下一步：envonce group create %s 重建分组\n回滚：需手动重建（文件已删除）", EN: "✓ Deleted group %s (%s)\nNext: envonce group create %s to rebuild the group\nRollback: must rebuild manually (file deleted)"},

	// —— init ——
	SummaryInitDir:      {ZH: "✓ 创建目录 %s（env.d/ services/ logs/ state/）", EN: "✓ Created directory %s (env.d/ services/ logs/ state/)"},
	SummaryInitFiles:    {ZH: "✓ 创建 env.d/default.env、config.toml", EN: "✓ Created env.d/default.env, config.toml"},
	SummaryInitShell:    {ZH: "✓ 已向 shell rc 追加集成行（marker: %s）", EN: "✓ Appended integration lines to shell rc (marker: %s)"},
	SummaryInitMarker:   {ZH: "✓ 写入初始化标记 .initialized", EN: "✓ Wrote initialization marker .initialized"},
	SummaryInitNextRoll: {ZH: "下一步：\n  envonce env set OLLAMA_MODELS=/Volumes/SSD/ollama/models\n  envonce service take ollama\n\n回滚：envonce init --uninstall", EN: "Next:\n  envonce env set OLLAMA_MODELS=/Volumes/SSD/ollama/models\n  envonce service take ollama\n\nRollback: envonce init --uninstall"},

	// —— doctor ——
	DoctorMsgNoConfigDir:        {ZH: "无法定位配置目录", EN: "cannot locate config directory"},
	DoctorMsgNotInit:            {ZH: "未运行 envonce init", EN: "envonce init not run"},
	DoctorMsgBrewMissing:        {ZH: "brew 不在 PATH", EN: "brew not in PATH"},
	DoctorMsgSecurityMissing:    {ZH: "/usr/bin/security 缺失", EN: "/usr/bin/security missing"},
	DoctorMsgBrewPlistCollision: {ZH: "存在 brew 残留 plist：%s", EN: "brew leftover plist present: %s"},
	DoctorMsgXDGDrift:           {ZH: "XDG 配置目录漂移：.initialized 记录 %s，当前 %s", EN: "XDG config dir drift: .initialized recorded %s, current %s"},

	// —— status labels (service status / list) ——
	StatusRunning:          {ZH: "运行中 (pid=%d)", EN: "Running (pid=%d)"},
	StatusNotLoaded:        {ZH: "未加载", EN: "Not loaded"},
	StatusLoadedNotRunning: {ZH: "已加载但未运行", EN: "Loaded, not running"},
	StatusCrashed:          {ZH: "⚠ 进程已退出 (exit code=%s)，launchd 已重启 %d 次——疑似 crash-loop，见 logs/%s.err.log", EN: "⚠ process exited (code=%s), restarted %d time(s) — likely crash-loop, see logs/%s.err.log"},
	EnvParseWarn:           {ZH: "⚠ env 解析告警: %v", EN: "⚠ env parse warning: %v"},
	EnvParseOK:             {ZH: "✓ env 解析正常", EN: "✓ env parse OK"},

	// —— cli error messages ——
	ErrUnknownService:  {ZH: "未知服务 %s", EN: "Unknown service: %s"},
	ErrGroupExists:     {ZH: "组 %s 已存在", EN: "Group %s already exists"},
	ErrInvalidKV:       {ZH: "参数须为 KEY=VALUE", EN: "Argument must be KEY=VALUE"},
	ErrNotFound:        {ZH: "未找到 %s", EN: "Not found: %s"},
	ErrBrewStopFailed:  {ZH: "⚠ brew services stop 失败（可忽略）: %v", EN: "⚠ brew services stop failed (ignorable): %v"},
	ErrBrewStartFailed: {ZH: "⚠ brew services start 失败（可手动 `brew services start %s`）: %v", EN: "⚠ brew services start failed (run `brew services start %s` manually): %v"},

	// —— first-run banner ——
	BannerNotInit: {ZH: "envonce 尚未初始化 — 运行 `envonce init` 完成 shell 集成与目录创建", EN: "envonce is not initialized — run `envonce init` to set up shell integration and directories"},

	// —— flag descriptions ——
	FlagLang:        {ZH: "界面语言 zh|en（默认按系统自动检测）", EN: "UI language zh|en (auto-detect by default)"},
	FlagVersion:     {ZH: "打印版本号并退出", EN: "Print version and exit"},
	FlagUninstall:   {ZH: "移除 shell 集成（保留配置与数据）", EN: "Remove shell integration (keep config and data)"},
	FlagGroup:       {ZH: "写入的组（默认 default）", EN: "Target group (default: default)"},
	FlagGroups:      {ZH: "显式组列表（逗号分隔）", EN: "Explicit group list (comma-separated)"},
	FlagService:     {ZH: "按服务解析其组", EN: "Resolve groups for the given service"},
	FlagBinary:      {ZH: "服务二进制绝对路径", EN: "Absolute path to the service binary"},
	FlagKeepAlive:   {ZH: "崩溃后自动重启（KeepAlive）", EN: "Auto-restart on crash (launchd KeepAlive)"},
	FlagRunAtLoad:   {ZH: "加载时立即启动（RunAtLoad）", EN: "Start immediately on load (launchd RunAtLoad)"},
	FlagRestoreBrew: {ZH: "交还 brew 管理（执行 brew services start）", EN: "Hand back to brew (runs brew services start)"},
}
