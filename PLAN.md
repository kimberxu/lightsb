# lightsb 执行计划

执行计划、命令、步骤、上游事实基线与进度。必须遵从的约束在 `AGENTS.md`。

## 1. 上游事实基线

全部条目均已在本目录实测（命令与结果附后），基线为 tag `v1.14.2`。

| 项 | 值 |
|---|---|
| 基线 tag | `v1.14.2`（2026-09-24，当前最新稳定版），commit `af6e64c3b69e6132ebaee0e1a3d24e93903f6709` |
| 基线分支 | `stable`（上游 `stable` HEAD = `777dec2bb943a810c5df605324d2375486869e2f`，比 `v1.14.2` 多 **20** 个提交；**不用**它当基线） |
| 移除提交 | `8ae93a98`（"Remove overdue deprecated features"，最早出现于 `v1.13.1`，删掉 `route.go` 的 legacy inbound 转换 → 字段静默失效）、`6913b11e`（"Reject removed legacy inbound fields…"，最早出现于 `v1.13.4`，改为启动即报错） |
| 消费逻辑位置 | `route/route.go` 3 处：`367` / `754` / `878`，形如 `if action.OverrideDestination && M.IsDomainName(metadata.Domain) {`（`M` = `common/metadata`，`route.go:25` import 别名） |
| 断点位置 | `option/rule_action.go:325` `RouteActionSniff` 无该 JSON 字段；`route/rule/rule_action.go:107`（`case C.RuleActionTypeSniff` 在 `:106`）`NewRuleAction` 不赋值 → 内部字段恒 false，属死代码 |
| 字段定义 | `route/rule/rule_action.go:510` 起 `RuleActionSniff`，`:516` 为 `OverrideDestination bool`（保留但注释 `// Deprecated`）；行号按 `v1.14.2` |
| 上游立场 | Issue #2407 `closed/not_planned`、#2503 `closed/duplicate`、#2919 `closed/duplicate`、#3982 `closed/**completed**`、#4211 `closed/not_planned`（`gh api repos/SagerNet/sing-box/issues/<n>` 实测，**无 spam，3982 是 completed**）；PR #2505 补丁同构、`base=dev-next`、`closed` 未 `merged`。维护者明确拒绝该功能，**不要向上游提 PR** |
| 替代品不存在 | `route-options` 的 `override_address`/`override_port` 只接受**静态**地址（`option/rule_action.go:180` `OverrideAddress string`），无法表达运行期嗅探出的域名，**不是等价替代** |
| 上游 worktree 事实 | `docs/schema.json` 中 `override_destination` 出现 **0** 次；`.gitignore:21` = `CLAUDE.md`、`:22` = `AGENTS.md`（故 fork 文档必须 `git add -f`） |

复核命令（需要时重跑）：

```bash
git -C repo rev-list --count v1.14.2..upstream/stable            # 20
git -C repo grep -n 'OverrideDestination' -- route/route.go route/rule/rule_action.go
git -C repo merge-base --is-ancestor 6913b11e v1.13.4 && echo YES # YES（v1.13.3 为 NO）
```

## 2. 分支策略

**结论：`kimberxu/lightsb` 是 GitHub 原生 fork（继承上游全部 ref），fork 上只保留两个 ref——`stable`（rebase 锚点）与 `lightsb`（唯一工作分支，基线 `v1.14.2`）；其余 39 个继承分支全部删除，默认分支设为 `lightsb`。**

### 2.1 仓库拓扑（实测）

```
upstream = https://github.com/SagerNet/sing-box.git   (default branch: testing, 只读)
origin   = https://github.com/kimberxu/lightsb.git    (GitHub fork: fork=true, parent=source=SagerNet/sing-box)

fork 分支  stable   ← 上游 stable 线镜像，仅 rebase 锚点/升级判断，**不在其上开发**
          lightsb  ← 唯一工作分支，基线 = tag v1.14.2，默认分支
  └─ af6e64c3  Bump version (= v1.14.2)                      ← 基线
     2994aedb  docs: rewrite fork docs for lightsb            ← commit 0
     9210529f  docs: record fork branch pruning               ← commit 1
     8fcd24c7  sniff: restore override_destination for sniff action  ← commit 2（§3 补丁 + 文档 + schema + §5.2 测试）
     8dd73986  ci: fork-owned build and test workflows        ← commit 3（§4.2）
fork tag   v1.14.2-lightsb.1  ← 待打（出包时指向其上产物对应的提交）
```

