// Octop GUI 前端逻辑 — WorkBuddy 风格三栏
let currentOps = null; // 待审批的ops
let tasks = [];        // {id, title, threadId, at}
let curTask = null;

function el(id) { return document.getElementById(id); }
function addBubble(cls, text) {
  const d = document.createElement('div');
  d.className = 'bubble ' + cls;
  d.textContent = text;
  el('chat').appendChild(d);
  el('chat').scrollTop = el('chat').scrollHeight;
  return d;
}

/* ---------- 任务列表(localStorage) ---------- */
function loadTasks() {
  try { tasks = JSON.parse(localStorage.getItem('octop_tasks') || '[]'); } catch { tasks = []; }
  renderTasks();
}
function saveTasks() { localStorage.setItem('octop_tasks', JSON.stringify(tasks.slice(0, 50))); }
function newTask() {
  curTask = { id: Date.now(), title: '新任务', threadId: '', at: Date.now() };
  tasks.unshift(curTask); saveTasks(); renderTasks();
  el('taskTitle').textContent = '新任务';
  el('chat').innerHTML = '';
  addBubble('sys', '新任务开始。选择项目目录后, AI 就能读写里面的文件。');
  switchTab('files');
}
function renderTasks() {
  const box = el('taskList');
  box.textContent = '';
  tasks.forEach(t => {
    const d = document.createElement('div');
    d.className = 'task-item' + (curTask && t.id === curTask.id ? ' active' : '');
    d.textContent = t.title;
    d.onclick = () => switchTask(t);
    box.appendChild(d);
  });
}
function switchTask(t) {
  curTask = t; renderTasks();
  el('taskTitle').textContent = t.title;
  addBubble('sys', '已切换到「' + t.title + '」(会话历史在服务端, 继续聊会接上下文)');
}
function renameTask(title) {
  if (!curTask) return;
  curTask.title = title.slice(0, 24) || '新任务';
  el('taskTitle').textContent = curTask.title;
  saveTasks(); renderTasks();
}

/* ---------- 登录 ---------- */
async function doLogin() {
  const srv = el('srv').value.trim();
  const user = el('user').value.trim();
  const pass = el('pass').value;
  if (!user || !pass) { el('loginErr').textContent = '请输入账号和密码'; return; }
  el('btnLogin').textContent = '登录中...';
  const r = await window.go.main.App.Login(srv, user, pass);
  el('btnLogin').textContent = '登 录';
  if (!r.ok) { el('loginErr').textContent = r.msg; return; }
  el('login').classList.add('hidden');
  el('main').classList.remove('hidden');
  el('whoami').textContent = user + (r.role === 'admin' ? ' · 管理员' : '');
  el('expertName').textContent = r.experts && r.experts.length ? '专家: ' + r.experts[0].name : '';
  el('usage').textContent = r.usage ? r.usage.split('\n')[0] : '';
  if (r.role === 'admin') {
    el('adminBtn').classList.remove('hidden');
    el('adminTab').classList.remove('hidden');
  }
  loadTasks();
  if (!tasks.length) newTask();
  else { curTask = tasks[0]; renderTasks(); el('taskTitle').textContent = curTask.title; }
  addBubble('sys', '登录成功。选择项目目录后, AI 就能读写里面的文件。');
}
function logout() { location.reload(); }

/* ---------- 工作区 ---------- */
async function pickWS() {
  const dir = await window.go.main.App.PickDirectory();
  if (!dir) return;
  const r = await window.go.main.App.SetWorkspace(dir);
  if (!r.ok) { addBubble('sys', '目录设置失败: ' + r.msg); return; }
  el('wsName').textContent = dir.split('/').pop();
  refreshFiles();
  addBubble('sys', '项目目录: ' + dir);
}

async function refreshFiles() {
  const files = await window.go.main.App.ListFiles();
  const box = el('files');
  box.textContent = '';
  (files || []).forEach(f => {
    const d = document.createElement('div');
    d.className = 'file-item';
    d.textContent = '📄 ' + f;
    d.onclick = () => previewFile(f);
    box.appendChild(d);
  });
  if (!files || !files.length) box.textContent = '(空目录)';
}

