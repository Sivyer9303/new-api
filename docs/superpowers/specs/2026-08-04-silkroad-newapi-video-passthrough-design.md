# Design: New API 渠道视频透传（SilkRoad 逆向低价 / 海外满血）

**日期：** 2026-08-04  
**状态：** 已批准 / 实现完成  
**读者：** 不一定熟悉 new-api 代码的维护者  

---

## 1. 背景（用白话说）

new-api 是一个 AI 网关：用户打你的接口，你再转发到真正的上游，并在中间做鉴权、选渠道、扣费。

SilkRoad（`https://ai.silkroadai.io`）本身也是 **new-api 兼容网关**。它对外提供视频接口：

- 提交：`POST /v1/video/generations`
- 查询：`GET /v1/video/generations/{task_id}`

我们自己的 new-api **对客户端**也已经有同名入口，但「渠道类型」决定了 **出站** 打哪家、用什么协议。目前：

| 渠道类型 | 出站协议 | 能否直接接 SilkRoad |
|----------|----------|---------------------|
| DoubaoVideo (54) | 火山官方 `/api/v3/contents/generations/tasks` | 否（实测 404） |
| Sora / OpenAI | `/v1/videos` | 否（能提交，但查询 `task_not_exist`） |
| New API (60) | 同步对话透传可用 | **视频任务适配器未注册，现在不行** |

本次目标：给 **New API 渠道** 补上视频任务透传，专门接 SilkRoad 的两档按秒视频。

---

## 2. 目标与非目标

### 2.1 目标（MVP）

1. 用户打本站 `POST/GET /v1/video/generations`，渠道类型选 **New API**，base URL 填 SilkRoad。
2. 支持两档模型（按秒计费）：逆向低价 / 海外满血（见上）。
3. **只允许友好字段**：客户端必须传 `generation_type`、时长、`aspect_ratio` 等约定字段；由适配器按配置配方展开成上游 JSON。**禁止**开放 SilkRoad 原生字段直通。
4. **时长、比例必填**；取值必须落在 SilkRoad tab 配置的启用列表中；无默认秒数。
5. **视频参数全面配置化**（展示名 / 取值 / 上游字段名 + 生成类型配方）。见 §5.3。
6. **成片自托管**：成功后由指定节点下载到 **本地磁盘**；对外只返回本站地址；**全局保留 7 天**。多机时固化成片节点。见 §5.6。
7. 计费：售价/秒 × 时长预扣；成功保持预扣；失败全退；上游扣了本站必须扣。
8. **测试以 mock 为主**。

### 2.2 非目标（明确不做）

- 国内企业级 `seedance2.0-pro-*`（token 结算）
- 把客户端统一成 OpenAI `/v1/videos`
- 依赖 SilkRoad `GET /v1/usage` 做实时对账
- 开放原生透传 / 高级用户绕过友好字段
- 修改 DoubaoVideo / Sora 现有行为
- 永久无限期免费托管（默认有保留期；过期删除）

---

## 3. 关键实测结论（设计依据）

1. SilkRoad 响应头含 `x-new-api-version`，错误类型 `new_api_error`。
2. `/v1/video/generations` 提交 + 轮询全链路可用；完成态示例字段：`status=completed`，`video_url`，部分场景还有 `url`。
3. `/v1/videos` 查询对 Seedance 任务返回 `task_not_exist` → 禁止走 Sora 适配器路径。
4. 两档按秒；失败上游会退（文档 `/v1/usage` 的 `type=refund` 亦提及视频失败退款）。本站对齐：失败退预扣。

---

## 4. new-api 里相关机制（读代码后的说明）

不必会写 Go，但实现/评审时认这几块即可：

```
客户端
  → router/video-router.go          注册 /v1/video/generations
  → controller.RelayTask            选渠道、重试、成功后写库
  → relay.RelayTaskSubmit           校验 → 估价 → 预扣 → 调适配器
  → channel.TaskAdaptor             「某一类上游怎么说话」
  → service 定时轮询 UpdateVideoTasks
       → FetchTask + ParseTaskResult
       → 成功 settle / 失败 RefundTaskQuota
```

### 4.1 这是「已有能力」还是「新功能」？（重要）