fork 现状（实测，2026-10-01）：**`heads = 2`（`lightsb` 默认分支 + `stable` 镜像）**、`tags = 632`（全部继承自上游、
非我方推送）、`fork=true`（parent/source = `SagerNet/sing-box`）、public、`has_issues=false`；
`git ls-remote --symref origin HEAD` → `ref: refs/heads/lightsb`。

工作副本：`/root/workspace/lightsb/repo`（本机 shallow clone，`--no-tags`；`origin` = fork，无 worktree 级 `upstream` remote。
需要比对上游时用 `git fetch https://github.com/SagerNet/sing-box.git <ref>` 或临时 `git remote add upstream`）。

### 2.2 建分支与删分支（顺序不可颠倒）

```bash
# 1) 取全量 ref，建以 tag v1.14.2 为基线的工作分支
git -C repo fetch origin --prune
git -C repo switch -c lightsb v1.14.2
# 2) fork 文档改写（.gitignore:22 忽略 AGENTS.md，必须 -f）
git -C repo add -f AGENTS.md PLAN.md
git -C repo commit -m "docs: rewrite fork docs for lightsb"
git -C repo push -u origin lightsb

# 3) 先改默认分支，再删其余（默认分支被删会报错）
gh api -X PATCH repos/kimberxu/lightsb -f default_branch=lightsb

# 4) 保留 stable + lightsb，删除其余所有继承分支
#    注意：REST 的 ref 路径必须对 '/' 做完整百分号编码；gh api 只替换 '-' 不编码 '/'
#    （曾用 gh api -X DELETE .../heads/dependabot/go_modules/xxx 实测 422 Reference does not exist）
python3 - <<'PY'
import subprocess, urllib.parse
names = subprocess.run(["gh","api","repos/kimberxu/lightsb/branches","--paginate","--jq",".[].name"],
                       capture_output=True, text=True, check=True).stdout.split()
for b in [n for n in names if n not in ("lightsb","stable")]:
    gh = f"repos/kimberxu/lightsb/git/refs/heads/{urllib.parse.quote(b, safe='')}"
    r = subprocess.run(["gh","api","-X","DELETE",gh], capture_output=True, text=True)
    print("deleted" if r.returncode==0 else f"FAIL {b} {r.stderr.strip()[:80]}", b)
PY
# 另法（不改默认分支也能删，git 会原生编码 '/'）：git push origin --delete a b c
```

> 实测教训：整批 `git push origin --delete <40 个含 '/' 的分支>` 在本机跑了 900s 未返回；`gh api` 单条串行 38 个约 75s 完成。
> 批量删除前**必须**先确认默认分支不是待删分支，否则该条必失败。

`stable` 保持为上游镜像（fork 的 ref 不会自动跟随上游），升级判断前刷新：

```bash
git -C repo fetch upstream --prune
git -C repo push origin upstream/stable:stable      # 仅快进
git -C repo rev-list --count lightsb..upstream/stable   # 是否该换基线见 §2.4
```

### 2.3 tag 与 Release 纪律

- fork 上现有 632 个 tag **全部是上游自己的 ref**（fork 继承，非我方推送）；**不要再推任何上游 tag**，也不要基于它们建 Release。
- 出包只用自有 tag `v1.14.2-lightsb.N`（匹配 §4.2 的 `v*` 触发器）。
- 删除分支**不影响** tag 与 fork 关系；重置 fork 关系没必要，也不要做（会丢 `stable` 锚点）。

### 2.4 为什么基线用 **tag** 而不是 `stable` 分支 HEAD

`stable` HEAD 比 `v1.14.2` 多 20 个未发布提交。若基于它构建，二进制会含 1.14.2 之后的修复却仍标 `1.14.2` —— **版本号说谎**，
与"只针对最新稳定发行版"矛盾，排查失去基准。因此**基线锚定最新稳定 tag（`v1.14.2`），构建锚定同一 tag**；
`stable` 只用于判断"是否该升级基线"。

### 2.5 升级到新稳定版的流程

```bash
git fetch upstream --tags
git checkout lightsb
git rebase --onto v1.14.3 v1.14.2 lightsb   # 把补丁搬到新 tag
git tag v1.14.3-lightsb.1
git push origin lightsb v1.14.3-lightsb.1
```

