# lightsb 约束（必须遵从）

本仓库是 [SagerNet/sing-box](https://github.com/SagerNet/sing-box) 的 **stable fork**（GitHub 原生 fork：`kimberxu/lightsb`），
唯一功能改动：恢复「嗅探到的域名覆写连接目标地址」，通过 `action: sniff` 的 `override_destination` 字段实现。

本文件只放**必须遵从的约束、禁止事项、许可边界**。执行计划、命令、步骤、上游事实基线、当前进度在
`PLAN.md`（同目录），两者不重复。

## C1 身份与范围
- 非官方修改版：与 SagerNet 无关；公开分发规则见 C3。
- 只跟随上游 **stable 发行版**；基线锚定最新稳定 tag（取值见 `PLAN.md` §1），**不用** `stable` 分支 HEAD。
- 改动前先复核 `PLAN.md` §1 表中的行号是否漂移（那些行号按基线 tag 记录）。
- 不新增第二个功能；不"顺手"改其它文件。
- 不把 upstream `testing`/`unstable` 合入 fork。
- 不恢复 legacy `inbound.sniff_override_destination` 字段。
- 不向上游提 issue/PR 索要该功能（维护者已明确拒绝，历史见 `PLAN.md` §1）。
- 不触碰 fork 网络关系：不向 `SagerNet/sing-box` 推任何 ref；`upstream` remote **只读**（仅 fetch / rebase 锚点）。

## C2 改动范围白名单
只允许改动下列文件（新增文件同样算改动）：

| 文件 | 允许的改动 |
|---|---|
| `option/rule_action.go` | 加 `OverrideDestination bool` JSON 字段 |
| `route/rule/rule_action.go` | `NewRuleAction` 接线 + 改掉 `// Deprecated` 注释 |
| `docs/configuration/route/rule_action.md` | 新增 `override_destination` 说明（标注 fork 特有） |
| `docs/configuration/route/rule_action.zh.md` | 同上（中文） |
| `docs/schema.json` | 由 `make schema` 重新生成，**禁止手工编辑** |
| `test/sniff_override_test.go` | 新增回归测试（用例见 `PLAN.md` §5.2） |
| `.github/workflows/*` | fork 自己的 CI：新增 `build-lightsb.yml`；上游 workflow 按 `PLAN.md` §4.2 处置 |
| `AGENTS.md`、`PLAN.md`、`.gitignore` | fork 基础设施 |

- 补丁内容以 `PLAN.md` §3 为唯一真源；验收标准以 `PLAN.md` §5 为唯一真源。
- **禁止改动**：`route/route.go` 的 3 处消费逻辑（与上游保持零 diff）、`option/inbound.go` 的 legacy 字段、
  `release/*`、`constant/*`，以及白名单外任何其它源码/CI/文档（不"顺手清理"上游文件）。

## C3 许可、分发与环境
- 上游 `LICENSE` 为 GPLv3+，附加条款：`no derivative work may use the name or imply association with this application without prior consent`（原文见仓根 `README.md`）。
- **公开**发布二进制/Release：必须改名（如 `sing-box-1.14.2-lightsb-linux-amd64.tar.gz`，不得使用上游命名与上游 Release 页面）、
  显式声明"非官方修改版，与 SagerNet 无关"、随包附 `LICENSE`。自用/内网分发可保持官方式文件命名。
- 不推送上游 tag；出包只用自有 tag `v1.14.2-lightsb.N`。工作分支 `lightsb` 与 fork 上继承自上游的 tag 是两回事，
  不得用上游 tag（如 `v1.14.2`）在本 fork 创建 Release。
- 改动/rebase 后**必须**重跑 `PLAN.md` §5 验收门（GitHub Actions 不是调试环境）。
- 本机实测 ≈5 GiB 内存、cgroup `memory.max = max`（无硬限，见 `PLAN.md` §5.3）；**仍**按低内存纪律执行：
  重编译/全量测试前先 `free -h`，用 `systemd-run --user --scope -p MemoryMax=1G -- <cmd>` 包裹，
  且不与 dev server 并行；本机不是部署环境，任何"跑通"结论以实际命令输出为准。