用一张表说清，避免误解成「再造一套 new-api」：

| 东西 | 现在有没有 | 本次要不要动 |
|------|------------|--------------|
| 渠道类型 **New API (60)** | **已有**（后台可选） | 继续用，不新建渠道类型 |
| 同步对话透传（chat / Claude / Gemini…） | **已有**（`relay/channel/newapi`） | 不动 |
| 客户端入口 `/v1/video/generations` | **已有**（路由 + 任务框架） | 不动入口 |
| 预扣 / 轮询 / 失败退款框架 | **已有** | 复用 |
| **视频 TaskAdaptor（出站怎么打 SilkRoad）** | **没有**（`GetTaskAdaptor` 未注册 60） | **新增一小块适配代码** |

结论：

- **不是**新开一个「SilkRoad 渠道类型 / 新业务子系统」。
- **是**给已有 New API 渠道 **补上视频任务适配能力**（和 DoubaoVideo、Sora 一样，挂在现有 `TaskAdaptor` 插件点上）。
- 对运营：还是选「New API」，填 SilkRoad base URL + 对应档 key。
- 对开发：新增包 `relay/channel/task/newapi/` + 一行注册，属于 **增量功能模块（适配器）**，不是从零做网关。

### 4.2 为什么改 New API 渠道，而不是新建「SilkRoad」类型

- 产品语义已是「上游是另一台 new-api」；SilkRoad 符合。
- 对话透传已存在（`relay/channel/newapi`）；缺的是 **TaskAdaptor**。
- 少一个渠道类型 = 少前端常量 / 文档 / 运营认知成本。

### 4.3 计费如何自动「成功保持预扣」

`controller.RelayTask` 写入任务时：

```go
PerCallBilling: ... || relayInfo.PriceData.UsePrice
```

当模型配置了 **固定 `model_price`**（按次/按量价表，而非纯 `model_ratio`）时，`UsePrice=true` → `PerCallBilling=true`。

轮询结束时 `settleTaskBillingOnComplete`：**若 PerCallBilling，跳过差额结算，保留预扣。**

这对按秒档正好：

- 配置：`model_price` = 本站售价 / 秒
- `EstimateBilling` 返回 `{"seconds": N}`
- 预扣 = `model_price × QuotaPerUnit × group_ratio × seconds`
- 成功：不重算；失败：`RefundTaskQuota`

### 4.4 成功时结果 URL 怎么存

轮询里若 `ParseTaskResult` 给出非空 `Url`（普通 http 直链），会写入 `task.PrivateData.ResultURL`。  
因此透传适配器必须在 `completed`/`SUCCESS` 时解析 `video_url`（或等价 `url`/`result_url`）。

---

## 5. 架构设计

### 5.1 新增包

建议路径：`relay/channel/task/newapi/`（与 `task/sora`、`task/doubao` 并列）。

> 注意：已有 `relay/channel/newapi` 是 **同步** Adaptor；视频用 **TaskAdaptor**，放 `task/newapi` 避免混在一个类型里。

### 5.2 注册

在 `relay/relay_adaptor.go` 的 `GetTaskAdaptor` 中增加：

```go
case constant.ChannelTypeNewAPI:
    return &tasknewapi.TaskAdaptor{}
```

### 5.3 TaskAdaptor 行为

| 方法 | 行为 |
|------|------|
| `ValidateRequestAndSetAction` | 先 `ValidateBasicTaskRequest`，再按命中的 **profile** 校验：时长必填∈配置、比例（若 required）、`generation_type`（若传）及素材张数；**无默认时长** |
| `EstimateBilling` | 仅使用已通过白名单的时长 N，返回 `{"seconds": float64(N)}`（无默认值分支） |
| `AdjustBillingOnSubmit` | `nil`（MVP） |
| `AdjustBillingOnComplete` | `0`（依赖 PerCallBilling 保持预扣） |
| `BuildRequestURL` | `{base}/v1/video/generations` |
| `BuildRequestHeader` | `Authorization: Bearer {key}`，`Content-Type: application/json` |
| `BuildRequestBody` | 读原始 JSON → `map` → 写入 `info.UpstreamModelName` → 再 marshal（保留 `aspect_ratio`、`reference_*`、`video_config`、`generate_audio` 等） |
| `DoRequest` | `channel.DoTaskApiRequest` |
| `DoResponse` | 解析 `id`/`task_id`；对客户端回写公开 `PublicTaskID`；返回上游真实 task id |
| `FetchTask` | `GET {base}/v1/video/generations/{upstreamTaskId}` |
| `ParseTaskResult` | 状态映射见下；成功填 `Url` |
| `ConvertToOpenAIVideo` | 可选实现，便于以后 `/v1/videos` 查询；MVP 主路径是 generations |