`VERSION` 输入同步改为 `1.14.3`，产物名随之变化。**§5 验收门必须在新基线重跑**，尤其 §5.2 的 fail-before/pass-after。

## 3. 补丁规范（唯一真源）

> `make schema` 会用 `TAGS = DEFAULT_BUILD_TAGS_OTHERS`（`Makefile:3` 默认值；**不含** `with_naive_outbound`）
> 执行 `go run -ldflags "$(LDFLAGS_SHARED)" --tags "$(TAGS)" ./cmd/sing-box schema -o docs/schema.json`（`Makefile:36`），
> 是一次完整 go run 编译，属重任务：**不要在本机裸跑**，用 `systemd-run --user --scope -p MemoryMax=1G -- make schema` 包裹，或放到 CI/宿主机。
> 生成后 diff 应只多出 `override_destination` 一个布尔字段（`v1.14.2` 的 `docs/schema.json` 中该词出现 0 次，已核对）。

补丁内容（对 `v1.14.2` 逐字生效，已 `git apply --check` 验证 rc=0）：

```diff
--- a/option/rule_action.go
+++ b/option/rule_action.go
@@ -325,4 +325,5 @@
 type RouteActionSniff struct {
-	Sniffer badoption.Listable[string] `json:"sniffer,omitempty" enum:"tls,http,quic,dns,stun,bittorrent,dtls,ssh,rdp,ntp"`
-	Timeout badoption.Duration         `json:"timeout,omitempty"`
+	Sniffer             badoption.Listable[string] `json:"sniffer,omitempty" enum:"tls,http,quic,dns,stun,bittorrent,dtls,ssh,rdp,ntp"`
+	Timeout             badoption.Duration         `json:"timeout,omitempty"`
+	OverrideDestination bool                       `json:"override_destination,omitempty"`
 }
--- a/route/rule/rule_action.go
+++ b/route/rule/rule_action.go
@@ -106,6 +106,7 @@
 	case C.RuleActionTypeSniff:
 		sniffAction := &RuleActionSniff{
-			SnifferNames: action.SniffOptions.Sniffer,
-			Timeout:      time.Duration(action.SniffOptions.Timeout),
+			SnifferNames:        action.SniffOptions.Sniffer,
+			Timeout:             time.Duration(action.SniffOptions.Timeout),
+			OverrideDestination: action.SniffOptions.OverrideDestination,
 		}
 		return sniffAction, sniffAction.build()
@@ -510,8 +511,8 @@
 type RuleActionSniff struct {
 	SnifferNames   []string
 	StreamSniffers []sniff.StreamSniffer
 	PacketSniffers []sniff.PacketSniffer
 	Timeout        time.Duration
-	// Deprecated
+	// Override the connection destination with the sniffed domain name.
 	OverrideDestination bool
 }
```

**块内各 hunk、各文件段之间不得有空行**（git 会把空行后的一行当成 patch 头解析，报"片段没有头信息"）。

验证（去围栏后粘贴整块）：

```bash
cd repo && git apply --check - <<'PATCH'   # 期望 rc=0、无输出
…（上面的 diff，去掉 ```diff 围栏）…
PATCH
```

配置形态（与上游 deprecated 字段语义一致：仅在嗅探成功且 `M.IsDomainName` 成立时把 `Destination` 换成域名，**端口不变**）：

```json
{ "action": "sniff", "override_destination": true }
```

语义边界（以代码事实为准）：
- 无 SNI/Host（如连字面 IP 且客户端不发 SNI）时嗅探失败，不覆写；`M.IsDomainName` 是 `//go:linkname` 到 `net.isDomainName`
  （`sing@v0.9.7-…/common/metadata/domain.go`；实现在 Go 运行时 `net/dnsclient.go`），**只做 RFC1035 语法校验**：
  字符集 `a-zA-Z0-9_-`、标签 ≤63、总长 ≤254、无前导/连续点、末段非空。**无后缀黑名单**——`foo.onion` 会通过，
  `.onion` 是被"前导点"语法拒绝而非因 `onion`。嗅探出的 `metadata.Domain` 只要语法合法即覆写。
- 覆写点 3 处：`route.go:367`（`PreMatch`，UDP 首包）、`:754`（`actionSniff` 的 stream/TCP 分支，`PeekStream` 成功后）、
  `:878`（`actionSniff` 的 packet/UDP-QUIC 分支 `finally`）。即 **stream 与 packet 两条嗅探路径都覆写**；
  UDP 除 `actionSniff` 外**额外**还有 pre-match 首包路径（不是"只在 pre-match 生效"）。
