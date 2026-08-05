# SilkRoad New API Video Passthrough Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让渠道类型 New API (60) 能对接 SilkRoad 两档按秒视频（逆向低价 / 海外满血）：友好字段校验与配方展开、按秒预扣计费、成片仅落成片节点本地盘并通过专用下载域名对外提供、全局保留 7 天。

**Architecture:** 复用现有 `RelayTask` / 轮询 / `PerCallBilling`（`model_price`×`seconds`）。新增 `setting/silkroad_setting` 配置 + 拓展页 SilkRoad tab；新增 `relay/channel/task/newapi` TaskAdaptor 并注册到 `GetTaskAdaptor`；轮询成功后由成片节点异步下载到本地，`ResultURL` 仅暴露 `public_download_base_url`；CF DNS 下 API 双机、下载子域单指成片机。

**Tech Stack:** Go 1.22+、Gin、GORM、testify；前端 React 19 + RHF + Zod + i18next（拓展设置页）；本地磁盘存储。

**Spec:** `docs/superpowers/specs/2026-08-04-silkroad-newapi-video-passthrough-design.md`

## Global Constraints

- 不做国内企业级 token 结算；不做 `/v1/videos` 上游路径；不做原生字段透传。
- 时长、比例、`generation_type` 均必填；取值必须 ∈ 配置启用列表；无默认时长。
- 资损：上游成功则本站必须扣费；下载失败不退款，只重试。
- 存储：仅 `local`；`retention_days` 全局默认 7；成片仅 `ingest_node_name` 节点写入。
- 对外下载 URL 必须使用 `public_download_base_url`（成片专用子域），禁止返回上游 `video_url`。
- JSON 一律 `common.Marshal` / `common.Unmarshal`；测试用 `testify/require` + `assert`；默认 mock，不打真上游。
- 提交 commit 仅在用户明确要求时执行（计划里的 Commit 步骤视为「实现者检查点」，默认先 `git status` 展示，不擅自 commit）。

---

## File Map

| Path | Responsibility |
|------|----------------|
| `setting/silkroad_setting/config.go` | 配置结构、默认 profiles、Register、Getters |
| `setting/silkroad_setting/validate.go` | 保存/加载校验 |
| `setting/silkroad_setting/resolve.go` | 按 model 匹配 profile；选项查找 |
| `setting/silkroad_setting/*_test.go` | 配置单测 |
| `relay/channel/task/newapi/adaptor.go` | TaskAdaptor 实现 |
| `relay/channel/task/newapi/request_build.go` | 友好字段 → 上游 JSON |
| `relay/channel/task/newapi/constants.go` | ChannelName |
| `relay/channel/task/newapi/*_test.go` | Adaptor mock 单测 |
| `relay/relay_adaptor.go` | `case ChannelTypeNewAPI` |
| `model/task.go` | PrivateData 增加上游 URL / storage 字段 |
| `service/silkroad_video_storage.go` | 本地路径、写入、删除、是否成片节点 |
| `service/silkroad_video_ingest.go` | 认领下载、重试 |
| `service/silkroad_video_cleanup.go` | 过期删除 |
| `controller/video_proxy.go` 或新建 `controller/silkroad_video_content.go` | 从本地盘提供下载 |
| `service/task_polling.go` | Success 时入队 ingest、ResultURL 用本站地址 |
| `web/src/features/system-settings/extensions/*` | SilkRoad tab UI |
| `web/src/i18n/locales/*.json` | i18n |

---

### Task 1: `silkroad_setting` 配置模型与校验

**Files:**
- Create: `setting/silkroad_setting/config.go`
- Create: `setting/silkroad_setting/validate.go`
- Create: `setting/silkroad_setting/resolve.go`
- Create: `setting/silkroad_setting/config_test.go`
- Create: `setting/silkroad_setting/validate_test.go`
- Create: `setting/silkroad_setting/resolve_test.go`