### 5.3.1 SilkRoad 视频参数总览（来自上游文档）

两档共用的「用户可选维度」大致如下（**国内企业级不做**）：

| 维度 | 逆向低价 (`seedance-2.0-*`) | 海外满血 (`dreamina-seedance-2-0-*`) | 备注 |
|------|------------------------------|--------------------------------------|------|
| 时长 | `seconds`: `"10"` / `"15"` | `duration`（文档默认 4，示例 5；枚举可配） | **本站必填，无默认** |
| 画面比例 | `aspect_ratio`: 16:9 / 9:16 / 1:1 / 4:3 / 3:4 / **21:9** | 同左 | 文档可缺省；**本站可配置为必填** |
| 生成类型 | 无单独字段，由素材 + `video_config.reference_mode` 组合 | 文生用不带 `-ref` 模型；带图/首尾帧/音视频用 `-ref` + `image`/`images`/`first_frame`/`last_frame`/`reference_videos`/`audio_url` | **应用层概念，需配置成「配方」** |
| 是否生成声音 | （文档未强调） | `generate_audio` true/false | 可配成开关选项 |
| 其它透传 | `prompt`、各类 URL 数组 | 同左 | 内容型字段，一般不做成下拉，只校验「该类型是否必填」 |

**生成类型 ↔ 上游怎么表达（设计时必须认清）：**

上游 **没有** 名叫 `generation_type` 的字段。所谓「文生 / 图生 / 多图 / 首尾帧」是：

- 选哪个模型（要不要 `-ref`）
- 带哪些素材字段
- 是否设置 `video_config.reference_mode`

因此配置里的「生成类型」应是一条 **配方（recipe）**，而不是单一 scalar。

### 5.3.2 统一配置模型：展示名 + 取值 + 上游字段

所有「下拉类」选项统一用同一结构（便于拓展 tab 复用表格编辑器）：

```json
{
  "label": "16:9 横屏",
  "value": "16:9",
  "upstream_key": "aspect_ratio",
  "enabled": true,
  "sort": 10
}
```

| 字段 | 含义 |
|------|------|
| `label` | **展示名**（后台 / 将来 Playground 下拉显示） |
| `value` | **实际参数值**（客户端应传的值 / 写入上游的值） |
| `upstream_key` | **上游 JSON 字段名**（支持点路径，如 `video_config.reference_mode`） |
| `enabled` / `sort` | 是否启用、排序 |

**时长**也用同一结构；因不同档上游字段名不同（`seconds` vs `duration`），按 **产品档 / 规则组** 分挂，而不是全球一个列表。

**生成类型**在 `value`/`label` 之外增加配方字段（见下）。

### 5.3.3 建议完整配置结构（`silkroad_setting`）