// 点击文件 → 预览内容(老板要的"点得开")
async function previewFile(f) {
  // ListFiles 返回 "name (sizeB)" 格式, 剥掉大小后缀才是真实相对路径
  const rel = f.replace(/\s*\(\d+B\)$/, '');
  el('pvName').textContent = rel;
  el('preview').textContent = '读取中…';
  const r = await window.go.main.App.ReadFile(rel);
  el('preview').textContent = r.ok ? r.content : ('读取失败: ' + r.msg);
}

/* ---------- 对话 ---------- */
async function send() {
  const t = el('msg').value.trim();
  if (!t) return;
  el('msg').value = '';
  addBubble('user', t);
  if (curTask && curTask.title === '新任务') renameTask(t);
  addBubble('ai', '思考中…');
  const r = await window.go.main.App.Ask(t);
  el('chat').lastChild.remove();
  if (!r.ok) { addBubble('ai', '❌ ' + r.msg); return; }
  handleAnswer(r);
}

function handleAnswer(r) {
  const cleaned = (r.answer || '').replace(/```fileops[\s\S]*?```/g, '').trim();
  if (cleaned) addBubble('ai', cleaned);
  if (r.ops && r.ops.length) showApprove(r.ops);
}

function showApprove(ops) {
  currentOps = ops;
  const card = document.createElement('div');
  card.className = 'approve';
  const title = document.createElement('b');
  title.textContent = 'AI 请求执行以下操作:';
  card.appendChild(title);
  ops.forEach(o => {
    const op = document.createElement('div');
    op.className = 'op';
    op.textContent = '· [' + o.op + '] ' + o.path + (o.to ? ' → ' + o.to : '');
    card.appendChild(op);
    const re = document.createElement('div');
    re.className = 'reason';
    re.textContent = o.reason || '';
    card.appendChild(re);
  });
  const ok = document.createElement('button');
  ok.className = 'btn-ok'; ok.textContent = '批准执行'; ok.onclick = () => applyOps(true, card);
  const no = document.createElement('button');
  no.className = 'btn-no'; no.textContent = '拒绝'; no.onclick = () => applyOps(false, card);
  card.appendChild(ok); card.appendChild(no);
  el('chat').appendChild(card);
  el('chat').scrollTop = el('chat').scrollHeight;
}

function addChange(text, cls) {
  const box = el('changes');
  if (box.classList.contains('muted')) { box.textContent = ''; box.classList.remove('muted'); }
  const d = document.createElement('div');
  d.className = 'chg';
  d.textContent = new Date().toLocaleTimeString() + '  ' + text;
  if (cls) d.style.color = cls;
  box.prepend(d);
}

async function applyOps(approved, card) {
  if (!currentOps) return;
  card.querySelectorAll('button').forEach(b => b.remove());
  const r = await window.go.main.App.Apply(JSON.stringify(currentOps), approved);
  currentOps = null;
  const d = document.createElement('div');
  d.className = 'bubble ai';
  r.results.forEach(x => {
    const s = document.createElement('span');
    s.className = 'res ' + x.status;
    const icon = x.status === 'done' ? '✅' : x.status === 'rejected' ? '⛔' : '❌';
    s.textContent = icon + ' [' + x.op + '] ' + x.path + ' — ' + x.detail;
    d.appendChild(s);
    addChange('[' + x.op + '] ' + x.path + ' — ' + x.detail,
      x.status === 'done' ? '#1a9e55' : '#e5484d');
  });
  el('chat').appendChild(d);
  el('chat').scrollTop = el('chat').scrollHeight;
  refreshFiles();
  if (approved && r.feedback) {
    addBubble('ai', '读取完成, 继续处理…');
    const r2 = await window.go.main.App.Resume('[客户端执行结果] 你请求的read已完成, 内容如下, 请继续你的任务(如需写文件继续用fileops):' + r.feedback);
    if (r2.ok) handleAnswer(r2);
    else addBubble('ai', '❌ 续答失败: ' + r2.msg);
  }
}

