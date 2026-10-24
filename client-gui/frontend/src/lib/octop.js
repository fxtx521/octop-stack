// Octop API 封装 — 以 wailsjs/go/main/App.d.ts 生成的真实签名为准
const A = () => window.go?.main?.App

export const api = {
  login: (srv, user, pass) => A().Login(srv, user, pass),
  ask: (text) => A().Ask(text),
  resume: (msg) => A().Resume(msg),
  tryResume: () => A().TryResume(),
  clearSession: () => A().ClearSession(),
  apply: (ops, approved) => A().Apply(JSON.stringify(ops), approved),
  listFiles: () => A().ListFiles(),
  readFile: (rel) => A().ReadFile(rel),
  setWorkspace: (root) => A().SetWorkspace(root),
  startupWorkspace: () => A().StartupWorkspace(),
  currentWorkspace: () => A().CurrentWorkspace(),
  pickDirectory: () => A().PickDirectory(),
  adminOverview: () => A().AdminOverview(),
  adminMetrics: () => A().AdminMetrics(),
  adminInvites: () => A().AdminInvites(),
  adminAudit: () => A().AdminAudit(),
  adminCreateInvite: (role, note) => A().AdminCreateInvite(role, note),
}