```json
{
  "profiles": [
    {
      "id": "seedance_reverse",
      "label": "逆向低价",
      "model_prefixes": ["seedance-2.0-"],
      "durations": [
        { "label": "10 秒", "value": "10", "upstream_key": "seconds", "enabled": true, "sort": 1 },
        { "label": "15 秒", "value": "15", "upstream_key": "seconds", "enabled": true, "sort": 2 }
      ],
      "aspect_ratios": [
        { "label": "16:9 横屏", "value": "16:9", "upstream_key": "aspect_ratio", "enabled": true, "sort": 1 },
        { "label": "9:16 竖屏", "value": "9:16", "upstream_key": "aspect_ratio", "enabled": true, "sort": 2 },
        { "label": "1:1 方形", "value": "1:1", "upstream_key": "aspect_ratio", "enabled": true, "sort": 3 },
        { "label": "4:3", "value": "4:3", "upstream_key": "aspect_ratio", "enabled": true, "sort": 4 },
        { "label": "3:4", "value": "3:4", "upstream_key": "aspect_ratio", "enabled": true, "sort": 5 },
        { "label": "21:9 超宽", "value": "21:9", "upstream_key": "aspect_ratio", "enabled": true, "sort": 6 }
      ],
      "aspect_ratio_required": true,
      "generation_types": [
        {
          "label": "文生视频",
          "value": "text2video",
          "enabled": true,
          "sort": 1,
          "require_ref_model": false,
          "upstream_sets": [],
          "media_requirements": { "images_min": 0, "images_max": 0 }
        },
        {
          "label": "图生视频（单图）",
          "value": "image2video",
          "enabled": true,
          "sort": 2,
          "require_ref_model": false,
          "upstream_sets": [
            { "upstream_key": "image_url", "from": "images[0]" }
          ],
          "media_requirements": { "images_min": 1, "images_max": 1 }
        },
        {
          "label": "多图生视频",
          "value": "multi_image",
          "enabled": true,
          "sort": 3,
          "require_ref_model": false,
          "upstream_sets": [
            { "upstream_key": "reference_image_urls", "from": "images" },
            { "upstream_key": "video_config.reference_mode", "value": "auto" }
          ],
          "media_requirements": { "images_min": 2, "images_max": 9 }
        },
        {
          "label": "首帧",
          "value": "start_frame",
          "enabled": true,
          "sort": 4,
          "require_ref_model": false,
          "upstream_sets": [
            { "upstream_key": "image_url", "from": "images[0]" },
            { "upstream_key": "video_config.reference_mode", "value": "start_frame" }
          ],
          "media_requirements": { "images_min": 1, "images_max": 1 }
        },
        {
          "label": "首尾帧",
          "value": "start_end",
          "enabled": true,
          "sort": 5,
          "require_ref_model": false,
          "upstream_sets": [
            { "upstream_key": "reference_image_urls", "from": "images" },
            { "upstream_key": "video_config.reference_mode", "value": "start_end" }
          ],
          "media_requirements": { "images_min": 2, "images_max": 2 }
        }
      ],
      "extra_options": [
        {
          "label": "参考模式-自动",
          "value": "auto",
          "upstream_key": "video_config.reference_mode",
          "enabled": true,
          "sort": 1
        }
      ]
    },
    {
      "id": "dreamina_overseas",
      "label": "海外满血",
      "model_prefixes": ["dreamina-seedance-2-0-"],
      "durations": [
        { "label": "4 秒", "value": "4", "upstream_key": "duration", "enabled": true, "sort": 1 },
        { "label": "5 秒", "value": "5", "upstream_key": "duration", "enabled": true, "sort": 2 }
      ],
      "aspect_ratios": [ "…同上…" ],
      "aspect_ratio_required": true,
      "generation_types": [
        {
          "label": "文生视频",
          "value": "text2video",
          "require_ref_model": false,
          "upstream_sets": [],
          "media_requirements": { "images_min": 0, "images_max": 0 }
        },
        {
          "label": "图生 / 参考生",
          "value": "image2video",
          "require_ref_model": true,
          "upstream_sets": [
            { "upstream_key": "image", "from": "images[0]" }
          ],
          "media_requirements": { "images_min": 1, "images_max": 9 }
        },
        {
          "label": "首尾帧",
          "value": "start_end",
          "require_ref_model": true,
          "upstream_sets": [
            { "upstream_key": "first_frame", "from": "images[0]" },
            { "upstream_key": "last_frame", "from": "images[1]" }
          ],
          "media_requirements": { "images_min": 2, "images_max": 2 }
        }
      ],
      "extra_options": [
        {
          "label": "生成声音-开",
          "value": "true",
          "upstream_key": "generate_audio",
          "enabled": true,
          "sort": 1
        },
        {
          "label": "生成声音-关",
          "value": "false",
          "upstream_key": "generate_audio",
          "enabled": true,
          "sort": 2
        }
      ]
    }
  ]
}
```

#### 生成类型配方字段

