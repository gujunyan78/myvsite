(function () {
  'use strict';

  const API = '/api';
  const $ = (s, r) => (r || document).querySelector(s);

  // Unique client id for like state
  let clientID = localStorage.getItem('vsite_cid');
  if (!clientID) {
    clientID = 'c_' + Math.random().toString(36).slice(2) + Date.now().toString(36);
    localStorage.setItem('vsite_cid', clientID);
  }

  function toast(msg) {
    const t = $('#toast');
    t.textContent = msg;
    t.hidden = false;
    clearTimeout(toast._t);
    toast._t = setTimeout(() => { t.hidden = true; }, 1800);
  }

  async function api(path, opts) {
    const res = await fetch(API + path, Object.assign({ headers: { 'Content-Type': 'application/json' } }, opts));
    const data = await res.json().catch(() => ({}));
    if (data.code && data.code !== 0) throw new Error(data.message || 'Request failed');
    return data.data;
  }

  const fmt = (n) => {
    n = Number(n) || 0;
    if (n >= 1000000000) return (n / 1000000000).toFixed(1) + 'B';
    if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M';
    if (n >= 1000) return (n / 1000).toFixed(1) + 'K';
    return String(n);
  };

  const icon = {
    heart: '<svg viewBox="0 0 24 24"><path d="M12 21s-7.5-4.6-10-9.3C.6 8.9 2 5.5 5.2 5.5c2 0 3.3 1.2 4 2.3.7-1.1 2-2.3 4-2.3 3.2 0 4.6 3.4 3.2 6.2C19.5 16.4 12 21 12 21z"/></svg>',
    comment: '<svg viewBox="0 0 24 24"><path d="M4 4h16a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H9l-5 4v-4H4a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2z"/></svg>',
    share: '<svg viewBox="0 0 24 24"><path d="M14 9V5l8 7-8 7v-4.1c-5 0-8.5 1.6-11 5.1 1-5 4-10 11-11z"/></svg>',
    back: '<svg viewBox="0 0 24 24"><path d="M15.5 4l-8 8 8 8 1.4-1.4L10.3 12l6.6-6.6z"/></svg>'
  };

  // Staggered cover ratios to form the masonry waterfall
  const ratios = ['3 / 4', '1 / 1', '4 / 5', '3 / 4'];
  function ratioFor(i) { return ratios[i % ratios.length]; }

  // ---------- Playback mode ----------
  // Mobile: the video in the current screen autoplays; scrolling it out pauses it.
  // Desktop: the list only shows cover images; clicking opens a modal player.
  const mobileMQ = window.matchMedia('(max-width: 820px)');
  const isMobile = () => mobileMQ.matches;

  // Only one player at a time
  function pauseOthers(cur) {
    document.querySelectorAll('video').forEach((v) => {
      if (v !== cur && !v.paused) {
        v._autoPaused = false;
        v.pause();
      }
    });
  }

  function stopTrackingVideos(root) {
    (root || document).querySelectorAll('video').forEach((v) => videoObserver.unobserve(v));
  }
  function stopTrackingCards(root) {
    (root || document).querySelectorAll('.card').forEach((c) => cardObserver.unobserve(c));
  }

  // Remove inline players and bring back the cover images (used when leaving mobile mode)
  function restoreCovers() {
    document.querySelectorAll('.card').forEach((card) => {
      const media = $('.media', card);
      const vid = $('video', media);
      if (!vid) return;
      vid.pause();
      videoObserver.unobserve(vid);
      vid.remove();
      media.classList.remove('playing');
      if (card._coverImg && !$('img', media)) {
        media.insertBefore(card._coverImg, media.firstChild);
      }
    });
  }

  // Mobile autoplay: card entering the screen plays, leaving pauses
  const cardObserver = new IntersectionObserver((entries) => {
    if (!isMobile()) return;
    entries.forEach((e) => {
      const card = e.target;
      const v = card._data;
      if (!v) return;
      if (e.intersectionRatio >= 0.6) {
        autoplayInline(card, v);
      } else if (e.intersectionRatio < 0.35) {
        const vid = $('video', card);
        if (vid && !vid.paused) {
          vid._autoPaused = true;
          vid.pause();
        }
      }
    });
  }, { threshold: [0, 0.35, 0.6] });

  // Legacy observer kept for any inline video element (resume/pause on scroll)
  const videoObserver = new IntersectionObserver((entries) => {
    entries.forEach((e) => {
      const vid = e.target;
      if (e.intersectionRatio >= 0.6) {
        if (vid._autoPaused) {
          vid._autoPaused = false;
          const p = vid.play();
          if (p && p.catch) p.catch(() => {});
        }
      } else if (e.intersectionRatio < 0.35 && !vid.paused) {
        vid._autoPaused = true;
        vid.pause();
      }
    });
  }, { threshold: [0, 0.35, 0.6] });

  // Switching device mode
  mobileMQ.addEventListener('change', (e) => {
    if (e.matches) {
      // switched to mobile: autoplay whatever is on screen now
      requestAnimationFrame(() => {
        document.querySelectorAll('.card').forEach((card) => {
          const r = card.getBoundingClientRect();
          const visible = Math.min(r.bottom, window.innerHeight) - Math.max(r.top, 0);
          if (r.height > 0 && visible / r.height >= 0.6 && card._data) {
            autoplayInline(card, card._data);
          }
        });
      });
    } else {
      // switched to PC: remove inline players, restore covers, close the modal
      closeModal();
      restoreCovers();
    }
  });

  const placeholder = 'data:image/svg+xml;utf8,' + encodeURIComponent(
    '<svg xmlns="http://www.w3.org/2000/svg" width="400" height="500"><rect width="400" height="500" fill="#23243a"/><text x="50%" y="50%" fill="#3a3c52" font-size="40" font-family="sans-serif" text-anchor="middle" dominant-baseline="middle">▶</text></svg>'
  );

  function el(html) {
    const d = document.createElement('div');
    d.innerHTML = html.trim();
    return d.firstElementChild;
  }

  // ---------- Card rendering ----------
  function renderCard(v, index) {
    const card = el(`
      <article class="card" data-id="${v.id}">
        <div class="media" style="aspect-ratio:${ratioFor(index)}">
          <img loading="lazy" src="${v.cover_url || placeholder}" alt="${escapeHtml(v.title)}" onerror="this.src='${placeholder}'">
          <div class="play-badge"></div>
        </div>
        <div class="card-body">
          <h3 class="card-title">${escapeHtml(v.title)}</h3>
          <div class="author-row">
            <img class="avatar" src="${v.blogger && v.blogger.avatar ? v.blogger.avatar : placeholder}" onerror="this.src='${placeholder}'">
            <span class="author-name" data-blogger="${v.blogger_id}">${escapeHtml(v.blogger ? v.blogger.account : 'Unknown creator')}</span>
          </div>
          <div class="actions">
            <button class="act act-comment" data-id="${v.id}">${icon.comment}<span>${fmt(v.comments)}</span></button>
            <button class="act act-like ${v.liked ? 'active' : ''}" data-id="${v.id}">${icon.heart}<span class="like-n">${fmt(v.likes)}</span></button>
            <button class="act act-spacer act-share" data-id="${v.id}">${icon.share}<span>Share</span></button>
          </div>
          <p class="meta-line">${fmt(v.views)} views</p>
        </div>
      </article>`);

    // Play: mobile autoplays via cardObserver, click toggles inline playback;
    // desktop only shows the cover here and opens the modal player on click.
    const media = $('.media', card);
    media.addEventListener('click', () => playVideo(card, v));
    card._data = v;
    cardObserver.observe(card);

    // Creator
    const author = $('.author-name', card);
    author.addEventListener('click', (e) => {
      e.stopPropagation();
      location.hash = '#/blogger/' + v.blogger_id;
    });

    // Comments / like / share
    $('.act-comment', card).addEventListener('click', (e) => { e.stopPropagation(); openComments(v.id); });
    $('.act-like', card).addEventListener('click', (e) => { e.stopPropagation(); toggleLike(v.id, card); });
    $('.act-share', card).addEventListener('click', (e) => { e.stopPropagation(); shareVideo(v); });

    return card;
  }

  function playVideo(card, v) {
    if (isMobile()) {
      // Mobile: toggle inline playback
      const media = $('.media', card);
      const vid = $('video', media);
      if (vid) {
        vid._autoPaused = false;
        vid.paused ? vid.play() : vid.pause();
      } else {
        mountInlineVideo(card, v);
      }
      return;
    }
    // Desktop: modal player, list stays image-only
    openModal(v);
  }

  function mountInlineVideo(card, v) {
    const media = $('.media', card);
    card._coverImg = $('img', media) || card._coverImg || null;
    media.innerHTML = '';
    const vid = document.createElement('video');
    vid.src = v.video_url;
    vid.controls = true;
    vid.playsInline = true;
    vid.preload = 'metadata';
    vid.setAttribute('playsinline', '');
    vid.addEventListener('play', () => pauseOthers(vid));
    videoObserver.observe(vid);
    media.appendChild(vid);
    media.classList.add('playing');
    tryPlay(vid);
    api(`/videos/${v.id}/view`, { method: 'POST' }).catch(() => {});
  }

  // Mobile autoplay entry point (called by cardObserver when a card enters the screen)
  function autoplayInline(card, v) {
    const media = $('.media', card);
    const vid = $('video', media);
    if (vid) {
      vid._autoPaused = false;
      if (vid.paused) tryPlay(vid);
    } else {
      mountInlineVideo(card, v);
    }
  }

  function tryPlay(vid) {
    const p = vid.play();
    if (p && p.catch) {
      p.catch(() => { vid.muted = true; vid.play(); });
    }
  }

  // ---------- Desktop modal player ----------
  let modalEl = null;

  function openModal(v) {
    closeModal();
    modalEl = el(`<div class="vmodal">
      <div class="vmodal-box">
        <button class="vmodal-close" aria-label="Close">×</button>
        <video src="${v.video_url}" controls playsinline preload="metadata"></video>
        <div class="vmodal-info">
          <h3>${escapeHtml(v.title)}</h3>
          <span class="vmodal-author">${escapeHtml(v.blogger ? v.blogger.account : '')}</span>
        </div>
      </div>
    </div>`);
    document.body.appendChild(modalEl);

    const vid = $('video', modalEl);
    vid.setAttribute('playsinline', '');
    vid.addEventListener('play', () => pauseOthers(vid));
    tryPlay(vid);
    api(`/videos/${v.id}/view`, { method: 'POST' }).catch(() => {});

    $('.vmodal-close', modalEl).addEventListener('click', closeModal);
    modalEl.addEventListener('click', (e) => { if (e.target === modalEl) closeModal(); });
    document.addEventListener('keydown', modalEscClose);
  }

  function modalEscClose(e) {
    if (e.key === 'Escape') closeModal();
  }

  function closeModal() {
    if (!modalEl) return;
    const vid = $('video', modalEl);
    if (vid) {
      vid.pause();
      vid.removeAttribute('src');
      try { vid.load(); } catch (err) { /* noop */ }
    }
    modalEl.remove();
    modalEl = null;
    document.removeEventListener('keydown', modalEscClose);
  }

  async function toggleLike(id, card) {
    try {
      const r = await api(`/videos/${id}/like`, {
        method: 'POST',
        body: JSON.stringify({ client_id: clientID })
      });
      const btn = $('.act-like', card);
      btn.classList.toggle('active', r.liked);
      $('.like-n', btn).textContent = fmt(r.likes);
      toast(r.liked ? 'Liked ❤' : 'Like removed');
    } catch (e) { toast(e.message); }
  }

  function shareVideo(v) {
    const url = location.origin + '/#' + v.id;
    const shareData = { title: v.title, text: v.title, url };
    if (navigator.share) {
      navigator.share(shareData).catch(() => {});
    } else if (navigator.clipboard) {
      navigator.clipboard.writeText(url).then(() => toast('Link copied')).catch(() => toast(url));
    } else {
      toast('Share link: ' + url);
    }
  }

  function escapeHtml(s) {
    return String(s || '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
  }

  // ---------- Infinite scroll feed ----------
  function createFeed(container, loaderEl, endEl, fetchFn) {
    const state = { loading: false, done: false, cursor: null, loaded: new Set(), total: 0 };
    const io = new IntersectionObserver((entries) => {
      if (entries[0].isIntersecting && !state.loading && !state.done) load();
    }, { rootMargin: '600px' });

    async function load() {
      if (state.loading || state.done) return;
      state.loading = true;
      loaderEl.hidden = false;
      try {
        const data = await fetchFn(state.cursor);
        const items = data.items || [];
        let added = 0;
        items.forEach((v, i) => {
          if (state.loaded.has(v.id)) return;
          state.loaded.add(v.id);
          container.appendChild(renderCard(v, container.children.length + i));
          added++;
        });
        state.total = data.total || 0;
        if (data.next_cursor && (added > 0 || items.length > 0)) {
          state.cursor = data.next_cursor;
        }
        if (!data.has_more || state.loaded.size >= state.total) {
          state.done = true;
        }
        if (state.done) {
          endEl.hidden = false;
          io.disconnect();
        }
      } catch (e) {
        toast('Failed to load: ' + e.message);
      } finally {
        state.loading = false;
        if (!state.done) loaderEl.hidden = false;
      }
    }

    load();
    io.observe(loaderEl);
    return {
      reset() { state.loading = false; state.done = false; state.cursor = null; state.loaded = new Set(); state.total = 0; endEl.hidden = true; stopTrackingVideos(container); stopTrackingCards(container); container.innerHTML = ''; io.observe(loaderEl); load(); },
      destroy() { io.disconnect(); }
    };
  }

  // ---------- View switching ----------
  const views = {
    home: $('#home-view'), bloggers: $('#bloggers-view'), blogger: $('#blogger-view')
  };
  function showView(name) {
    Object.keys(views).forEach((k) => { views[k].hidden = (k !== name); });
    $('.topnav a[data-nav]').forEach((a) => a.classList.toggle('active', a.dataset.nav === name || (name === 'blogger' && a.dataset.nav === 'home')));
    window.scrollTo(0, 0);
  }

  // Home feed
  let homeFeed = createFeed($('#feed'), $('#loader'),
    $('#feed-end'),
    (cursor) => api(`/videos?page_size=12${cursor ? '&cursor=' + encodeURIComponent(cursor) : ''}&client_id=${clientID}`));
  let searchFeed = null;
  $('#search-form').addEventListener('submit', (e) => {
    e.preventDefault();
    const kw = $('#search-input').value.trim();
    if (kw) { location.hash = '#/search/' + encodeURIComponent(kw); }
    else { location.hash = '#/'; }
  });

  // Creators directory
  async function renderBloggers() {
    try {
      const data = await api('/bloggers?page_size=60');
      const list = $('#blogger-list');
      list.innerHTML = '';
      (data.items || []).forEach((b) => {
        const card = el(`<div class="blogger-card" data-id="${b.id}">
          <img class="avatar" src="${b.avatar || placeholder}" onerror="this.src='${placeholder}'">
          <div><b>${escapeHtml(b.account)}</b><p>${escapeHtml(b.description || 'This creator has no bio yet')}</p></div>
        </div>`);
        card.addEventListener('click', () => { location.hash = '#/blogger/' + b.id; });
        list.appendChild(card);
      });
    } catch (e) { toast(e.message); }
  }

  // Creator page
  let bloggerFeed = null;
  let bloggerFeedID = null;
  async function renderBlogger(id) {
    try {
      const b = await api('/bloggers/' + id);
      showView('blogger');
      $('#blogger-head').innerHTML = `
        <img class="avatar" src="${b.avatar || placeholder}" onerror="this.src='${placeholder}'">
        <div class="blogger-info">
          <h2>${escapeHtml(b.account)}</h2>
          <p>${escapeHtml(b.description || 'This creator has no bio yet')}</p>
        </div>
        <div class="blogger-stat">
          <div><b>${fmt(b.video_count)}</b><span>videos</span></div>
        </div>`;
      // 更新 feed 归属的博主，避免切换到其他博主时仍请求旧 id
      bloggerFeedID = id;
      if (!bloggerFeed) {
        bloggerFeed = createFeed($('#blogger-feed'), $('#blogger-loader'), $('#blogger-end'),
          (cursor) => api(`/videos?blogger_id=${bloggerFeedID}&page_size=12${cursor ? '&cursor=' + encodeURIComponent(cursor) : ''}&client_id=${clientID}`));
      } else {
        bloggerFeed.reset();
      }
    } catch (e) { toast(e.message); showView('home'); }
  }

  function renderSearch(kw) {
    showView('home');
    $('#hero-title').textContent = 'Search: ' + decodeURIComponent(kw);
    $('#hero-sub').textContent = 'Related videos for you';
    $('#feed-end').hidden = true;
    stopTrackingVideos($('#feed'));
    stopTrackingCards($('#feed'));
    $('#feed').innerHTML = '';
    if (searchFeed) searchFeed.destroy();
    searchFeed = createFeed($('#feed'), $('#loader'), $('#feed-end'),
      (cursor) => api(`/videos?keyword=${kw}&page_size=12${cursor ? '&cursor=' + encodeURIComponent(cursor) : ''}&client_id=${clientID}`));
  }

  // ---------- Comment sheet ----------
  let currentVideo = null;
  function openComments(videoId) {
    currentVideo = videoId;
    $('#comment-mask').hidden = false;
    $('#comment-sheet').hidden = false;
    loadComments();
  }
  function closeComments() {
    $('#comment-mask').hidden = true;
    $('#comment-sheet').hidden = true;
    currentVideo = null;
  }
  $('#comment-close').addEventListener('click', closeComments);
  $('#comment-mask').addEventListener('click', closeComments);
  document.addEventListener('keydown', (e) => { if (e.key === 'Escape') closeComments(); });

  async function loadComments() {
    if (!currentVideo) return;
    try {
      const data = await api(`/videos/${currentVideo}/comments?page=1&page_size=30`);
      const list = $('#comment-list');
      const items = data.items || [];
      if (!items.length) { list.innerHTML = '<div class="comment-empty">No comments yet. Be the first!</div>'; return; }
      list.innerHTML = '';
      items.forEach((c) => {
        list.appendChild(el(`<div class="comment-item">
          <div class="c-head"><span class="c-name">${escapeHtml(c.nickname)}</span><span class="c-time">${fmtTime(c.created_at)}</span></div>
          <div class="c-content">${escapeHtml(c.content)}</div>
        </div>`));
      });
    } catch (e) { toast(e.message); }
  }
  $('#comment-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    if (!currentVideo) return;
    const content = $('#comment-content').value.trim();
    if (!content) { toast('Say something first~'); return; }
    const btn = e.target.querySelector('button');
    btn.disabled = true;
    try {
      await api(`/videos/${currentVideo}/comments`, {
        method: 'POST',
        body: JSON.stringify({ nickname: $('#comment-nick').value.trim(), content })
      });
      $('#comment-content').value = '';
      await loadComments();
      toast('Comment posted');
    } catch (err) { toast(err.message); }
    finally { btn.disabled = false; }
  });

  function fmtTime(s) {
    if (!s) return '';
    const d = new Date(s.replace(' ', 'T'));
    if (isNaN(d)) return s;
    const diff = (Date.now() - d.getTime()) / 1000;
    if (diff < 60) return 'just now';
    if (diff < 3600) return Math.floor(diff / 60) + ' min ago';
    if (diff < 86400) return Math.floor(diff / 3600) + ' h ago';
    if (diff < 2592000) return Math.floor(diff / 86400) + ' d ago';
    return s;
  }

  // ---------- Routing ----------
  function route() {
    const h = location.hash.replace(/^#/, '');
    if (h.startsWith('/blogger/')) {
      renderBlogger(h.split('/')[2]);
    } else if (h.startsWith('/search/')) {
      renderSearch(h.slice(8));
    } else if (h === '/bloggers') {
      showView('bloggers'); renderBloggers();
    } else {
      $('#hero-title').textContent = 'Discover great videos';
      $('#hero-sub').textContent = 'Scroll down for more — every screen is different';
      showView('home');
      // Returning from search results restores the random feed
      if (searchFeed) {
        searchFeed.destroy();
        searchFeed = null;
        stopTrackingVideos($('#feed'));
        stopTrackingCards($('#feed'));
        $('#feed').innerHTML = '';
        homeFeed.reset();
      }
    }
  }
  window.addEventListener('hashchange', route);
  route();
})();