- 覆写后 `metadata.Destination` 是域名：`IPCIDRItem.Match`（`route/rule/rule_item_cidr.go:77` 起）走
  「`Destination.IsIP()` 为假 → `len(DestinationAddresses)==0` → `return metadata.IPCIDRAcceptEmpty`」分支；
  `IPCIDRAcceptEmpty` 默认 `false`（`adapter/inbound.go:120` `ResetRuleCache` 内 `:122` 清零）。即**同一条规则里 `ip_cidr` 不再匹配该连接**，
  这是功能语义而非 bug，必须在文档写明；依赖 IP 分流的用户应在 sniff 规则**之前**用 IP 规则完成分流。

## 4. 构建（GitHub Actions）

### 4.1 产物清单（对齐官方 v1.14.2 release 的三个 asset，仅加 fork 标识）

| 产物 | 对应上游矩阵行 | CGO | 额外 tag | 附加文件 |
|---|---|---|---|---|
| `sing-box-<ver>-lightsb-linux-amd64.tar.gz` | `{os: linux, arch: amd64, variant: purego, naive: true}` | 0 | `,with_purego` | `libcronet.so` |
| `sing-box-<ver>-lightsb-linux-amd64-glibc.tar.gz` | `{os: linux, arch: amd64, variant: glibc, naive: true}` | 1 | — | — |
| `sing-box-<ver>-lightsb-linux-amd64-musl.tar.gz` | `{os: linux, arch: amd64, variant: musl, naive: true}` | 1 | `,with_musl` | — |

上游矩阵行号（`.github/workflows/build.yml:94-96`）与 `runs-on: ubuntu-26.04`（`:87`）、`go-version: 1.26.8`（`:148`）均照抄。

### 4.2 Workflow 契约

新建**独立** workflow `.github/workflows/build-lightsb.yml`。

- **触发：只写 `workflow_dispatch`**（输入 `version`，默认 `1.14.2`）+ `push: tags: ['v1.14.2-lightsb.*']`。
- **禁止** `push: branches` —— 公开 fork 上任何有写权限者的分支推送都会烧 Actions 额度，本仓库的构建只认自有 tag 与手动触发。
- runner `ubuntu-26.04`（与上游一致，保证 Chromium 工具链可用）；Go `1.26.8`。

上游 workflow 处置（当前 6 个：`build.yml` `docker.yml` `lint.yml` `linux.yml` `stale.yml` `test.yml`）：

| 上游文件 | 上游触发 | 处置 |
|---|---|---|
| `build.yml` | `push:[stable,testing,unstable]` 等 | **删除**（40+ 矩阵、跨平台打包，我们只出 3 个 amd64） |
| `linux.yml` | `release: published` | **删除**（会随我方 Release 触发） |
| `docker.yml` | `release: published` | **删除**（构建镜像推 ghcr） |
| `test.yml` | `push:[oldstable,stable,testing,unstable]` + `pull_request` 同分支 | **保留并改触发分支为 `lightsb`**；这是回归门。注意 matrix = 3 OS × 2 Go（`~1.25`/`~1.26`）共 6 job，unix 需 `-exec sudo`：建议精简为 `ubuntu-latest` × 单一 Go 以省额度 |
| `lint.yml` | 同上 | **删除**（10 平台 golangci-lint 很贵，本地 `go vet` 代替） |
| `stale.yml` | schedule | **删除** |

理由：上游 workflow 的 `push` 条件写的是上游分支名，我方分支叫 `lightsb` 本就不会命中；但 `release: published`
一类事件与分支无关，必须删。删除后我方只剩 `build-lightsb.yml` 与改写触发的 `test.yml`。

- tag 列表解析（真源是仓库内 `release/DEFAULT_BUILD_TAGS*`，照抄上游逻辑）：

```bash
# 三个产物都 naive: true → 都用 DEFAULT_BUILD_TAGS（含 with_naive_outbound）
TAGS=$(cat release/DEFAULT_BUILD_TAGS)
[ "$VARIANT" = purego ] && TAGS="$TAGS,with_purego"
[ "$VARIANT" = musl   ] && TAGS="$TAGS,with_musl"
LDFLAGS_SHARED=$(cat release/LDFLAGS)   # 必须原样使用，含 -checklinkname=0
```