| 字段 | 含义 |
|------|------|
| `label` / `value` | 展示名 / 本站对外枚举值（客户端可传 `generation_type` 或你们约定字段） |
| `require_ref_model` | 是否要求上游模型名带 `-ref`（满血档关键） |
| `upstream_sets` | 写入上游的键值：固定 `value`，或 `from` 映射客户端素材（如 `images[0]`） |
| `media_requirements` | 图片张数上下限等；不满足 → 400 |

> **对外 API 形态（已确认）：**  
> - **只允许友好字段**：`model`、`prompt`、`generation_type`、时长（`seconds` 或按 profile 约定）、`aspect_ratio`、素材 `images[]`（及配方需要的其它友好字段）。  
> - **比例必填**（`aspect_ratio_required` 固定为 true，或配置里不可关闭）。  
> - 适配器按配方展开成上游 JSON；拒绝夹带未声明的原生键（或 strip 后忽略——推荐 **直接 400 unknown_field**，更安全）。

### 5.3.4 配置入口与校验原则

- 存放：`setting/silkroad_setting` + Option `silkroad_setting.*`（大块可用单个 JSON：`silkroad_setting.profiles`）
- UI：拓展 → **SilkRoad** tab  
  - Profile 卡片（逆向 / 满血）  
  - 子表：时长 / 比例 / 生成类型 / 额外开关  
  - 每行编辑：展示名、取值、上游字段名（生成类型再编辑配方）
- 保存校验：`label`/`value`/`upstream_key` 非空；同 profile 内 `value` 不重复；`allowed` 至少启用一项；时长 value 可解析为正数
- 请求校验：命中 profile → **`generation_type` 必填** ∈ 启用列表；**时长必填** ∈ 启用列表；**`aspect_ratio` 必填** ∈ 启用列表；素材张数满足配方；拒绝未知字段
- **未命中 profile → 400**（不放行任意参数）

### 5.3.5 适配器行为（相对上一版的补充）

| 方法 | 补充 |
|------|------|
| `ValidateRequestAndSetAction` | 友好字段白名单；`generation_type` / 时长 / `aspect_ratio` 均必填且 ∈ 配置；按配方校验素材 |
| `EstimateBilling` | 仅用已通过校验的时长 |
| `BuildRequestBody` | **不透传客户端原始体**；由友好字段 + 配方 + model 映射 **生成** 上游 JSON |

### 5.4 状态映射（资损关键）

| 上游 status（大小写不敏感） | 内部 |
|-----------------------------|------|
| `queued` / `pending` | Queued |
| `in_progress` / `processing` / `running` | InProgress |
| `completed` / `success` / `SUCCESS` | Success。解析上游 `video_url`/`url`/`result_url` 写入 **私有字段**（不直接当对外 ResultURL）。无 Url 时仍标 Success 并保持预扣 + 告警。 |
| `failed` / `failure` / `cancelled` | Failure |
| 其它未知 | **保持 InProgress** |

**超时策略：** 长期非终态不自动全额退。

### 5.5 运营渠道配置

建议建 **两个** New API 渠道（因 SilkRoad 不同档 key 互斥）：

1. base = `https://ai.silkroadai.io`，key = 逆向低价档，模型列表 = `seedance-2.0-720,seedance-2.0-1080`
2. base 相同，key = 海外满血档，模型列表 = `dreamina-seedance-2-0-...`

模型价表：每个模型配置 `model_price` = **本站售价 / 秒**，且售价 ≥ 上游成本 + 差价。

### 5.6 成片自托管（不暴露上游链接）——可行，建议纳入本设计

#### 结论

**可行，而且对 SilkRoad/火山签名链几乎是刚需**（链接常约 24h 过期）。现有 new-api 的 `VideoProxy` 多是 **临时回源上游**，并不落盘；若只把上游 `video_url` 写进 `ResultURL` 再返回给用户，会直接暴露第三方地址，且过期后失效。

你要的模式是：

```
上游 completed + video_url
  → 本站尽快下载到自有存储（本地盘 / S3 / OSS 等）
  → 任务对外 ResultURL = https://你的站/v1/videos/{task_id}/content（或独立 /files/...）
  → 用户下载只打本站
  → 保留 N 天（如 7）后删除文件，之后下载 410/404
```

#### 推荐状态机（兼顾资损）

