# MaaS Gateway 本地开发控制台

`maas-gateway-dev-console` 是一个独立的本地开发控制台，用于配置 Gateway
依赖、运行本地 Mock 服务，以及通过普通模式或 Delve 调试模式启动 `maas-gateway`。

## 功能

- 启动内置 Mock 服务，提供健康检查接口。
- 在浏览器中配置每条 Mock 路由的方法、路径、请求匹配条件、状态码、响应头、
  响应体、SSE 事件、延迟，以及提前关闭流。
- 未配置的外部接口返回 HTTP 501，并在页面上突出显示对应请求。
- 记录完整的 Mock 入站请求，方便本地调试。
- 在浏览器中配置 Gateway 各个外部服务的基础地址。
- 支持追加 Gateway YAML 配置，用于尚无专用表单字段的依赖，例如链路追踪或内容审核服务。
- 在 `.runtime/` 下生成隔离的 Gateway `config.<ENV>.yml`。
- 使用普通 `go run` 模式或无界面的 Delve 模式启动、停止 Gateway。
- 展示 Gateway 状态，并在页面上增量显示进程日志。
- 将配置保存为本地 JSON profile，无需数据库或 Node.js。

## 快速开始

本项目的 Go 模块声明版本为 Go 1.22。启动实际 Gateway 时，还需要满足对应 Gateway
源码的构建要求；使用调试模式需要安装 Delve，并确保 `dlv` 可用。

```bash
cd /Users/taowang/Documents/goProject/maas-gateway-dev-console
go run ./cmd/dev-console
```

打开 <http://127.0.0.1:19080>。

默认 profile 在 <http://127.0.0.1:19090> 启动内置 Mock，
并将 Gateway 各外部服务地址指向它。Gateway 源码目录在页面和 profile 中默认保存为
`${MAAS_GATEWAY_DIR}`，启动 Gateway 时读取控制台进程的同名环境变量。
未设置时，使用当前用户主目录下的 `GolandProjects/maas-gateway`。

可以在启动控制台前指定本机路径：

```bash
export MAAS_GATEWAY_DIR="$HOME/GolandProjects/maas-gateway"
go run ./cmd/dev-console
```

在 GoLand 中启动控制台时，将 `MAAS_GATEWAY_DIR` 加到 Run Configuration 的环境变量中。
修改进程环境变量后，需要重启控制台。Source directory 也支持 `${HOME}` 等环境变量，
以及直接填写路径；profile 保留变量表达式，不会被展开后的本机绝对路径覆盖。

只运行 Mock，不启动 Web 控制台或 Gateway 进程控制器：

```bash
go run ./cmd/dev-console mock
curl http://127.0.0.1:19090/healthz
```

使用其他 profile：

```bash
go run ./cmd/dev-console -profile profiles/my-debug-session.json
```

如果该 profile 不存在，控制台会根据内置默认值创建它。

## 调试 Gateway

1. 启动控制台并打开页面。
2. 确认 Gateway 源码目录和外部服务地址。
3. 点击 **Debug with Delve**。
4. 默认 profile 会在 `127.0.0.1:2345` 启动无界面的调试服务，等待调试器连接。
5. 在 GoLand 中创建 **Go Remote** 配置，主机设为 `127.0.0.1`，端口设为 `2345`，
   连接后设置断点并恢复进程执行。

如需 Gateway 在 IDE 连接前就开始运行，可在 profile 中开启 `continue_immediately`。
若希望先连接 IDE 再开始断点调试，保持关闭即可。

默认配置还关闭了 Delve 的严格 Go 版本检查：原配置说明记录的本地 Delve 1.24.2
会将 Go 1.25 识别为超出其声明支持范围的版本。升级 Delve 并确认版本兼容后，
可重新开启 `check_go_version`。

修改 Gateway YAML 后，点击 **Apply & Restart** 应用配置并重启 Gateway。
修改控制台或 Mock 的监听地址后，需要重启控制台。

## Mock 路由匹配

路由按优先级从高到低匹配，可设置以下条件：

- 请求头；
- 查询参数；
- JSON 请求体的顶层字段或点分路径，例如 `stream` 或 `metadata.tenant_id`。

多条路由可以使用相同的方法和路径。默认 profile 利用这一点区分已知 API Key、
未知 API Key、流式 Chat 请求和非流式 Chat 请求。

左侧按 **Method + Path** 合并显示接口，并标注场景数。选择接口后，在右侧的
“匹配 / 响应场景”中切换，或点击“新增场景”“复制当前场景”。每个场景有独立的
ID、名称、启用状态、Priority、匹配条件和响应。新增或复制的场景默认禁用，
配置好后勾选 **Enabled** 并点击“保存场景”。

例如，同一个接口可以配置以下场景：

