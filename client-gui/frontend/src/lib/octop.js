// Octop API 封装 — 以 wailsjs/go/main/App.d.ts 生成的真实签名为准
const A = () => window.go?.main?.App

export const api = {
  login: (srv, user, pass) => A().Login(srv, user, pass),
  ask: (expert, thread, text) => A().Ask(expert || '', thread || '', text),
  resume: (expert, thread, msg) => A().Resume(expert || '', thread || '', msg),
  listExperts: () => A().ListExperts(),
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
  // 专家实例管理(官方 /api/agents)
  agentsManage: () => A().AgentsManage(),
  agentCreate: (name, desc, prompt, welcome, shared) => A().AgentCreate(name, desc, prompt, welcome, !!shared),
  agentUpdate: (id, fields) => A().AgentUpdate(id, JSON.stringify(fields)),
  agentToggle: (id, action) => A().AgentToggle(id, action),
  agentDelete: (id) => A().AgentDelete(id),
  // 技能库(官方 /api/agents/{id}/skills)
  skillList: (agentID) => A().SkillList(agentID || 'main'),
  skillCreate: (agentID, slug, desc, body) => A().SkillCreate(agentID, slug, desc, body),
  skillDelete: (agentID, slug) => A().SkillDelete(agentID, slug),
  // A档授权目录
  allowedList: () => A().AllowedList(),
  allowedAdd: (dir) => A().AllowedAdd(dir),
  allowedRemove: (dir) => A().AllowedRemove(dir),
}