- 编译命令（照抄上游形状，勿自创）：

```bash
go build -v -trimpath -o dist/sing-box -tags "$BUILD_TAGS" \
  -ldflags "-X 'github.com/sagernet/sing-box/constant.Version=${VERSION}' ${LDFLAGS_SHARED} -s -w -buildid=" \
  ./cmd/sing-box
```

- glibc/musl 行必须先准备 Chromium 工具链（否则 `with_naive_outbound` 的 CGO 链接失败）。**照抄上游
  `build.yml:180-216`**（`if: matrix.naive && matrix.variant != 'purego'`），版本固定用 `.github/CRONET_GO_VERSION`
  （当前 `0d28acc44093df24b2526dea3d6ffefd6b0a54f0`）：

```bash
CRONET_GO_VERSION=$(cat .github/CRONET_GO_VERSION)
git init ~/cronet-go && git -C ~/cronet-go remote add origin https://github.com/sagernet/cronet-go.git
git -C ~/cronet-go sparse-checkout set --no-cone '/*' '!/lib'
git -C ~/cronet-go fetch --depth=1 --filter=blob:none origin "$CRONET_GO_VERSION"
git -C ~/cronet-go checkout FETCH_HEAD
git -C ~/cronet-go submodule update --init --recursive --depth=1
rm -f ~/cronet-go/naiveproxy/src/build/linux/sysroot_scripts/keyring.gpg
(cd ~/cronet-go && GPG_TTY=/dev/null ./naiveproxy/src/build/linux/sysroot_scripts/generate_keyring.sh)
# musl 额外加 --libc=musl
(cd ~/cronet-go && go run ./cmd/build-naive --target=linux/amd64 download-toolchain)
(cd ~/cronet-go && go run ./cmd/build-naive --target=linux/amd64 env >> $GITHUB_ENV)
```

- purego 行必须从 cronet-go 稀疏检出 `lib/linux_amd64/libcronet.so` 一并打包（照抄 `build.yml:256-267`：
  只 sparse-checkout `"/lib/linux_amd64/libcronet.so"`，源版本同为 `CRONET_GO_VERSION`，然后 `cp` 到 `dist/`）。
- 打包：目录 `sing-box-${VERSION}-lightsb-linux-amd64[-glibc|-musl]/`，内含 `sing-box` + `LICENSE`（+ purego 的 `libcronet.so`），`tar -czvf`。
- 自动建 Release：job 需 `permissions: contents: write`；Release note **必须**含"非官方修改版，与 SagerNet 无关"（`AGENTS.md` C3）。
- **不要推送上游 tag**（fork 已继承 632 个上游 tag，无需再推；再推会污染 ref 命名空间）。

### 4.3 版本戳与命名（禁止依赖 `git describe`）

`make build` 默认走 `cmd/internal/read_tag`（`git describe --tags`），rebase 后会产生 `1.14.2-<shortsha>` 这类漂移版本号，
使产物名不可控。**本 fork 一律显式传版本**：`VERSION` 取 workflow 输入（默认 `1.14.2`），
产物名 = `sing-box-${VERSION}-lightsb-linux-amd64[-glibc|-musl].tar.gz`。
判定来源只看 `sing-box version` 的 `Revision:`（= fork 提交）、`Tags:`、`CGO:`。

### 4.4 可选精简模式

若不需要 NaiveProxy：从 tag 列表去掉 `with_naive_outbound` 并跳过工具链步骤 → CI 时间与体积大幅下降（purego 包不再含 `libcronet.so`）。
**默认不开**（要求与官方产物对齐）；开时必须在 Release note 注明 tag 列表已裁剪。

### 4.5 CI 成本预期

3 个 job 并行：purego 最快（CGO=0）；glibc/musl 各行需下载 Chromium 工具链，单行约 10–25 分钟，产物 30MB 左右（与官方一致）。
`ubuntu-26.04` 属较新 runner 标签，若账号不可用退回 `ubuntu-latest` 并自测工具链能否下载。
公开仓库 Actions 免费，但**默认只手动触发**。

## 5. 验收门

### 5.1 配置解析

```bash
sing-box check -c config-with-override.json
```

参考事实：未打补丁的官方 v1.14.2 对该配置在解析层报未知字段错误（`json: unknown field "override_destination"`；准确文案以实际运行 `sing-box check` 为准），补丁后必须通过。