| 阶段 | 行为 |
|------|------|
| 轮询到 Success + 拿到上游 URL | **先按成功结算扣费**（上游已扣，本站必须扣）；上游 URL 只写入 **PrivateData（不对用户 API 返回）** |
| 异步/同步拉取成片 | 下载成功 → 写本地对象键；`ResultURL`=本站地址；标记 `storage_status=ready` |
| 下载失败 | **不退款**（差价口径）；`storage_status=pending_retry`，重试队列 + 告警；用户查任务可见「处理中/准备下载」而非上游链接 |
| 保留到期 | 定时任务删对象；`storage_status=expired`；下载接口返回过期 |

> 关键：下载失败若退款，会对上游形成亏损；与你的资损原则冲突。应 **钱照扣、片重试**。

#### 配置（可放进同一 SilkRoad tab）

```json
{
  "storage": {
    "enabled": true,
    "driver": "local",
    "local_dir": "/data/silkroad-videos",
    "retention_days": 7,
    "max_retry": 5,
    "public_base_path": "/v1/videos",
    "ingest_node_name": "video-storage-1"
  }
}
```

已确认：**只用本地磁盘**；**全局保留 7 天**（不按模型拆分）。`ingest_node_name` 见下方多机说明。

#### 两台服务器 + Cloudflare DNS 分流（无传统 LB）

你现在是 **CF DNS 把流量分到两台**（解析到不同 IP），**没有**按路径转发的四层/七层 LB。  
因此：**不能**指望「同一个 `api.example.com` 下，只有 `/v1/videos/*/content` 进成片机」——用户下次解析可能打到另一台。

**推荐架构：API 双机 + 成片专用域名（只解析到一台）**

```
用户 API：     api.example.com     → CF DNS → 机器 A 或 B（现有分流不变）
用户下视频：   video.example.com   → CF DNS → 仅机器 A（成片节点，A 记录只指一台）
```

流程：

1. A/B 都可接提交、轮询、计费（照旧）。
2. 仅 **成片节点 A**（`NODE_NAME` = 配置的 `ingest_node_name`）执行：下载上游视频 → 写入 A 本地盘 → 过期清理。
3. 任务对外返回的下载地址 **一律用成片域名**，例如：  
   `https://video.example.com/v1/videos/{task_id}/content`  
   不要用会漂移的 `api.example.com`。
4. 非成片节点 B 轮询到 Success：把上游 URL 写入私有字段并标记待拉取；**不在 B 落盘**。成片节点通过 DB 队列认领并下载（或 B 调 A 内网拉取接口——若两机内网通）。
5. 用户下载永远打 `video.example.com` → DNS 只会到 A → 文件一定在。

配置示例：

```json
{
  "storage": {
    "enabled": true,
    "driver": "local",
    "local_dir": "/data/silkroad-videos",
    "retention_days": 7,
    "max_retry": 5,
    "ingest_node_name": "node-a",
    "public_download_base_url": "https://video.example.com"
  }
}
```

| 方案 | 是否适合 CF DNS 无路径 LB | 说明 |
|------|---------------------------|------|
| **成片专用子域（推荐）** | ✅ | 下载域名 DNS 单指成片机；API 域名继续双机分流 |
| 两台都存本地 | ❌ 除非再加同步/共享盘 | CF 随机打到无文件的那台会 404 |
| 应用 302 到成片机公网 IP/域名 | ✅ 可用 | 等价于专用子域；不如直接把对外 URL 写成子域清晰 |
| 指望 CF DNS 按 path 分流 | ❌ | DNS 不看 path |

**运维注意：**

- `video.example.com` 建议也走 CF 代理（橙云）便于 HTTPS/缓存策略；**缓存要关或按 task 谨慎**，避免缓存了 404/过期响应。
- 成片节点磁盘与进程要重点监控；API 节点挂了仍可能下视频，成片节点挂了则新片无法落盘、旧片无法下。
- 两机数据库须是 **同一套 DB**（任务队列/状态共享）；本地盘只在成片机。

---

## 6. 请求 / 响应契约（透传）

### 6.1 提交（客户端 → 本站 → SilkRoad）

请求体字段尽量透传。本站校验最少集：