**Interfaces:**
- Produces:
  - `type OptionItem struct { Label, Value, UpstreamKey string; Enabled bool; Sort int }`
  - `type GenerationType struct { Label, Value string; Enabled bool; Sort int; RequireRefModel bool; UpstreamSets []UpstreamSet; MediaRequirements MediaRequirements }`
  - `type UpstreamSet struct { UpstreamKey, Value, From string }`
  - `type MediaRequirements struct { ImagesMin, ImagesMax int }`
  - `type Profile struct { ID, Label string; ModelPrefixes []string; Durations, AspectRatios []OptionItem; GenerationTypes []GenerationType; ExtraOptions []OptionItem }`
  - `type StorageSetting struct { Enabled bool; Driver, LocalDir string; RetentionDays, MaxRetry int; IngestNodeName, PublicDownloadBaseURL string }`
  - `type SilkRoadSetting struct { Profiles []Profile; Storage StorageSetting }`
  - `func GetSilkRoadSetting() *SilkRoadSetting`
  - `func ValidateSilkRoadSetting(s *SilkRoadSetting) error`
  - `func MatchProfile(modelName string) (*Profile, bool)`
  - `func FindEnabledOption(items []OptionItem, value string) (*OptionItem, bool)`
  - `func FindGenerationType(p *Profile, value string) (*GenerationType, bool)`

- [ ] **Step 1: Write failing validate/resolve tests**

```go
package silkroad_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRejectsEmptyDurations(t *testing.T) {
	s := defaultSilkRoadSetting()
	s.Profiles[0].Durations = nil
	require.Error(t, ValidateSilkRoadSetting(&s))
}

func TestMatchProfileSeedance(t *testing.T) {
	// after Register + defaults loaded in test helper
	p, ok := MatchProfile("seedance-2.0-720")
	require.True(t, ok)
	assert.Equal(t, "seedance_reverse", p.ID)
}

func TestFindEnabledOptionDisabledSkipped(t *testing.T) {
	items := []OptionItem{{Label: "10s", Value: "10", UpstreamKey: "seconds", Enabled: false}}
	_, ok := FindEnabledOption(items, "10")
	assert.False(t, ok)
}
```

- [ ] **Step 2: Run tests — expect FAIL (package missing)**

Run: `cd /Users/sivyer/goproject/new-api && go test ./setting/silkroad_setting/ -count=1`

Expected: package not found / build failed

- [ ] **Step 3: Implement config + validate + resolve with defaults**

Defaults must include:
- Profile `seedance_reverse`: prefixes `seedance-2.0-`; durations 10/15 → `seconds`; aspect ratios 16:9,9:16,1:1,4:3,3:4,21:9; generation types text2video / image2video / multi_image / start_frame / start_end per design §5.3.3
- Profile `dreamina_overseas`: prefixes `dreamina-seedance-2-0-`; durations 4/5 → `duration`; same aspect ratios; generation types text2video / image2video / start_end with `require_ref_model` where needed
- Storage: `Enabled: true`, `Driver: "local"`, `LocalDir: "data/silkroad-videos"`, `RetentionDays: 7`, `MaxRetry: 5`, empty ingest/public URL (ops fills)

Validation rules:
- Each profile: non-empty id/label/prefixes; at least one enabled duration, aspect_ratio, generation_type
- OptionItem: label/value/upstream_key required when enabled
- Storage: if enabled, driver must be `local`; retention_days >= 1; max_retry >= 1; local_dir non-empty

```go
func init() {
	config.GlobalConfig.Register("silkroad_setting", &silkRoadSetting)
}
```

- [ ] **Step 4: Run tests — expect PASS**

Run: `go test ./setting/silkroad_setting/ -count=1`

Expected: PASS

- [ ] **Step 5: Checkpoint**

```bash
git status
# Do not commit unless user asked
```

---

### Task 2: TaskAdaptor — `ParseTaskResult`（纯 mock）

**Files:**
- Create: `relay/channel/task/newapi/constants.go`
- Create: `relay/channel/task/newapi/adaptor.go` (skeleton + ParseTaskResult)
- Create: `relay/channel/task/newapi/parse_test.go`

**Interfaces:**
- Consumes: `relaycommon.TaskInfo`, `model.TaskStatus*`
- Produces: `func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error)`

- [ ] **Step 1: Write failing parse tests**