**已实测（2026-10-01，本机）**：用 `TAGS=DEFAULT_BUILD_TAGS_OTHERS` 编译的二进制（`/tmp/sing-box-lightsb`，117 MB）——
- 含 `override_destination` 的配置：`check` **rc=0**；
- 负对照（同位置塞一个未知字段）：`check` 报 `route.rules[0].unknown_field_for_negative_control: json: unknown field "…"` **rc=1**，证明该门不是空转。
- `sing-box version` 输出 `Tags:` 与 `release/DEFAULT_BUILD_TAGS_OTHERS` 一致、`Revision: 9210529f…`、`CGO: enabled`、版本号 `unknown`（未传 `-X …constant.Version`，见 §4.3）。

### 5.2 行为回归测试（`test/sniff_override_test.go`，hermetic，不依赖外网/DNS）

**放 `test/` 模块**（`test/go.mod` 有 `replace github.com/sagernet/sing-box => ../`，自动带上补丁；根目录 `go test ./...` 不覆盖该模块，必须单独跑）：

```bash
cd test && go test -mod=mod -count=1 -v -run TestSniffOverrideDestination .
# -mod=mod 是必需的：v1.14.2 仓内的 test/go.mod 未随根 go.mod 更新（缺 sing/quic-go 等升级），
# 不加会报 "updates to go.mod needed; to update it: go mod tidy"。
# 该开关会改写 test/go.mod、test/go.sum —— 属白名单外文件，跑完必须 git checkout 还原；
# 若在 CI 里跑，改为先跑一次 go mod tidy 并把结果提交，或固定 GOFLAGS=-mod=mod。
```

**已实测（2026-10-01，本机，`go test -c` 后在无网络敏感路径下直跑二进制）**：

| 场景 | 结果 |
|---|---|
| 打过补丁（接线行存在） | `--- PASS: TestSniffOverrideDestination (0.38s)` |
| 注释掉 `route/rule/rule_action.go` 的 `OverrideDestination:` 接线行（字段/JSON 保留） | `--- FAIL … Received unexpected error: EOF`（握手被 `ip_cidr` reject 掐断） |
| 恢复接线行后重跑 | `--- PASS` |

即 fail-before → pass-after 双向对照成立（对照方式见下方判定）。
**端到端独立复核**（真实二进制 + JSON 配置，非 `option.Options` 程序化构造）：同机起 `openssl` 自签 `localhost` 证书的 TLS 服务 127.0.0.1:10002，
`mixed` 入口 10001 经 SOCKS5 连**字面 IP** 并对 SNI=`localhost` 握手，规则 `[sniff(tls), ip_cidr 127.0.0.1/32 → reject]`：

- `"override_destination": false` → 客户端 `SSLZeroReturnError`（TLS 被拒）；
- `"override_destination": true` → `OK b'pong'`（收到服务端返回字节）。

两个配置**仅这一个布尔不同**，因此覆盖链路的全部三层：JSON 解析 → `NewRuleAction` 接线 → `route.go` 覆写点。

⚠️ 实测踩坑：`reject` 在**程序化构造** `option.Options` 时 `Method` 为空串，而 `RuleActionReject.Error` 对空 `Method` 会 `panic: unknown reject method:`（默认值只在 `RejectActionOptions.UnmarshalJSON` 里填）。
测试里必须显式写 `RejectOptions: option.RejectActionOptions{Method: C.RuleActionRejectMethodDefault}`。

复用既有 helper（勿自造）：
- `startInstance(t, option.Options{...})`（`test/box_test.go:38`）
- `createSelfSignedCertificate(t, "localhost")`（`test/mkcert.go:22`）→ 返回三者的**文件路径**
  `(caPath, certPath, keyPath)` = `tempDir/ca.pem`、`tempDir/localhost.pem`、`tempDir/localhost.key.pem`；
  证书含 `DNSNames=["localhost"]`，用作 `InboundTLSOptions.CertificatePath`/`KeyPath`（**勿当 PEM 内容**）。
  测试自建 `tls.Listener` 时用 `tls.LoadX509KeyPair(certPath, keyPath)`。
- 端口常量（`test/shadowsocks_test.go:19-27`）：`serverPort/clientPort/testPort` = 10000/10001/10002。

