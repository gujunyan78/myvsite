(function () {
  'use strict';
  const API = '/api';
  const $ = (s, r) => (r || document).querySelector(s);
  const $$ = (s, r) => Array.prototype.slice.call((r || document).querySelectorAll(s));

  let token = localStorage.getItem('vsite_token') || '';
  let me = null;

  function toast(msg) {
    const t = $('#toast'); t.textContent = msg; t.hidden = false;
    clearTimeout(toast._t); toast._t = setTimeout(() => { t.hidden = true; }, 1800);
  }
  function esc(s) { return String(s == null ? '' : s).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }
  const fmt = (n) => { n = Number(n) || 0; if (n >= 1e9) return (n / 1e9).toFixed(1) + 'B'; if (n >= 1e6) return (n / 1e6).toFixed(1) + 'M'; if (n >= 1e3) return (n / 1e3).toFixed(1) + 'K'; return String(n); };

  async function api(path, opts) {
    opts = opts || {};
    // FormData 必须由浏览器自动带 multipart boundary，不能强制 JSON
    const isForm = typeof FormData !== 'undefined' && opts.body instanceof FormData;
    const headers = Object.assign(isForm ? {} : { 'Content-Type': 'application/json' }, opts.headers || {});
    if (token) headers['X-Admin-Token'] = token;
    const res = await fetch(API + path, Object.assign({ headers }, opts));
    const data = await res.json().catch(() => ({}));
    if (!res.ok || (data.code && data.code !== 0)) {
      const err = new Error(data.message || 'Request failed');
      if (res.status === 401) { token = ''; localStorage.removeItem('vsite_token'); }
      throw err;
    }
    return data.data;
  }

  // ---------- Login ----------
  $('#login-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    try {
      const d = await api('/admin/login', {
        method: 'POST',
        body: JSON.stringify({ username: $('#login-user').value.trim(), password: $('#login-pass').value })
      });
      token = d.token; localStorage.setItem('vsite_token', token);
      if (d.must_change_password) return showForce(d.username);
      enterAdmin();
    } catch (err) { toast(err.message); }
  });

  function showForce(username) {
    $('#login-page').hidden = true;
    $('#admin-app').hidden = true;
    $('#force-page').hidden = false;
    $('#force-user').value = username || '';
    $('#force-old').value = '';
    $('#force-new').value = '';
  }
  $('#force-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const np = $('#force-new').value;
    if (np.length < 6) { toast('New password must be at least 6 characters'); return; }
    try {
      const d = await api('/admin/password', {
        method: 'POST',
        body: JSON.stringify({ old_password: $('#force-old').value, new_password: np, username: $('#force-user').value.trim() })
      });
      token = ''; localStorage.removeItem('vsite_token');
      toast('Password changed. Please sign in again.');
      $('#force-page').hidden = true;
      $('#login-page').hidden = false;
      $('#login-user').value = d.username || '';
      $('#login-pass').value = '';
      $('#login-pass').focus();
    } catch (err) { toast(err.message); }
  });

  async function enterAdmin() {
    $('#login-page').hidden = true;
    $('#force-page').hidden = true;
    $('#admin-app').hidden = false;
    try { me = await api('/admin/profile'); $('#who').textContent = 'Signed in as: ' + me.username; }
    catch (e) { me = null; }
    switchTab('overview');
    loadBloggersSelect();
    loadBloggers();
    loadVideos();
    loadComments();
  }

  $('#logout-btn').addEventListener('click', async () => {
    try { await api('/admin/logout', { method: 'POST' }); } catch (e) {}
    token = ''; localStorage.removeItem('vsite_token');
    $('#admin-app').hidden = true; $('#login-page').hidden = false;
  });

  // ---------- Tabs ----------
  function switchTab(name) {
    $$('.sidebar nav a').forEach((a) => a.classList.toggle('active', a.dataset.tab === name));
    $$('.tab').forEach((t) => { t.hidden = (t.id !== 'tab-' + name); });
    $('#page-title').textContent = { overview: 'Overview', bloggers: 'Creators', videos: 'Videos', comments: 'Comments' }[name];
    $('#head-actions').innerHTML = '';
    if (name === 'overview') loadStats();
  }
  $$('.sidebar nav a').forEach((a) => a.addEventListener('click', () => switchTab(a.dataset.tab)));

  // ---------- Overview ----------
  async function loadStats() {
    try {
      const s = await api('/admin/stats');
      const grid = $('#stat-grid');
      grid.innerHTML = '';
      [['Total videos', s.videos], ['Creators', s.bloggers], ['Total views', s.views], ['Total likes', s.likes], ['Comments', s.comments]]
        .forEach(([k, v]) => {
          const d = document.createElement('div'); d.className = 'stat';
          d.innerHTML = '<b>' + fmt(v) + '</b><span>' + k + '</span>';
          grid.appendChild(d);
        });
    } catch (e) { toast(e.message); }
  }

  // ---------- Creators ----------
  let bPage = 1;
  async function loadBloggers() {
    const kw = $('#blogger-kw').value.trim();
    const data = await api('/admin/bloggers?page=' + bPage + '&page_size=10&keyword=' + encodeURIComponent(kw)).catch(() => ({ items: [], total: 0 }));
    const tb = $('#blogger-table tbody'); tb.innerHTML = '';
    (data.items || []).forEach((b) => {
      const tr = document.createElement('tr');
      tr.innerHTML = `<td>${b.id}</td>
        <td><img class="avatar-sm" src="${b.avatar || ''}" onerror="this.style.visibility='hidden'"></td>
        <td>${esc(b.account)}</td>
        <td style="max-width:240px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">${esc(b.description)}</td>
        <td>${b.video_count}</td>
        <td><span class="tag ${b.status ? 'on' : 'off'}">${b.status ? 'Active' : 'Hidden'}</span></td>
        <td><div class="row-btns">
          <button class="btn sm" data-edit="${b.id}">Edit</button>
          <button class="btn sm ghost link-danger" data-del="${b.id}">Delete</button>
        </div></td>`;
      tb.appendChild(tr);
    });
    renderPager('#blogger-pager', data.total, bPage, 10, (p) => { bPage = p; loadBloggers(); });
  }
  $('#blogger-kw').addEventListener('input', debounce(() => { bPage = 1; loadBloggers(); }, 350));
  $('#blogger-table').addEventListener('click', async (e) => {
    const ed = e.target.dataset.edit, del = e.target.dataset.del;
    if (ed) openBloggerModal(parseInt(ed, 10));
    if (del) {
      if (!confirm('Delete this creator and all of their videos?')) return;
      try { await api('/admin/bloggers/' + del, { method: 'DELETE' }); toast('Deleted'); loadBloggers(); loadVideos(); }
      catch (err) { toast(err.message); }
    }
  });

  function openBloggerModal(id) {
    $('#blogger-modal-title').textContent = id ? 'Edit creator' : 'New creator';
    if (id) {
      api('/bloggers/' + id).then((b) => {
        $('#blogger-id').value = b.id; $('#blogger-account').value = b.account;
        $('#blogger-desc').value = b.description; $('#blogger-avatar').value = b.avatar; $('#blogger-status').value = String(b.status);
      }).catch((e) => toast(e.message));
    } else {
      $('#blogger-form').reset(); $('#blogger-id').value = ''; $('#blogger-status').value = '1';
    }
    $('#blogger-modal').hidden = false;
  }
  $('#blogger-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const id = $('#blogger-id').value;
    const body = { account: $('#blogger-account').value.trim(), description: $('#blogger-desc').value.trim(), avatar: $('#blogger-avatar').value.trim(), status: parseInt($('#blogger-status').value, 10) };
    if (!body.account) { toast('Creator account is required'); return; }
    try {
      if (id) await api('/admin/bloggers/' + id, { method: 'PUT', body: JSON.stringify(body) });
      else await api('/admin/bloggers', { method: 'POST', body: JSON.stringify(body) });
      toast('Saved'); $('#blogger-modal').hidden = true; loadBloggers(); loadVideos();
    } catch (err) { toast(err.message); }
  });

  // ---------- Videos ----------
  let vPage = 1;
  async function loadBloggersSelect() {
    const data = await api('/bloggers?page_size=100').catch(() => ({ items: [] }));
    const sel = $('#video-blogger-id');
    sel.innerHTML = '<option value="">Select a creator</option>';
    const filterSel = $('#video-blogger');
    filterSel.innerHTML = '<option value="">All creators</option>';
    const cSel = $('#comment-video');
    cSel.innerHTML = '<option value="">All creators</option>';
    (data.items || []).forEach((b) => {
      sel.insertAdjacentHTML('beforeend', `<option value="${b.id}">${esc(b.account)}</option>`);
      filterSel.insertAdjacentHTML('beforeend', `<option value="${b.id}">${esc(b.account)}</option>`);
      cSel.insertAdjacentHTML('beforeend', `<option value="${b.id}">${esc(b.account)}</option>`);
    });
  }
  async function loadVideos() {
    const kw = $('#video-kw').value.trim();
    const bid = $('#video-blogger').value;
    const q = `page=${vPage}&page_size=10&keyword=${encodeURIComponent(kw)}${bid ? '&blogger_id=' + bid : ''}`;
    const data = await api('/admin/videos?' + q).catch(() => ({ items: [], total: 0 }));
    const tb = $('#video-table tbody'); tb.innerHTML = '';
    (data.items || []).forEach((v) => {
      const tr = document.createElement('tr');
      tr.innerHTML = `<td>${v.id}</td>
        <td><img class="thumb" src="${v.cover_url || ''}" onerror="this.style.visibility='hidden'"></td>
        <td style="max-width:220px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">${esc(v.title)}</td>
        <td>${esc(v.blogger ? v.blogger.account : '-')}</td>
        <td>${fmt(v.likes)} / ${fmt(v.views)}</td>
        <td>${v.comments}</td>
        <td><span class="tag ${v.status ? 'on' : 'off'}">${v.status ? 'Online' : 'Offline'}</span></td>
        <td><div class="row-btns">
          <button class="btn sm" data-edit="${v.id}">Edit</button>
          <button class="btn sm ghost" data-cmt="${v.blogger_id}">Comments</button>
          <button class="btn sm ghost link-danger" data-del="${v.id}">Delete</button>
        </div></td>`;
      tb.appendChild(tr);
    });
    renderPager('#video-pager', data.total, vPage, 10, (p) => { vPage = p; loadVideos(); });
  }
  $('#video-kw').addEventListener('input', debounce(() => { vPage = 1; loadVideos(); }, 350));
  $('#video-blogger').addEventListener('change', () => { vPage = 1; loadVideos(); });
  $('#video-table').addEventListener('click', async (e) => {
    const ed = e.target.dataset.edit, del = e.target.dataset.del, cmt = e.target.dataset.cmt;
    if (ed) openVideoModal(parseInt(ed, 10));
    if (cmt) { switchTab('comments'); $('#comment-video').value = cmt; loadComments(); }
    if (del) {
      if (!confirm('Delete this video?')) return;
      try { await api('/admin/videos/' + del, { method: 'DELETE' }); toast('Deleted'); loadVideos(); } catch (err) { toast(err.message); }
    }
  });

  function openVideoModal(id) {
    $('#video-modal-title').textContent = id ? 'Edit video' : 'New video';
    $('#video-modal').hidden = true;
    loadBloggersSelect().then(() => {
      if (id) {
        api('/videos/' + id).then((v) => {
          $('#video-id').value = v.id;
          $('#video-blogger-id').value = String(v.blogger_id);
          $('#video-title').value = v.title;
          $('#video-url').value = v.video_url;
          $('#video-cover').value = v.cover_url;
          $('#video-desc').value = v.description;
          $('#video-status').value = String(v.status);
        }).catch((e) => toast(e.message));
      } else {
        $('#video-form').reset(); $('#video-id').value = ''; $('#video-status').value = '1';
      }
      $('#video-modal').hidden = false;
    });
  }
  $('#video-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const id = $('#video-id').value;
    const body = {
      blogger_id: parseInt($('#video-blogger-id').value, 10),
      title: $('#video-title').value.trim(),
      video_url: $('#video-url').value.trim(),
      cover_url: $('#video-cover').value.trim(),
      description: $('#video-desc').value.trim(),
      status: parseInt($('#video-status').value, 10)
    };
    if (!body.blogger_id) { toast('Please select a creator'); return; }
    if (!body.title || !body.video_url) { toast('Title and video URL are required'); return; }
    try {
      if (id) await api('/admin/videos/' + id, { method: 'PUT', body: JSON.stringify(body) });
      else await api('/admin/videos', { method: 'POST', body: JSON.stringify(body) });
      toast('Saved'); $('#video-modal').hidden = true; loadVideos();
    } catch (err) { toast(err.message); }
  });

  // Cover upload
  $('#upload-cover').addEventListener('click', () => {
    const inp = document.createElement('input'); inp.type = 'file'; inp.accept = 'image/*';
    inp.onchange = async () => {
      const f = inp.files[0]; if (!f) return;
      const fd = new FormData(); fd.append('file', f);
      try {
        const d = await api('/admin/upload', { method: 'POST', body: fd, headers: {} });
        $('#video-cover').value = d.url; toast('Uploaded');
      } catch (e) { toast(e.message); }
    };
    inp.click();
  });

  // ---------- Comments ----------
  let cPage = 1;
  async function loadComments() {
    const bid = $('#comment-video').value;
    const q = `page=${cPage}&page_size=10${bid ? '&blogger_id=' + bid : ''}`;
    const data = await api('/admin/videos?' + q).catch(() => ({ items: [] }));
    const tb = $('#comment-table tbody'); tb.innerHTML = '';
    let count = 0;
    for (const v of (data.items || [])) {
      const cd = await api('/videos/' + v.id + '/comments?page=1&page_size=5').catch(() => ({ items: [] }));
      (cd.items || []).forEach((c) => {
        count++;
        const tr = document.createElement('tr');
        tr.innerHTML = `<td>${c.id}</td>
          <td style="max-width:160px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">${esc(v.title)}</td>
          <td>${esc(c.nickname)}</td>
          <td style="max-width:300px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">${esc(c.content)}</td>
          <td>${c.created_at || ''}</td>
          <td><button class="btn sm ghost link-danger" data-del="${c.id}">Delete</button></td>`;
        tb.appendChild(tr);
      });
    }
    if (count === 0) tb.innerHTML = '<tr><td colspan="6" style="text-align:center;color:var(--muted)">No comments</td></tr>';
  }
  $('#comment-video').addEventListener('change', () => { cPage = 1; loadComments(); });
  $('#comment-table').addEventListener('click', async (e) => {
    const del = e.target.dataset.del;
    if (del && confirm('Delete this comment?')) {
      try { await api('/admin/comments/' + del, { method: 'DELETE' }); toast('Deleted'); loadComments(); } catch (err) { toast(err.message); }
    }
  });

  // ---------- Common ----------
  function renderPager(sel, total, page, size, cb) {
    const el = $(sel); el.innerHTML = '';
    const pages = Math.max(1, Math.ceil((total || 0) / size));
    const prev = document.createElement('button'); prev.textContent = 'Prev'; prev.disabled = page <= 1;
    prev.onclick = () => cb(page - 1);
    const info = document.createElement('span'); info.textContent = `Page ${page} / ${pages} (${total} items)`;
    const next = document.createElement('button'); next.textContent = 'Next'; next.disabled = page >= pages;
    next.onclick = () => cb(page + 1);
    el.append(prev, info, next);
  }
  function debounce(fn, ms) { let t; return (...a) => { clearTimeout(t); t = setTimeout(() => fn(...a), ms); }; }

  // Modal close
  $$('[data-close]').forEach((b) => b.addEventListener('click', () => { $('#blogger-modal').hidden = true; $('#video-modal').hidden = true; }));
  $$('[data-act="new-blogger"]').forEach((b) => b.addEventListener('click', () => openBloggerModal(0)));
  $$('[data-act="new-video"]').forEach((b) => b.addEventListener('click', () => openVideoModal(0)));
  window.addEventListener('click', (e) => {
    if (e.target.classList.contains('modal')) { e.target.hidden = true; }
  });

  // ---------- Bootstrap ----------
  if (token) { api('/admin/profile').then(() => enterAdmin()).catch(() => { $('#login-page').hidden = false; }); }
  else { $('#login-page').hidden = false; }
})();
