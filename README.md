# octop-stack

**English** | [中文](./README_CN.md)

Enterprise value layer built on top of [TencentCloud/Octop](https://github.com/TencentCloud/Octop): a thin desktop client (macOS / Windows) for local-folder AI operations, plus cloud-side management tooling for onboarding, permissions and billing.

**Octop stays stock.** This repo never modifies Octop source code — it only calls official APIs (REST + MCP gateway). Official upgrades never break it. (Integrity verified: 2277 files, 0 modified.)

## What's inside

- `云端管控实测/` — closed-loop test of pure-API cloud control: invite-code onboarding, custom role with least privilege, per-user usage accounting, admin endpoints blocked for staff. 9/10 checks passed on an isolated instance (127.0.0.1:8099), production untouched.
- `client/` — Go thin client: log in to your Octop cloud, list experts, ask questions. Same codebase builds for macOS and Windows.

## Principles

1. Never modify upstream source — API/CLI only, verified by RECORD hash audit.
2. Every feature ships with an end-to-end verification run.
3. Tests run on isolated instances; production data never touched.

## Roadmap

- [x] Cloud control closed loop (invite → role → call AI → per-user billing)
- [x] Thin client v0.1: login + expert call (macOS)
- [ ] Local folder operations (sandboxed workspace)
- [ ] Windows build + packaging
- [ ] Production cloud deployment (VPS + TLS)

## License

Apache-2.0