/* ---------- Tab 切换 ---------- */
function switchTab(name) {
  document.querySelectorAll('.tab').forEach(b => b.classList.toggle('active', b.dataset.tab === name));
  ['files', 'changes', 'admin'].forEach(t =>
    el('tab-' + t).classList.toggle('hidden', t !== name));
  if (name === 'admin') loadAdmin();
}

/* ---------- 管理后台(admin) ---------- */
async function openAdmin() { switchTab('admin'); }

async function loadAdmin() {
  const pane = el('adminPane');
  pane.classList.remove('muted');
  pane.textContent = '加载中…';
  const [ov, mt] = await Promise.all([
    window.go.main.App.AdminOverview(),
    window.go.main.App.AdminMetrics()
  ]);
  if (!ov.ok) { pane.textContent = '加载失败: ' + ov.msg; return; }
  pane.textContent = '';
  const h = document.createElement('h4'); h.textContent = '📊 运行指标'; pane.appendChild(h);
  const m = mt.ok ? mt.data : {};
  [['消息总数', m.messages_total], ['流错误', m.stream_errors_total],
   ['在线Agent', m.agent_active], ['定时任务', m.cron_runs_total]].forEach(([k, v]) => {
    const d = document.createElement('div'); d.className = 'kv';
    d.textContent = k; const s = document.createElement('b'); s.textContent = v ?? '-';
    d.appendChild(s); pane.appendChild(d);
  });
  const h2 = document.createElement('h4'); h2.textContent = '👥 用户 (' + (ov.data.totals ? ov.data.totals.users : '?') + ')'; pane.appendChild(h2);
  (ov.data.users || []).forEach(u => {
    const d = document.createElement('div'); d.className = 'kv';
    d.textContent = u.username;
    const s = document.createElement('span'); s.className = 'muted';
    s.textContent = u.role === 'admin' ? '管理员' : '员工';
    d.appendChild(s); pane.appendChild(d);
  });
  const h3 = document.createElement('h4'); h3.textContent = '🤖 专家 (' + (ov.data.totals ? ov.data.totals.agents : '?') + ')'; pane.appendChild(h3);
  (ov.data.agents || []).forEach(a => {
    const d = document.createElement('div'); d.className = 'kv';
    d.textContent = a.name;
    const s = document.createElement('span'); s.className = 'muted';
    s.textContent = a.state === 'running' ? '✅ 运行中' : a.state;
    d.appendChild(s); pane.appendChild(d);
  });
  const h4 = document.createElement('h4'); h4.textContent = '🎫 开号邀请码'; pane.appendChild(h4);
  const btnRow = document.createElement('div'); btnRow.style.margin = '6px 0';
  const bStaff = document.createElement('button'); bStaff.className = 'mini-btn'; bStaff.textContent = '+ 员工码';
  const bAdmin = document.createElement('button'); bAdmin.className = 'mini-btn ghosty'; bAdmin.textContent = '+ 管理员码';
  bStaff.onclick = () => createInvite('staff');
  bAdmin.onclick = () => createInvite('admin');
  btnRow.appendChild(bStaff); btnRow.appendChild(bAdmin); pane.appendChild(btnRow);
  const inv = await window.go.main.App.AdminInvites();
  (inv.ok && inv.data._list ? JSON.parse(inv.data._list) : []).slice(0, 8).forEach(i => {
    const d = document.createElement('div'); d.className = 'kv';
    d.textContent = i.code;
    const s = document.createElement('span'); s.className = 'muted';
    s.textContent = i.status === 'used' ? '已用' : i.status === 'revoked' ? '已撤销' : '可用';
    d.appendChild(s); pane.appendChild(d);
  });
}

async function createInvite(role) {
  const r = await window.go.main.App.AdminCreateInvite(role, '工作台开号');
  if (!r.ok) { alert('生成失败: ' + r.msg); return; }
  loadAdmin();
}

// ES module作用域: inline onclick找不到函数, 显式挂window
window.doLogin = doLogin;
window.pickWS = pickWS;
window.send = send;
window.applyOps = applyOps;
window.newTask = newTask;
window.switchTab = switchTab;
window.openAdmin = openAdmin;
window.logout = logout;