| 场景 | Request match JSON | Priority | 响应 |
| --- | --- | --- | --- |
| 模拟错误 | `{"headers":{"X-Mock-Scenario":"error"}}` | 100 | HTTP 500，自定义错误体 |
| 默认成功 | `{}` | 0 | HTTP 200，正常响应 |

条件按 AND 组合，命中第一条就停止；相同优先级按 profile 中的顺序匹配。
切换场景时，合法的编辑会保留为浏览器内的草稿；点击保存后才写入 profile 并生效。
存在未保存的更改时，页面会显示提示。

“接口说明”用于描述用途，同一 Method + Path 下的场景共用，保存后显示在左侧接口下方，
不参与请求匹配。

响应类型选择 **SSE 流式响应** 后，可以逐卡片编辑事件 data 和发送前延迟，
支持新增、删除、上移、下移和 JSON 格式化，也可直接添加 `[DONE]`。
JSON data 会展开显示，无需手动编辑转义字符串；普通文本也可原样发送。
切换响应类型时会自动调整 Content-Type，保存格式仍为原有的 `response.sse` 数组。
“发送多少条事件后关闭”（`close_after_events`）设为 0 时发送全部，正数用于模拟提前断流。

## Gateway 日志

页面按日志级别着色：ERROR / FATAL / PANIC 为红色，WARN 为黄色，INFO 为绿色，
DEBUG / TRACE 为灰色。支持 JSON 的 `level` / `severity` 字段以及常见文本格式，
时间和日志来源单独着色。终端 ANSI 控制码会清理，日志内容按纯文本展示。

页面最多保留最近 5000 条日志。滚动到历史日志时不会强制跳回底部，回到底部后自动跟随新日志。

## 默认 Mock 接口

- `POST /api/internal/maas/v2/gateway/llm_router/list`
- `POST /api/internal/maas/v2/gateway/dynamic_config`
- `POST /api/internal/maas/v2/model/list`
- `POST /api/internal/platform/v1/api_key/get_tenant_id`
- `POST /api/internal/platform-base/v1/stop_guard/check`
- `POST /api/internal/maas/v2/tenant_config/all_usage_limit`
- `POST /api/internal/maas/v2/model_tenant_has_access`
- `POST /api/internal/maas/v2/tenant_config/detail`
- `POST /api/internal/maas/v2/data_count/receive_token_count`
- `POST /mock-engine/v1/chat/completions`

新增 Gateway 依赖时，可直接在 **Mock routes → Add** 中注册接口，无需修改 Go 代码。

## 运行时文件

生成的 Gateway 配置及其他会话产物存放在：

```text
.runtime/current/
└── gateway-config/
    └── config.ci.yml
```

`.runtime/` 已加入 Git 忽略规则。需要持久保存并纳入版本管理的配置放在 `profiles/`。

## Git 与项目边界

本目录使用独立的 Git 仓库。建议在 GoLand、Cursor 等 IDE 中直接打开
`maas-gateway-dev-console` 目录，并将版本控制根目录设置为本目录。

提交前可检查仓库根目录和待提交文件：

```bash
git rev-parse --show-toplevel
git status --short
```

仓库根目录应为当前项目目录。项目中的 Git 操作只管理本项目文件，
`.runtime/`、`.idea/`、日志和 `.DS_Store` 均已忽略。

如果打开的是父目录 `goProject`，IDE 仍可能显示父仓库及其他项目；
此时应选择本项目仓库进行提交。父目录中若也存在 Git 仓库，
不要在父仓库中将本项目作为普通文件夹或嵌套仓库误加入暂存区。

## 本地 API

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/api/profile` | 读取当前 profile |
| `PUT` | `/api/profile` | 校验、保存并启用 profile |
| `GET` | `/api/runtime` | 读取 Mock 和 Gateway 进程状态 |
| `POST` | `/api/gateway/start` | 启动 Gateway，请求体为 `{"mode":"run"}` 或 `{"mode":"debug"}` |
| `POST` | `/api/gateway/restart` | 应用生成的配置并重启 Gateway |
| `POST` | `/api/gateway/stop` | 停止受管理的 Gateway 进程组 |
| `GET` | `/api/requests` | 读取已记录的 Mock 请求 |
| `DELETE` | `/api/requests` | 清空已记录的 Mock 请求 |
| `GET` | `/api/logs` | 读取 Gateway 进程日志 |
| `DELETE` | `/api/logs` | 清空页面展示的 Gateway 日志 |

## 验证

```bash
go test ./...
go vet ./...
```

如已安装 Node.js，还可运行前端回归检查（只使用 Node 内置测试工具，不影响应用运行依赖）：

```bash
node --test scripts/ui.test.cjs
```

Mock 会记录请求头和请求体。控制台默认应绑定回环地址；profile、请求历史、
API Key 和 Gateway 日志均应作为本地调试数据妥善保管。