```go
func TestParseTaskResultCompletedWithVideoURL(t *testing.T) {
	a := &TaskAdaptor{}
	body := []byte(`{"id":"cgt-1","status":"completed","progress":100,"video_url":"https://cdn.example/a.mp4"}`)
	info, err := a.ParseTaskResult(body)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusSuccess, info.Status)
	assert.Equal(t, "https://cdn.example/a.mp4", info.Url)
}

func TestParseTaskResultSUCCESS(t *testing.T) {
	a := &TaskAdaptor{}
	body := []byte(`{"status":"SUCCESS","video_url":"https://cdn.example/b.mp4"}`)
	info, err := a.ParseTaskResult(body)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusSuccess, info.Status)
}

func TestParseTaskResultUnknownKeepsInProgress(t *testing.T) {
	a := &TaskAdaptor{}
	info, err := a.ParseTaskResult([]byte(`{"status":"weird"}`))
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusInProgress, info.Status)
}

func TestParseTaskResultFailed(t *testing.T) {
	a := &TaskAdaptor{}
	info, err := a.ParseTaskResult([]byte(`{"status":"failed"}`))
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusFailure, info.Status)
}
```

- [ ] **Step 2: Run — expect FAIL**

Run: `go test ./relay/channel/task/newapi/ -count=1 -run Parse`

- [ ] **Step 3: Implement ParseTaskResult**

Map status case-insensitively: queued/pending → Queued; in_progress/processing/running → InProgress; completed/success → Success (Url from video_url|url|result_url); failed/failure/cancelled → Failure; else → InProgress with non-empty Status.

Embed `taskcommon.BaseBilling` on `TaskAdaptor`.

- [ ] **Step 4: Run — expect PASS**

Run: `go test ./relay/channel/task/newapi/ -count=1 -run Parse`

---

### Task 3: TaskAdaptor — 友好字段校验 + `EstimateBilling` + `BuildRequestBody`

**Files:**
- Create: `relay/channel/task/newapi/request_build.go`
- Create: `relay/channel/task/newapi/validate.go`
- Modify: `relay/channel/task/newapi/adaptor.go`
- Create: `relay/channel/task/newapi/billing_test.go`
- Create: `relay/channel/task/newapi/validate_test.go`
- Create: `relay/channel/task/newapi/request_build_test.go`

**Interfaces:**
- Friendly client JSON fields only: `model`, `prompt`, `generation_type`, `seconds`|`duration`, `aspect_ratio`, `images` ([]string), optional extras declared by ExtraOptions
- Produces: `ValidateRequestAndSetAction`, `EstimateBilling`, `BuildRequestBody`

- [ ] **Step 1: Write failing tests**

```go
func TestValidateRequiresGenerationTypeAndAspectAndDuration(t *testing.T) {
	// gin test context with body missing generation_type → TaskError 400
}

func TestValidateRejectsDurationNotInConfig(t *testing.T) {
	// seedance model + seconds "12" → 400
}

func TestEstimateBillingUsesSeconds(t *testing.T) {
	// after validate store, EstimateBilling returns map[string]float64{"seconds": 10}
}

func TestBuildRequestBodyImage2VideoSetsImageURL(t *testing.T) {
	// generation_type image2video + images[0] → upstream JSON has image_url, seconds "10", aspect_ratio
	// and does NOT contain generation_type
}
```

Use `httptest` + `gin.CreateTestContext` pattern from `relay/common/relay_utils_test.go`.

Inject config in tests via package-level `var getSetting = silkroad_setting.GetSilkRoadSetting` swapped in test, **or** pass setting into unexported helpers — prefer testable helpers:

```go
func validateFriendlyRequest(req FriendlyRequest, profile *silkroad_setting.Profile) error
func buildUpstreamBody(req FriendlyRequest, profile *silkroad_setting.Profile, upstreamModel string) ([]byte, error)
```

- [ ] **Step 2: Run — expect FAIL**

Run: `go test ./relay/channel/task/newapi/ -count=1`

- [ ] **Step 3: Implement validation + build**