用例结构：
1. 本地起 `tls.Listen("tcp", "127.0.0.1:testPort", ...)`，证书 DNSNames = `localhost`，握手成功后回写固定字节；
2. `startInstance`：`mixed` inbound（`clientPort`）→ outbound `direct`；
   规则顺序 `[{action:"sniff", override_destination:true}, {"ip_cidr":["127.0.0.1/32"], action:"reject"}]`（默认 direct）；
3. 客户端经 socks5 连 `127.0.0.1:testPort`（**字面 IP**）并对 SNI=`localhost` 握手。

判定：
- 补丁后：`override_destination:true` 生效 → `Destination` 变 `localhost:testPort` → `ip_cidr 127.0.0.1/32` 不再匹配（见 §3 语义边界）→ 不命中 reject → 落到默认 direct → TLS 握手成功。
- 可靠的 fail→pass 对照（同代码库内证明断言有效）：临时注释掉 §3 中 `route/rule/rule_action.go` 的
  `OverrideDestination: action.SniffOptions.OverrideDestination,` 赋值行（字段与 JSON 保留）重编译再跑本用例，应失败
  （`Destination` 仍 `127.0.0.1` → 命中 reject）；恢复该行后应通过。
- 旧 tag 上的"fail"表现为：`sing-box check`（§5.1）报未知字段；把本测试文件直接放到未打补丁的 v1.14.2 上则**编译失败**
  （`option.RouteActionSniff.OverrideDestination` 字段不存在），**不是**"连接被拒"。

只跑 `-run` 单个用例，避免拖起全套（部分用例需 `-exec sudo`/外网/docker；本用例都不需要）。

### 5.3 冒烟

本机实测 `MemTotal ≈ 5 GiB`（`/proc/meminfo` = 5242880 kB）、cgroup `memory.max = max`（**无硬限**），且本机**不是部署环境**。
仍按低内存纪律执行：重编译/全量测试前先 `free -h`，用 `systemd-run --user --scope -p MemoryMax=1G -- go build …` 包裹，
不与 dev server 并行；增量验证优先，全量构建放 CI/宿主机。

## 6. 跟随上游的流程

见 §2.5 的 rebase 命令。补充检查：

```bash
git diff <新tag> --stat                 # 只允许出现 AGENTS.md §C2 白名单文件
git diff <新tag> -- route/route.go      # 必须为空（消费逻辑零 diff）
go build ./cmd/sing-box && go test ./route/... ./option/...
```

预期冲突点（按概率排序）：
1. `option/rule_action.go` 的 `RouteActionSniff` struct（上游可能新增字段）；
2. `route/rule/rule_action.go` 的 `RuleActionTypeSniff` 分支；
3. `route/route.go` 的 3 处 `OverrideDestination` 消费点 —— 上游若删除该字段或重构 pre-match，本 fork 需**自己重写**等价覆写逻辑
   （这是唯一长期风险，不要只盯 struct 冲突）。

上游若真删掉 `RuleActionSniff.OverrideDestination`：在 `route/rule/rule_action.go` 自带该字段并由 `NewRuleAction` 赋值，
覆写动作在 `route.go` 对应位置补 3 行（与本次补丁同构）。**不要**把 `route.go` 的改动当常规 diff 保留，能零改动就零改动。

## 7. 当前状态

- [x] 已完成：fork 建仓（`kimberxu/lightsb`，`fork=true`、parent/source = `SagerNet/sing-box`、public、`has_issues=false`）；
  以 tag `v1.14.2` 复核全部基线事实（`route.go:367/754/878`、`option/rule_action.go:325`/`:180`、`rule/rule_action.go:107/510/516`、
  `.gitignore:21-22`、`schema.json` 0 命中、`Makefile` schema 目标、`release/DEFAULT_BUILD_TAGS*`/`LDFLAGS`、移除提交归属 v1.13.1/v1.13.4、
  上游 issue/PR 状态、`test.yml` 触发与 matrix、`build.yml` 工具链步骤与 `CRONET_GO_VERSION`、`ubuntu-26.04`/Go 1.26.8）；
  §3 补丁 `git apply --check` 通过（rc=0）；fork 文档改写为 lightsb（本文件 + `AGENTS.md`）。