- `model`、`prompt` 必填
- **时长必填且必须命中模型对应白名单**（§5.3.1）；不使用 `MaxTaskDurationSeconds` 作为「随便填只要不太大」的放行条件
- 低价档向上游传 `seconds` 字符串；满血档传 `duration` 数字（与文档一致；若客户端只传了另一种字段名，适配器在校验通过后可规范化再透传）

客户端响应（对齐现有视频任务习惯 / SilkRoad）：

- 对外 `id` / `task_id` = 本站公开 `task_xxxx`
- 库内保存上游真实 id（如 `cgt-...` / `task_...`）供轮询

### 6.2 查询

- 客户端：`GET /v1/video/generations/{公开task_id}` → 读本地任务（现有 `RelayTaskFetch`）
- 后台轮询：用上游 id 打 SilkRoad，更新本地状态与 `ResultURL`

---

## 7. 资损与计费细则

| 场景 | 本站行为 | 是否符合「上游扣了我也要扣」 |
|------|----------|------------------------------|
| 提交上游失败 | defer Refund 预扣 | 上游未扣，退合理 |
| 轮询 Failure | RefundTaskQuota | 上游失败应退，对齐 |
| 轮询 Success + 有 URL | PerCallBilling 保持预扣 | 同步扣住 |
| Success 但无 URL | **仍保持预扣** + 告警（不自动退） | 避免上游已扣、本站误退；人工补链或事后处理 |
| 未知状态长时间 | 不自动退 | 防止误退导致对上游亏损 |
| 预扣偏高 | 可接受 | 差价模型允许 |

**不做：** 用 `/v1/usage` 实时校准金额（MVP）。二期若确认视频的 `request_id` 规则，再加对账任务。

---

## 8. 代码改动清单（实现时）

| 文件 / 区域 | 改动 |
|-------------|------|
| `relay/channel/task/newapi/*` | TaskAdaptor + mock 单测 |
| `relay/relay_adaptor.go` | 注册 ChannelTypeNewAPI |
| `setting/silkroad_setting/*` | profiles + storage 配置 |
| 拓展 SilkRoad tab UI + i18n | 参数表 + 保留天数等 |
| 成片拉取 worker + 存储抽象 + 过期清理 | §5.6 |
| 查询/下载接口 | 只返回本站 URL；隐藏上游 |
| **不改** | Doubao / Sora 既有计费核心 |

预计新增业务代码量：一个小适配器包（约数百行）+ 测试。

---

## 9. 测试策略（强制：mock 为主）

上游一次视频很贵，**默认 CI / 本地不得打真 SilkRoad**。真机冒烟仅人工、可选。

### 9.1 单元测试（`relay/channel/task/newapi/adaptor_test.go`）

使用 `testify/require` + `assert`，表格驱动。Mock 手段：

- `ParseTaskResult`：直接喂 JSON 字节，无 HTTP
- `BuildRequestURL` / `EstimateBilling`：构造 `gin.Context` + body（可参考 `relay/common/relay_utils_test.go`）
- `BuildRequestBody`：断言透传字段仍在、`model` 被替换
- `FetchTask`：`httptest.Server` mock 上游，断言 path/header
- `DoResponse`：`httptest` 假响应，断言返回 upstream id 与写给客户端的公开 id

建议用例（至少覆盖）：

1. `EstimateBilling`：合法 `seconds:"10"`（配置含 10）→ `seconds=10`
2. `EstimateBilling`：合法 `duration:5`（配置含 5）→ `seconds=5`
3. `Validate`：缺少时长 → 400（**无默认**）
4. `Validate`：秒数不在 **当前配置** 白名单 → 400
5. `Validate`：模型不匹配任何 `duration_rules` → 400
6. 配置热更新：改 `allowed_seconds` 后，新请求按新名单校验（用可注入的 getter mock）
7. 保存配置：空名单 / 非正整数 / 非法 JSON → 拒绝
8. `BuildRequestURL` 固定后缀 `/v1/video/generations`
9. `BuildRequestBody` 保留透传字段 + 规范化 `preferred_field`
10. `BuildRequestBody` 写入映射后的 upstream model
11. `DoResponse` 兼容只有 `task_id` / 只有 `id`
12. `DoResponse` 缺少 id → 错误
13. `ParseTaskResult`：`queued` / `in_progress` / `completed`+`video_url`
14. `ParseTaskResult`：`SUCCESS` + `video_url`
15. `ParseTaskResult`：`failed` / `FAILURE`
16. `ParseTaskResult`：`completed` 但无 url → 仍 Success，Url 为空
17. `ParseTaskResult`：未知 status → InProgress（非空 Status）
18. `FetchTask` mock server：path 含 task id，Authorization Bearer
19. （可选）前端 SilkRoad tab：schema 校验空名单