Rules:
- Match profile by `info.OriginModelName`; no match → 400
- Require `generation_type`, aspect_ratio, duration/seconds
- Unknown top-level keys → 400
- `require_ref_model`: if true, `info.UpstreamModelName` must contain `-ref` or append/reject per ops convention — **reject with clear error if mapped model lacks `-ref`**
- Apply `UpstreamSets` (`value` or `from` images[i]/images)
- Normalize duration to `upstream_key` type: `seconds` → JSON string; `duration` → JSON number
- `EstimateBilling` returns `{"seconds": float64(N)}` only

- [ ] **Step 4: Run — expect PASS**

Run: `go test ./relay/channel/task/newapi/ -count=1`

---

### Task 4: TaskAdaptor — `FetchTask` / `DoResponse` / 注册

**Files:**
- Modify: `relay/channel/task/newapi/adaptor.go`
- Create: `relay/channel/task/newapi/http_test.go`
- Modify: `relay/relay_adaptor.go` — add import + `case constant.ChannelTypeNewAPI:`

**Interfaces:**
- `BuildRequestURL` → `{base}/v1/video/generations`
- `FetchTask` → `GET {base}/v1/video/generations/{task_id}`
- `DoResponse` → parse id/task_id; return upstream id; respond with PublicTaskID

- [ ] **Step 1: Write httptest tests for FetchTask path + Bearer header**

```go
func TestFetchTaskHitsGenerationsPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/video/generations/cgt-1", r.URL.Path)
		assert.Equal(t, "Bearer k", r.Header.Get("Authorization"))
		w.Write([]byte(`{"status":"queued"}`))
	}))
	defer srv.Close()
	a := &TaskAdaptor{}
	resp, err := a.FetchTask(srv.URL, "k", map[string]any{"task_id": "cgt-1"}, "")
	require.NoError(t, err)
	defer resp.Body.Close()
}
```

- [ ] **Step 2: Implement FetchTask, BuildRequestHeader, DoRequest, DoResponse, GetModelList/GetChannelName**

- [ ] **Step 3: Register in `GetTaskAdaptor`**

```go
case constant.ChannelTypeNewAPI:
	return &tasknewapi.TaskAdaptor{}
```

- [ ] **Step 4: Run tests**

Run: `go test ./relay/channel/task/newapi/ ./relay/ -count=1 -run 'FetchTask|GetTaskAdaptor|NewAPI'`

Note: if no GetTaskAdaptor test exists, add a tiny test in `relay/relay_adaptor_test.go` or `relay/channel/task/newapi/register_test.go` that calls `relay.GetTaskAdaptor(constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeNewAPI)))` and asserts non-nil.

---

### Task 5: Task PrivateData 扩展（上游 URL + 存储状态）

**Files:**
- Modify: `model/task.go` (`TaskPrivateData`)
- Create: `model/task_silkroad_private_test.go` (marshal round-trip)

**Interfaces:**
- Add fields:
  - `UpstreamResultURL string \`json:"upstream_result_url,omitempty"\``
  - `StorageStatus string \`json:"storage_status,omitempty"\`` // pending|ready|failed|expired
  - `StoragePath string \`json:"storage_path,omitempty"\``
  - `StorageExpiresAt int64 \`json:"storage_expires_at,omitempty"\``
  - `StorageRetryCount int \`json:"storage_retry_count,omitempty"\``

- [ ] **Step 1: Write round-trip test for new fields**
- [ ] **Step 2: Add fields**
- [ ] **Step 3: Run `go test ./model/ -count=1 -run Silkroad`**

---

### Task 6: 本地存储工具 + 成片节点判断

**Files:**
- Create: `service/silkroad_video_storage.go`
- Create: `service/silkroad_video_storage_test.go`

**Interfaces:**
- `func IsSilkRoadIngestNode() bool` — `common.NodeName == setting.Storage.IngestNodeName`（IngestNodeName 空则 false，避免双机误写）
- `func SilkRoadVideoLocalPath(taskID string) string` — under LocalDir
- `func WriteSilkRoadVideoFile(taskID string, r io.Reader) (absPath string, size int64, err error)`
- `func OpenSilkRoadVideoFile(taskID string) (*os.File, error)`
- `func DeleteSilkRoadVideoFile(taskID string) error`
- `func BuildSilkRoadPublicURL(taskID string) string` — `strings.TrimRight(PublicDownloadBaseURL,"/") + "/v1/videos/" + taskID + "/content"`