- [x] 已完成（2026-10-01，实测）：建工作副本 `repo/`、以 `v1.14.2` 建分支 `lightsb`、提交并推送 `2994aedb docs: rewrite fork docs for lightsb`
  （remote blob 校验一致：`AGENTS.md 1a647e2e…`、`PLAN.md 10bc1b0b…`）；`PATCH default_branch=lightsb` 生效；
  删除继承的 39 个分支（`docs`、`dev-*`、`dependabot/*`、`renovate/*`、`copilot/*`、`testing`、`unstable`、`oldstable`、`archive`、
  `ccm-ocm-improvements`、`cloudflared`、`fix-acme-http-tls-challenge`、`draft-windows-auto-redirect`、`revert-4376-*`、`usbip`、`dev-ping` 等），
  现存 ref 仅 `lightsb`(2994aedb) + `stable`(777dec2b)，632 个上游 tag 保留未动。
- [x] 已完成（2026-10-01，实测）：落地 §3 补丁并提交 `8fcd24c7`——`option/rule_action.go`、`route/rule/rule_action.go`、
  `docs/configuration/route/rule_action{,.zh}.md`、`docs/schema.json`（`make schema` 重生成，diff 仅多 `override_destination: boolean`）、
  `test/sniff_override_test.go`。验证：`go build`/`go vet` 干净；`sing-box check` 含该字段 rc=0、负对照 rc=1；
  回归测试 PASS，且注释掉接线行后同用例 FAIL(EOF)——fail-before/pass-after 成立；另有真实二进制+JSON 的端到端对照（见 §5.2）。
- [x] 已完成（2026-10-01）：§4.2 CI 落地并提交 `8dd73986`——新增 `build-lightsb.yml`，删除上游 `build.yml`/`linux.yml`/`docker.yml`/
  `lint.yml`/`stale.yml`，`test.yml` 改为单 ubuntu job、触发分支 `lightsb`、并**补跑 `test/` 模块**（上游 `./...` 跨不过模块边界）。
  提交 `2994aedb`、`9210529f`、`8fcd24c7`、`8dd73986` 均已推送；`git diff v1.14.2 HEAD` 只落在 AGENTS.md §C2 白名单内，
  `route/route.go` 零 diff。
- [x] 已完成（2026-10-01，GitHub Actions 实测）：首次 CI 跑通并已发版。
  - `build-lightsb.yml` run `36800603882`（dispatch）**success**，三 job 全绿（purego/glibc/musl）。
  - 三产物下载核对：`sing-box version 1.14.2`、`Tags:` 与 `DEFAULT_BUILD_TAGS` 一致（purego/musl 各带 `with_purego`/`with_musl`）、
    `CGO` 分别 disabled/enabled/enabled、`Revision: 9cef805a`、`libcronet.so` 仅出现在 purego 包、每包含 `LICENSE`；
    `sing-box check` 含 `override_destination` 三产物均 rc=0，负对照均 rc=1。
  - 用**发布产物**重跑端到端：`override_destination=false` → TLS 被拒；`=true` → 收到 `pong`。
  - tag `v1.14.2-lightsb.1` → run `36803236718` **success**，Release 已发布（非 draft/prerelease），
    三个 asset 齐备，release note 含"非官方修改版，与 SagerNet 无关"；`v*-lightsb.*` 触发器工作正常（**未**打包：因产物命名不含 `-glibc`，
    `glibc` 那条 job 也产出了 `sing-box-1.14.2-lightsb-linux-amd64.tar.gz`，与上游 `-glibc` 命名不同，见 §4.1）。
  - `test.yml`（回归门）已由 `disabled_manually` 重新启用，run `36801368476` **success**。
- [x] 已完成（2026-10-01）：CI 首次运行暴露并修掉一个**真实缺陷**（提交 `464c452d`）——上游的 `test/` 模块其 `go.mod` 相对根 `go.mod` 已陈旧，
  带 `BUILD_TAGS`（`with_gvisor` 等）编译该模块会因 tailscale/gvisor API 不匹配直接失败；该模块上游从不测试（`go test ./...` 跨不过模块边界），
  故缺陷长期隐藏。回归用例不需要任何 tag，步骤已改为不带 tag 运行（`-mod=mod` 保留，理由见 §5.2）。
- [ ] 未完成 / 待决定：产物命名是否给 glibc 包也加 `-glibc` 后缀（当前与上游命名不一一对应）；
  `test/` 模块的 `go.mod` 陈旧属上游缺陷，本 fork 未修（修会越出 C2 白名单，除 `test/sniff_override_test.go` 外的 test 文件不可改）。