### 9.2 计费相关单测（可放同包或 `service` 已有模式）

不强制改 `service/task_billing_test.go`，但适配器侧应文档化断言：

- 当 `UsePrice=true` 时完成态不依赖 `TotalTokens`
- 本适配器 `AdjustBillingOnComplete` 恒为 0

若易测：用现有 `settleTaskBillingOnComplete` + mock adaptor，断言 PerCallBilling 跳过重算（已有类似测试可参考，避免重复造轮子）。

### 9.3 禁止的「假测试」

- 随机 fuzz、sleep 计时、只打日志无断言
- 为了覆盖率调用真网
- 断言私有常量文件布局而无行为契约

### 9.4 人工冒烟（可选、付费）

仅用海外满血或逆向低价测试 key：

1. 本站 New API 渠道 → 提交 1 条最短视频  
2. 轮询至成功，确认本站扣费与 ResultURL  
3. 故意错误模型确认失败退款  

不进入默认 CI。

---

## 10. 风险与缓解

| 风险 | 缓解 |
|------|------|
| 状态字符串遗漏导致卡住 | 表驱动测试覆盖文档出现过的所有 status；未知→InProgress |
| 成功无 URL | 保持预扣 + 告警（不自动退），优先防对上游亏损 |
| 误退导致对上游亏损 | 未知/超时不自动退 |
| 两档 key 用混 | 运营分渠道；文档写清 |
| 透传丢字段 | BuildRequestBody 用 map 透传 + 测试断言 |
| `/v1/videos` 被误用 | 本适配器只实现 generations 路径；不注册为 Sora |

---

## 11. 实现顺序建议

1. 写 `silkroad_setting` 校验 + 默认配置测试  
2. 写时长白名单 Validate / EstimateBilling 测试（注入 mock 配置）  
3. 实现 TaskAdaptor + 注册  
4. 补 FetchTask / DoResponse / ParseTaskResult mock 测试  
5. 拓展设置 SilkRoad tab UI + i18n  
6. （可选）真 key 冒烟一条  
7. 运营：配两渠道 + 在 tab 里确认秒数名单

---

## 12. 开放问题（评审时确认）

1. ~~是否包含国内企业级~~ → **否**  
2. 资损口径 → **上游扣了本站必须扣**  
3. `completed` 无 URL → **保持预扣 + 告警**  
4. ~~默认时长~~ → **必填 + 配置白名单**  
5. ~~秒数硬编码~~ → **SilkRoad tab 配置**  
6. ~~API 形态~~ → **只允许友好字段**；**比例必填**  
7. ~~成片链接~~ → **本站本地盘托管 + 全局保留 7 天**  
8. ~~存储驱动~~ → **仅 local**  
9. **多机 + CF DNS** → API 双机分流不变；**成片用专用子域只解析到一台**；该机本地盘存片（已确认方向）

---

## 13. 附录：模型列表示例（运营）

**逆向低价**

- `seedance-2.0-720`
- `seedance-2.0-1080`

**海外满血（示例，以 SilkRoad `/v1/models` 为准）**

- `dreamina-seedance-2-0-480p` / `-ref`
- `dreamina-seedance-2-0-720p` / `-ref`
- `dreamina-seedance-2-0-1080p` / `-ref`
- `dreamina-seedance-2-0-4k` / `-ref`
- `dreamina-seedance-2-0-fast-480p` / `-ref`
- `dreamina-seedance-2-0-fast-720p` / `-ref`

`-ref`：参考输入档（图/首尾帧/参考视频/音频）；纯文生用不带 `-ref` 的模型。