- [ ] **Step 1: Tests with `t.TempDir()` for write/read/delete + public URL join**
- [ ] **Step 2: Implement**
- [ ] **Step 3: `go test ./service/ -count=1 -run SilkRoadVideo`**

---

### Task 7: Ingest worker（仅成片节点）+ 轮询挂钩

**Files:**
- Create: `service/silkroad_video_ingest.go`
- Create: `service/silkroad_video_ingest_test.go`
- Modify: `service/task_polling.go` — on Success for New API / when storage enabled: set UpstreamResultURL from taskResult.Url; set ResultURL to public URL placeholder; StorageStatus=pending; do **not** put upstream URL in ResultURL
- Modify: `main.go` or existing system task registration — start ingest loop if ingest node

**Interfaces:**
- `func EnqueueSilkRoadVideoIngest(taskID string)` or mark DB fields only
- `func RunSilkRoadVideoIngestOnce(ctx context.Context) error` — claim tasks with StorageStatus=pending|failed and retry_count < max, download with SSRF client, write file, set ready + expires_at = now+retention_days*86400
- Hook: after Success settle path when channel type is NewAPI **or** when `silkroad_setting.Storage.Enabled` and profile matched — safest: check `task.Platform == strconv.Itoa(ChannelTypeNewAPI)`

Important polling change (pseudo):

```go
if taskResult.Status == model.TaskStatusSuccess {
    if shouldSilkRoadStore(task) {
        task.PrivateData.UpstreamResultURL = taskResult.Url
        task.PrivateData.StorageStatus = "pending"
        task.PrivateData.ResultURL = service.BuildSilkRoadPublicURL(task.TaskID)
        // Url for settle still success; billing unchanged
    } else if taskResult.Url != "" {
        task.PrivateData.ResultURL = taskResult.Url
    }
    ...
}
```

- [ ] **Step 1: Unit test ingest with `httptest` fake upstream video body + temp LocalDir; mock task struct in-memory if DB heavy — prefer testing `ingestOne(task, fetch)` helper**
- [ ] **Step 2: Implement download using `service.GetSSRFProtectedHTTPClient()`**
- [ ] **Step 3: Only run claim loop when `IsSilkRoadIngestNode()`**
- [ ] **Step 4: Wire polling Success branch**
- [ ] **Step 5: `go test ./service/ -count=1 -run SilkRoad`**

---

### Task 8: 本地下载接口 + 隐藏上游 URL

**Files:**
- Modify: `controller/video_proxy.go` **or** Create: `controller/silkroad_video_content.go` + router
- Modify: fetch DTO path so user-facing task JSON uses `GetResultURL()` only (already) — ensure UpstreamResultURL never copied into public TaskDto

Behavior for content GET:
1. Auth as today
2. If `StorageStatus==ready` and local file exists → `http.ServeContent`
3. If pending → 409/425 with message processing
4. If expired → 410
5. Never redirect client to UpstreamResultURL

- [ ] **Step 1: Handler tests with gin + temp file**
- [ ] **Step 2: Implement**
- [ ] **Step 3: Ensure `video.example.com` can hit same routes (same app binary on ingest node)

---

### Task 9: 过期清理

**Files:**
- Create: `service/silkroad_video_cleanup.go`
- Create: `service/silkroad_video_cleanup_test.go`
- Wire periodic run on ingest node only

- [ ] **Step 1: Test deletes file and sets StorageStatus=expired when StorageExpiresAt < now**
- [ ] **Step 2: Implement claim by expires_at**
- [ ] **Step 3: Run tests**

---

### Task 10: 拓展设置 SilkRoad tab（前端）

**Files:**
- Modify: `web/src/features/system-settings/types.ts` — ExtensionsSettings keys
- Modify: `web/src/features/system-settings/extensions/index.tsx` — defaults
- Modify: `web/src/features/system-settings/extensions/section-registry.ts` — section `silkroad`
- Create: `web/src/features/system-settings/extensions/silkroad-settings-section.tsx`
- Modify: `web/src/i18n/locales/en.json`, `zh.json` (and others if required by project i18n skill — at least en+zh)
- Backend: ensure option get/update accepts `silkroad_setting` JSON via existing config system (follow lottery_setting persistence path)

