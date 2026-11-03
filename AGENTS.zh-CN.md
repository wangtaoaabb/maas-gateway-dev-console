# MaaS Gateway Dev Console — 中文维护指南

这份文档面向项目维护者，说明代码归属、配置流程、界面约定和验证方法，
与供编码智能体读取的 [AGENTS.md](AGENTS.md) 对应。
日常启动和使用说明见 [README.md](README.md)。两份维护指南需要同步更新。

## 1. 项目范围与仓库边界

本项目是独立的本地 Web 控制台，包含可配置的 HTTP/SSE Mock 服务和 Gateway 进程管理器。

- 本目录拥有独立 Git 仓库。提交前在本目录执行 `git rev-parse --show-toplevel`，
  确认仓库根目录正确，避免把旁边的其他项目一起暂存。
- `../maas-gateway-local-mock` 负责 Gateway 黑盒 E2E 测试，不在本项目中修改它。
- 不修改、重置或替换配置中指定的 Gateway 源码仓库。
- 正常使用时，需要持久保存的配置放在 `profiles/`；生成的会话文件放在 `.runtime/`。
  `.runtime/`、`.idea/`、日志和 `.DS_Store` 已被 Git 忽略。
- 优先使用 Go 标准库和原生 HTML/CSS/JavaScript。Node.js 仅用于可选的前端测试，
  运行应用本身不需要 Node.js。
- 遵守父工作区的修改授权约定：排查、解释默认只读；用户授权具体实现后再修改。

## 2. 代码在哪里

| 内容 | 对应位置 |
| --- | --- |
| 命令行入口、选择 profile、控制台生命周期 | `cmd/dev-console/main.go` |
| 配置结构、默认值、校验、保存、源码路径变量展开 | `internal/config/` |
| Mock 匹配、响应生成、请求记录 | `internal/mock/` |
| 生成 Gateway YAML 配置 | `internal/runtime/gateway_config.go` |
| Gateway/Delve 生命周期、进程组和日志采集 | `internal/process/` |
| 本地 API 与嵌入程序的前端资源 | `internal/web/` |
| 前端行为回归检查 | `scripts/ui.test.cjs` |

排查时先看负责该行为的模块及直接调用者，避免一开始就扫描整个仓库。

## 3. 配置从哪里来，启动时怎么使用

正常开发时，从本项目根目录启动控制台。当前工作目录决定 `.runtime/` 的位置，
相对 profile 路径也从当前工作目录解析。

配置的主要流程是：

```text
启动时选择 profile → 页面编辑并保存 JSON → 启动 Gateway 时生成 YAML → 传给 Gateway 进程
```

- 默认使用 `profiles/default.json`；通过 `-profile <path>` 可以指定其他文件。
  如果指定文件不存在，会用默认值创建。页面保存的是当前选中的那份 profile。
- Gateway 源码目录默认保存为 `${MAAS_GATEWAY_DIR}`。只在启动 Gateway 时，
  从控制台进程的环境变量中展开；未设置或值为空时，使用 `$HOME/GolandProjects/maas-gateway`。
- profile 和页面中保留变量表达式，不把展开后的本机绝对路径写回。
  也支持直接填路径或 `${HOME}` 等其他变量；除 `MAAS_GATEWAY_DIR` 的默认规则外，
  未设置的变量应明确报错，不能自行猜测路径。
- 每次启动 Gateway 都生成 `.runtime/current/gateway-config/config.<ENV>.yml`，
  并通过子进程环境变量 `ENV`、`CONF_PATH` 指定加载位置。
  不隐式读取或合并 Gateway 源码仓库中的原有 YAML。
- **Additional Gateway YAML** 按文本追加，不做配置树的深度合并。
  生成的 YAML 是运行产物，下次启动会覆盖，不应当作长期配置直接维护。
- 修改 Gateway 配置后点击“应用并重启”；保存 Mock 路由后，后续请求立即生效。
  修改监听地址或控制台进程的环境变量，需要重启控制台。
- 控制台、Mock 和 Delve 默认只监听回环地址。
  停止 Gateway 时要终止受管理的整个进程组，包括 Go/Delve 启动的后代进程。

## 4. 一个接口，多个匹配场景

配置文件仍使用扁平的 `mock.routes` 数组。页面按 **Method + Path** 把它们归为同一个接口；
数组中的每条路由仍是有独立 ID 的匹配/响应场景，没有额外的分组存储结构。

