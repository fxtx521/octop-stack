// Octop GUI 前端逻辑
let currentOps = null; // 待审批的ops

function el(id) { return document.getElementById(id); }
function addBubble(cls, text) {
  const d = document.createElement('div');
  d.className = 'bubble ' + cls;
  d.textContent = text;
  el('chat').appendChild(d);
  el('chat').scrollTop = el('chat').scrollHeight;
  return d;
}

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
  el('expertName').textContent = r.experts && r.experts.length ? '专家: ' + r.experts[0].name : '';
  el('usage').textContent = r.usage ? r.usage.split('\n')[0] : '';
  addBubble('sys', '登录成功。选择项目目录后, AI 就能读写里面的文件。');
}

async function pickWS() {
  const dir = await window.go.main.App.PickDirectory();
  if (!dir) return;
  const r = await window.go.main.App.SetWorkspace(dir);
  if (!r.ok) { addBubble('sys', '目录设置失败: ' + r.msg); return; }
  el('wsName').textContent = dir.split('/').pop();
  refreshFiles();
  addBubble('sys', '项目目录: ' + dir + ' (共 ' + r.manifest.split('\n').length + ' 行清单)');
}

async function refreshFiles() {
  const files = await window.go.main.App.ListFiles();
  el('files').innerHTML = files.map(f => '📄 ' + f).join('<br>') || '(空目录)';
}

async function send() {
  const t = el('msg').value.trim();
  if (!t) return;
  el('msg').value = '';
  addBubble('user', t);
  addBubble('ai', '思考中…');
  const r = await window.go.main.App.Ask(t);
  el('chat').lastChild.remove(); // 移除思考中
  if (!r.ok) { addBubble('ai', '❌ ' + r.msg); return; }
  handleAnswer(r);
}

function handleAnswer(r) {
  // 去掉fileops代码块原文, 只展示说明文字
  const cleaned = r.answer.replace(/```fileops[\s\S]*?```/g, '').trim();
  if (cleaned) addBubble('ai', cleaned);
  if (r.ops && r.ops.length) {
    showApprove(r.ops);
  }
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

async function applyOps(approved, card) {
  if (!currentOps) return;
  card.querySelectorAll('button').forEach(b => b.remove());
  const r = await window.go.main.App.Apply(JSON.stringify(currentOps), approved);
  currentOps = null;
  const d = document.createElement('div');
  d.className = 'bubble ai';
  // 结果行用DOM构造, 不拼HTML(AI/路径内容不可信)
  r.results.forEach(x => {
    const s = document.createElement('span');
    s.className = 'res ' + x.status;
    const icon = x.status === 'done' ? '✅' : x.status === 'rejected' ? '⛔' : '❌';
    s.textContent = icon + ' [' + x.op + '] ' + x.path + ' — ' + x.detail;
    d.appendChild(s);
  });
  el('chat').appendChild(d);
  el('chat').scrollTop = el('chat').scrollHeight;
  refreshFiles();
  // read有内容→自动喂回专家续答
  if (approved && r.feedback) {
    addBubble('ai', '读取完成, 继续处理…');
    const r2 = await window.go.main.App.Resume('[客户端执行结果] 你请求的read已完成, 内容如下, 请继续你的任务(如需写文件继续用fileops):' + r.feedback);
    if (r2.ok) handleAnswer(r2);
    else addBubble('ai', '❌ 续答失败: ' + r2.msg);
  }
}

// ES module作用域: inline onclick找不到函数, 显式挂window
window.doLogin = doLogin;
window.pickWS = pickWS;
window.send = send;
window.applyOps = applyOps;