**UI minimum:**
- Storage: local_dir, retention_days (default 7), ingest_node_name, public_download_base_url, enabled
- Profiles: editable JSON textarea **or** simple tables for durations/aspect_ratios (MVP may use validated JSON editor like other advanced settings if tables too large — prefer structured forms for durations/aspect at least)

- [ ] **Step 1: Add section registry entry + empty form that loads/saves `silkroad_setting` blob**
- [ ] **Step 2: Zod validate retention_days >= 1, driver local, non-empty public_download_base_url when enabled**
- [ ] **Step 3: `bun run typecheck` in `web/`**
- [ ] **Step 4: Add i18n keys via project i18n skill if editing locales**

---

### Task 11: 端到端 mock 冒烟（无真上游）

**Files:**
- Create: `relay/channel/task/newapi/e2e_build_test.go` — full friendly request → upstream JSON golden
- Create: `service/silkroad_video_flow_test.go` — pending → ingest fake CDN → ready → open file

- [ ] **Step 1: Golden JSON for seedance text2video and dreamina image2video**
- [ ] **Step 2: Ingest flow test**
- [ ] **Step 3: `go test ./relay/channel/task/newapi/ ./service/ -count=1`**

---

### Task 12: 运营检查清单（文档，非代码）

**Files:**
- Modify: design spec status → Approved / Implementation in progress
- Create short ops notes in plan appendix below

- [x] **Step 1: Write ops checklist into this plan’s Appendix A**
- [x] **Step 2: Mark design doc status field to `已批准 / 实现完成`**

---

## Appendix A — Ops checklist

生产双机 + CF 上线前按序核对（配置存于共用 DB，拓展设置 → SilkRoad tab）：

1. 两台机连 **同一 DB**；环境变量 **`NODE_NAME`** 互不相同（与进程内 `common.NodeName` 一致）。
2. **成片机**：`NODE_NAME` = SilkRoad tab 的 **`ingest_node_name`**；**`storage.enabled=true`**，`local_dir` 可写；与另一台跑同一 new-api 二进制/版本。
3. **CF DNS**：`api` 子域 **A/B 分流** 两台；**`video`**（`public_download_base_url` 的主机名）**仅解析到成片机 A**。
4. 建 **两个 New API (60) 渠道**（逆向 / 海外各一 upstream key）；模型 **`model_price` = 售价/秒**（配合按秒 `EstimateBilling` / 按次预扣）。
5. SilkRoad tab：核对两档 profile 的 **时长、比例、generation_type** 启用项与运营预期一致；`retention_days` 默认 7。
6. **人工付费冒烟**：`POST /v1/video/generations` → 轮询 `completed` → 成片机 `local_dir` 出现文件 → 用 **video 域名** `GET /v1/videos/{task_id}/content` 下载 → 任务 JSON / 响应体 **不含上游 `video_url` 或 SilkRoad 域名**。

---

## Spec coverage self-check

| Spec requirement | Task |
|------------------|------|
| New API TaskAdaptor + generations URL | 2–4 |
| Friendly-only fields, required duration/aspect/generation_type | 3 |
| Configurable profiles label/value/upstream_key | 1, 10 |
| Per-second EstimateBilling + UsePrice/PerCallBilling | 3（配置运营） |
| Hide upstream URL; local store; 7-day retention | 5–9 |
| Ingest node + CF DNS dedicated download host | 6–8, Appendix A |
| Mock tests, no live SilkRoad in CI | 1–4, 6–7, 11 |

## Placeholder scan

无 TBD；存储驱动仅 local；保留天数全局 7。

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-08-04-silkroad-newapi-video-passthrough.md`.

**Two execution options:**

1. **Subagent-Driven (recommended)** — 每个 Task 开新子代理，任务间审查，迭代快  
2. **Inline Execution** — 本会话按 executing-plans 连续做，带检查点  

**Which approach?**