- 只匹配启用的场景。Priority 从高到低，相同优先级保持 profile 中的顺序。
- 请求头、查询参数、JSON 字段的条件按 AND 组合；第一条命中后就返回响应。
- 空匹配条件相当于兜底，应该放在较低优先级。
- 未配置的接口必须返回 **HTTP 501**，请求记录中的 `matched=false`。
  不允许用通用成功响应掩盖缺失的依赖接口。
- “新增场景”和“复制当前场景”创建的分支默认禁用，配置好后再勾选 **Enabled** 并保存。
  “新增接口”的初始路由默认启用，但仍需保存后才生效。
- 接口说明展示在左侧接口下方；页面保存说明时，会同步给相同 Method + Path 的场景。
- 切换场景时，合法编辑保留为浏览器内的草稿；通过明确的保存操作才持久化。
  保留未保存提示，避免无效 JSON 或重复 ID 静默覆盖其他场景。

SSE 沿用 `response.sse` 事件数组，每条事件保存 `data` 和 `delay_ms`。
普通响应、SSE 响应、纯文本 data 和 `[DONE]` 都要保持兼容。
`close_after_events=0` 表示发送全部事件，正数用于模拟提前断流。

事件卡片支持新增、删除、排序、JSON 格式化和逐事件延迟。
未编辑的原始 data 应当保持不变，不能仅因页面展开显示 JSON 就改写保存内容。

## 5. 界面风格与交互约定

当前采用已经确认的 **EVA / NERV 作战终端风格**：

- 黑色、炭灰色作为工作区底色；橙色工业标题；紫色场景选中状态；
  荧绿用于活动状态和运行按钮；红色用于删除、停止和错误。
- 使用切角边框和适量警示斜纹。代码、日志、输入框必须清晰可读，
  不让装饰遮挡操作，保留中英双语操作提示和响应式布局。
- 用 HTML/CSS/SVG 实现界面。设计图只是参考，不能用整张图片代替实际控件。
  前端资源会嵌入 Go 程序，修改后需要重新启动或构建控制台，并刷新浏览器。
- 下拉框统一使用深色自定义菜单；原生 select 仍是表单值来源，必须与可见控件同步。
- 保留方向键导航、Enter 选择、Esc 和点击外部关闭、焦点提示以及 ARIA 状态。
  菜单根据视口调整位置；高亮选项只滚动菜单内部，不能带动整页滚动。
- 日志优先按声明的级别着色：ERROR/FATAL/PANIC 红色、WARN 橙色、INFO 绿色、
  DEBUG/TRACE 紫色。兼容 JSON 和普通文本日志，清理 ANSI 控制码。
- 用户可控内容用 `textContent` 等纯文本方式显示，不直接拼入 HTML。
- 页面最多保留 5000 条日志；查看历史日志时不强制跳到底部，回到底部后再自动跟随。

## 6. 怎么验证

常用命令：

```bash
go test ./...
go vet ./...
node --check internal/web/static/app.js
node --test scripts/ui.test.cjs
```

按改动范围选择检查。前端检查需要 Node.js，但不会增加应用运行依赖。
只有文档修改时，核对链接、路径和内容一致性即可，不需要重跑无关的应用测试。

- Mock 行为使用 `httptest` 验证。
- 直接验证生成的 Gateway YAML，不修改 Gateway 仓库里的配置样例。
- 进程参数的单元测试不能启动真实 Gateway。
- 涉及 Mock 行为时，按需冒烟验证 `/healthz`、控制面 JSON、以 `[DONE]` 结束的流式响应、
  场景优先级与兜底，以及未配置接口的 HTTP 501。
- 界面改动检查场景切换和保存、SSE 编辑、自定义菜单键盘交互、代码和日志可读性、
  窄屏是否溢出。交付实际浏览器截图，不把生成的设计图当成已经实现的页面。
- 验证使用临时 profile 和独立回环端口，不覆盖用户正在使用的配置，不停止已有服务；
  只清理自己启动的测试进程。
- 请求体、请求头/API Key、profile 和 Gateway 日志都属于敏感本地调试数据，
  不提交真实凭据或采集的会话数据。

## 7. 文档维护

项目边界、配置流程、匹配行为、界面约定或验证命令变更时，同步更新本文件和 `AGENTS.md`。
`README.md` 负责安装、使用与排错；两份维护指南负责协作和维护约定。
写文档时明确区分已经实现的行为和待讨论的方案。
