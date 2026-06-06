const defaults = {
        updatedAt: '',
        status: 'Kindle home • waiting for PC data',
        mail: { unread: 0, summary: 'No unread mail', items: [] },
        events: [],
        music: {
          badge: 'IDLE',
          summary: 'Waiting for PC player status',
          state: 'idle',
          track: '',
          artist: '',
          album: '',
          position: '',
          duration: '',
          source: 'PC',
          items: [
            { label: 'Bridge', value: 'update data.js from PC' },
            { label: 'Players', value: 'Spotify / MPD / playerctl' },
            { label: 'Updates', value: 'only on changes' }
          ]
        }
      };

      const APP_POLL_MS = 15000;
      const DATA_RELOAD_MS = 0;
      const HASS_RECONNECT_MS = 60000;
      const HASS_CALENDAR_RELOAD_MS = 900000;
      const LOCAL_API_BASE = 'http://127.0.0.1:8177';
      const SETTINGS_EDGE_PX = 36;
      let lastMinuteKey = '';
      let lastDateKey = '';
      let lastWeekKey = '';
      let lastDataLoad = 0;
      let lastHassConnectAttempt = 0;
      let lastLocalSettingsLoad = 0;
      let hassSocket = null;
      let hassAuthFailed = false;
      let hassMessageId = 1;
      let hassPendingRequests = {};
      let lastCalendarRefresh = 0;

      function normalizeOrientation(value) {
        const numeric = Number(value);
        if (!Number.isFinite(numeric)) return 270;
        const normalized = ((numeric % 360) + 360) % 360;
        return normalized === 0 || normalized === 90 || normalized === 180 || normalized === 270 ? normalized : 270;
      }

      function getQueryOrientation() {
        const params = new URLSearchParams(location.search || '');
        return normalizeOrientation(params.get('orientation'));
      }

      const localDeviceState = {
        brightness: 0,
        brightnessMax: 2399,
        batteryLevel: null,
        batteryStatus: '',
        orientation: getQueryOrientation(),
        apiReady: false,
        settingsOpen: false,
        settingsOffset: 0
      };

      const settingsGesture = {
        active: false,
        mode: '',
        startX: 0,
        startY: 0,
        startOffset: 0
      };

      function applyShellOrientation() {
        const shell = document.getElementById('appShell');
        if (!shell) return;
        const orientation = normalizeOrientation(localDeviceState.orientation);
        const viewportWidth = Math.max(window.innerWidth || 0, document.documentElement.clientWidth || 0);
        const viewportHeight = Math.max(window.innerHeight || 0, document.documentElement.clientHeight || 0);
        const rotated = orientation === 90 || orientation === 270;
        shell.style.width = (rotated ? viewportHeight : viewportWidth) + 'px';
        shell.style.height = (rotated ? viewportWidth : viewportHeight) + 'px';

        if (orientation === 0) shell.style.transform = 'none';
        else if (orientation === 90) shell.style.transform = 'translateX(' + viewportWidth + 'px) rotate(90deg)';
        else if (orientation === 180) shell.style.transform = 'translate(' + viewportWidth + 'px, ' + viewportHeight + 'px) rotate(180deg)';
        else shell.style.transform = 'translateY(' + viewportHeight + 'px) rotate(270deg)';

        applySettingsLayout();
        setSettingsDrawerOffset(localDeviceState.settingsOpen ? 0 : settingsPanelSize(), false);
      }

      function pad(n) { return String(n).padStart(2, '0'); }

      function setText(id, value) {
        const el = document.getElementById(id);
        const text = String(value);
        if (el && el.textContent !== text) el.textContent = text;
      }

      function clamp(value, min, max) {
        return Math.min(max, Math.max(min, value));
      }

      function brightnessToPercent(value, max) {
        if (!max) return 0;
        return clamp(Math.round((Number(value || 0) / Number(max || 1)) * 100), 0, 100);
      }

      function percentToBrightness(percent, max) {
        return clamp(Math.round((Number(percent || 0) / 100) * Number(max || 1)), 0, Number(max || 1));
      }

      function updateBrightnessUi(value, max) {
        const percent = brightnessToPercent(value, max);
        const slider = document.getElementById('settingsBrightnessSlider');
        if (slider) slider.value = percent;
        setText('brightnessQuickValue', percent + '%');
        setText('settingsBrightnessValue', percent + '%');
      }

      function updateBatteryUi(level, status) {
        const pct = level == null || Number.isNaN(Number(level)) ? '--' : (String(level) + '%');
        const suffix = status ? (' • ' + status) : '';
        setText('deviceBatteryValue', pct + suffix);
      }

      function updateOrientationUi(value) {
        const orientation = normalizeOrientation(value);
        setText('settingsOrientationValue', orientation + '°');
        [0, 90, 180, 270].forEach(candidate => {
          const el = document.getElementById('orientationBtn' + candidate);
          if (el) el.classList.toggle('active', candidate === orientation);
        });
      }

      function settingsSide() {
        return 'right';
      }

      function settingsPanelSize() {
        const panel = document.getElementById('settingsPanel');
        const side = settingsSide();
        if (!panel) return 360;
        return side === 'left' || side === 'right' ? panel.offsetWidth : panel.offsetHeight;
      }

      function applySettingsLayout() {
        const edge = document.getElementById('settingsEdgeZone');
        const panel = document.getElementById('settingsPanel');
        if (!edge || !panel) return;
        const side = settingsSide();

        edge.style.left = 'auto';
        edge.style.right = 'auto';
        edge.style.top = 'auto';
        edge.style.bottom = 'auto';
        edge.style.width = '';
        edge.style.height = '';

        panel.style.left = 'auto';
        panel.style.right = 'auto';
        panel.style.top = 'auto';
        panel.style.bottom = 'auto';
        panel.style.width = '';
        panel.style.height = '';
        panel.style.maxWidth = '';
        panel.style.maxHeight = '';
        panel.style.borderLeft = '2px solid var(--dark)';
        panel.style.borderRight = '2px solid var(--dark)';
        panel.style.borderTop = '2px solid var(--dark)';
        panel.style.borderBottom = '2px solid var(--dark)';

        if (side === 'right') {
          edge.style.top = '0'; edge.style.right = '0'; edge.style.width = '28px'; edge.style.height = '100%';
          panel.style.top = '0'; panel.style.right = '0'; panel.style.width = '360px'; panel.style.maxWidth = 'calc(100% - 28px)'; panel.style.height = '100%';
          panel.style.borderRight = '0';
          panel.style.borderRadius = '22px 0 0 22px';
          panel.style.boxShadow = '-5px 5px 0 var(--dark)';
        } else if (side === 'left') {
          edge.style.top = '0'; edge.style.left = '0'; edge.style.width = '28px'; edge.style.height = '100%';
          panel.style.top = '0'; panel.style.left = '0'; panel.style.width = '360px'; panel.style.maxWidth = 'calc(100% - 28px)'; panel.style.height = '100%';
          panel.style.borderLeft = '0';
          panel.style.borderRadius = '0 22px 22px 0';
          panel.style.boxShadow = '5px 5px 0 var(--dark)';
        } else if (side === 'top') {
          edge.style.top = '0'; edge.style.left = '0'; edge.style.width = '100%'; edge.style.height = '28px';
          panel.style.top = '0'; panel.style.left = '0'; panel.style.width = '100%'; panel.style.height = '360px'; panel.style.maxHeight = 'calc(100% - 28px)';
          panel.style.borderTop = '0';
          panel.style.borderRadius = '0 0 22px 22px';
          panel.style.boxShadow = '5px 5px 0 var(--dark)';
        } else {
          edge.style.bottom = '0'; edge.style.left = '0'; edge.style.width = '100%'; edge.style.height = '28px';
          panel.style.bottom = '0'; panel.style.left = '0'; panel.style.width = '100%'; panel.style.height = '360px'; panel.style.maxHeight = 'calc(100% - 28px)';
          panel.style.borderBottom = '0';
          panel.style.borderRadius = '22px 22px 0 0';
          panel.style.boxShadow = '5px -5px 0 var(--dark)';
        }
      }

      function setSettingsDrawerOffset(offset, animated) {
        const backdrop = document.getElementById('settingsBackdrop');
        const panel = document.getElementById('settingsPanel');
        if (!backdrop || !panel) return;
        const side = settingsSide();
        const size = settingsPanelSize();
        const clamped = clamp(Number(offset || 0), 0, size);
        localDeviceState.settingsOffset = clamped;
        panel.style.transition = animated ? 'transform 180ms ease' : 'none';
        if (side === 'right') panel.style.transform = 'translateX(' + clamped + 'px)';
        else if (side === 'left') panel.style.transform = 'translateX(' + (-clamped) + 'px)';
        else if (side === 'top') panel.style.transform = 'translateY(' + (-clamped) + 'px)';
        else panel.style.transform = 'translateY(' + clamped + 'px)';
        const openness = 1 - (clamped / Math.max(size, 1));
        backdrop.classList.toggle('open', openness > 0.001);
        backdrop.style.background = 'rgba(23, 23, 23, ' + (0.28 * openness) + ')';
      }

      function syncSettingsDrawer(open, animated) {
        localDeviceState.settingsOpen = !!open;
        setSettingsDrawerOffset(open ? 0 : settingsPanelSize(), animated !== false);
      }

      function isInteractiveSettingsTarget(target) {
        return !!(target && target.closest && target.closest('button,input,select,textarea,label,a'));
      }

      function beginSettingsGesture(mode, touch) {
        settingsGesture.active = true;
        settingsGesture.mode = mode;
        settingsGesture.startX = touch.clientX;
        settingsGesture.startY = touch.clientY;
        settingsGesture.startOffset = localDeviceState.settingsOffset;
      }

      function drawerDragDelta(dx, dy) {
        if (localDeviceState.orientation === 90) return dy;
        if (localDeviceState.orientation === 180) return -dx;
        if (localDeviceState.orientation === 270) return -dy;
        return dx;
      }

      function drawerPrimaryDistance(dx, dy) {
        if (localDeviceState.orientation === 90 || localDeviceState.orientation === 270) return Math.abs(dy);
        return Math.abs(dx);
      }

      function drawerSecondaryDistance(dx, dy) {
        if (localDeviceState.orientation === 90 || localDeviceState.orientation === 270) return Math.abs(dx);
        return Math.abs(dy);
      }

      function handleSettingsTouchStart(event) {
        const panel = document.getElementById('settingsPanel');
        const edge = document.getElementById('settingsEdgeZone');
        if (!panel || !edge || !event.touches || !event.touches.length) return;
        const touch = event.touches[0];
        const panelRect = panel.getBoundingClientRect();
        const edgeRect = edge.getBoundingClientRect();
        const inEdge = touch.clientX >= edgeRect.left && touch.clientX <= edgeRect.right && touch.clientY >= edgeRect.top && touch.clientY <= edgeRect.bottom;

        if (!localDeviceState.settingsOpen) {
          if (!inEdge) return;
          syncSettingsDrawer(false, false);
          beginSettingsGesture('open', touch);
          return;
        }

        if (isInteractiveSettingsTarget(event.target)) return;
        if (touch.clientX >= panelRect.left && touch.clientX <= panelRect.right && touch.clientY >= panelRect.top && touch.clientY <= panelRect.bottom) beginSettingsGesture('close', touch);
      }

      function handleSettingsTouchMove(event) {
        if (!settingsGesture.active || !event.touches || !event.touches.length) return;
        const touch = event.touches[0];
        const dx = touch.clientX - settingsGesture.startX;
        const dy = touch.clientY - settingsGesture.startY;
        if (drawerSecondaryDistance(dx, dy) > drawerPrimaryDistance(dx, dy) + 8) return;
        const delta = drawerDragDelta(dx, dy);
        const size = settingsPanelSize();
        let offset = settingsGesture.startOffset;
        if (settingsGesture.mode === 'open') offset = clamp(size + delta, 0, size);
        if (settingsGesture.mode === 'close') offset = clamp(delta, 0, size);
        setSettingsDrawerOffset(offset, false);
        event.preventDefault();
      }

      function finishSettingsGesture() {
        if (!settingsGesture.active) return;
        const size = settingsPanelSize();
        const shouldOpen = localDeviceState.settingsOffset < size * 0.45;
        syncSettingsDrawer(shouldOpen, true);
        settingsGesture.active = false;
        settingsGesture.mode = '';
      }

      async function fetchLocalText(path, options) {
        const response = await fetch(LOCAL_API_BASE + path, options || {});
        if (!response.ok) throw new Error('HTTP ' + response.status);
        return (await response.text()).trim();
      }

      async function loadLocalSettings(force) {
        const now = Date.now();
        if (!force && now - lastLocalSettingsLoad < 60000) return;
        lastLocalSettingsLoad = now;
        try {
          const [brightnessText, maxText, orientationText, batteryLevelText, batteryStatusText] = await Promise.all([
            fetchLocalText('/brightness'),
            fetchLocalText('/brightness-max'),
            fetchLocalText('/orientation'),
            fetchLocalText('/battery-level'),
            fetchLocalText('/battery-status')
          ]);
          localDeviceState.brightness = Math.max(0, parseInt(brightnessText, 10) || 0);
          localDeviceState.brightnessMax = Math.max(1, parseInt(maxText, 10) || 2399);
          localDeviceState.batteryLevel = Math.max(0, parseInt(batteryLevelText, 10) || 0);
          localDeviceState.batteryStatus = String(batteryStatusText || '');
          localDeviceState.orientation = normalizeOrientation(parseInt(orientationText, 10));
          localDeviceState.apiReady = true;
          updateBrightnessUi(localDeviceState.brightness, localDeviceState.brightnessMax);
          updateBatteryUi(localDeviceState.batteryLevel, localDeviceState.batteryStatus);
          updateOrientationUi(localDeviceState.orientation);
          applyShellOrientation();
          setText('settingsStatus', 'On-device settings ready');
        } catch (_) {
          localDeviceState.apiReady = false;
          updateOrientationUi(localDeviceState.orientation);
          updateBatteryUi(null, '');
          setText('brightnessQuickValue', '--');
          setText('settingsStatus', 'Local device API unavailable');
        }
      }

      function previewBrightness() {
        const slider = document.getElementById('settingsBrightnessSlider');
        const percent = clamp(Number(slider && slider.value || 0), 0, 100);
        setText('settingsBrightnessValue', percent + '%');
      }

      async function saveBrightness() {
        const slider = document.getElementById('settingsBrightnessSlider');
        const button = document.getElementById('saveBrightnessBtn');
        const percent = clamp(Number(slider && slider.value || 0), 0, 100);
        const raw = percentToBrightness(percent, localDeviceState.brightnessMax);
        setText('settingsStatus', 'Applying brightness…');
        if (button) button.disabled = true;
        try {
          const brightnessText = await fetchLocalText('/brightness?value=' + encodeURIComponent(raw), { method: 'POST' });
          localDeviceState.brightness = Math.max(0, parseInt(brightnessText, 10) || raw);
          localDeviceState.apiReady = true;
          updateBrightnessUi(localDeviceState.brightness, localDeviceState.brightnessMax);
          setText('settingsStatus', 'Brightness saved on device');
        } catch (_) {
          setText('settingsStatus', 'Failed to update brightness');
        }
        if (button) button.disabled = false;
      }

      async function saveOrientation(value) {
        const orientation = normalizeOrientation(value);
        setText('settingsStatus', 'Applying rotation…');
        try {
          const orientationText = await fetchLocalText('/orientation?value=' + encodeURIComponent(orientation), { method: 'POST' });
          localDeviceState.orientation = normalizeOrientation(parseInt(orientationText, 10));
          updateOrientationUi(localDeviceState.orientation);
          applyShellOrientation();
          setText('settingsStatus', 'Rotation saved on device');
        } catch (_) {
          setText('settingsStatus', 'Failed to update rotation');
        }
      }

      function setTheme(theme) {
        const isDark = theme === 'dark';
        document.body.classList.toggle('dark-theme', isDark);
        setText('settingsThemeValue', isDark ? 'Dark' : 'Light');
        const btnLight = document.getElementById('themeBtnLight');
        const btnDark = document.getElementById('themeBtnDark');
        if (btnLight) btnLight.classList.toggle('active', !isDark);
        if (btnDark) btnDark.classList.toggle('active', isDark);
        try {
          localStorage.setItem('theme', theme);
        } catch (_) {}
      }

      function openSettings() {
        syncSettingsDrawer(true, true);
        loadLocalSettings(true);
      }

      function closeSettings(event) {
        if (event && event.target && event.target.id !== 'settingsBackdrop') return;
        syncSettingsDrawer(false, true);
      }

      function tick() {
        const now = new Date();
        const h = now.getHours();
        const minuteKey = `${h}:${now.getMinutes()}`;
        if (minuteKey !== lastMinuteKey) {
          lastMinuteKey = minuteKey;
          const greeting = h < 12 ? 'Good morning' : h < 18 ? 'Good afternoon' : 'Good evening';
          setText('greeting', greeting);
          setText('hour', pad(h));
          setText('minute', pad(now.getMinutes()));
        }

        const dateKey = now.toDateString();
        if (dateKey !== lastDateKey) {
          lastDateKey = dateKey;
          setText('dateLine', now.toLocaleDateString(undefined, { weekday: 'long', month: 'long', day: 'numeric' }));
          setText('monthName', now.toLocaleDateString(undefined, { month: 'long' }));
          setText('yearName', String(now.getFullYear()));
          renderWeek(now);
        }
      }

      function renderWeek(now) {
        const target = document.getElementById('days');
        if (!target) return;
        const weekKey = `${now.getFullYear()}-${now.getMonth()}-${now.getDate()}`;
        if (weekKey === lastWeekKey) return;
        lastWeekKey = weekKey;
        const start = new Date(now);
        const mondayOffset = (start.getDay() + 6) % 7;
        start.setDate(start.getDate() - mondayOffset);
        target.innerHTML = '';
        for (let i = 0; i < 7; i++) {
          const d = new Date(start);
          d.setDate(start.getDate() + i);
          const el = document.createElement('div');
          el.className = 'day' + (d.toDateString() === now.toDateString() ? ' today' : '');
          el.innerHTML = `<b>${d.toLocaleDateString(undefined, { weekday: 'short' })}</b><span>${d.getDate()}</span>`;
          target.appendChild(el);
        }
      }

      function item(title, detail, meta, extraClass = '') {
        return `<div class="item ${extraClass}"><div class="item-main"><div class="item-title">${escapeHtml(title)}</div><div class="item-detail">${escapeHtml(detail || '')}</div></div><div class="item-meta">${escapeHtml(meta || '')}</div></div>`;
      }

      function empty(text) {
        return `<div class="empty">${escapeHtml(text)}</div>`;
      }

      function escapeHtml(value) {
        return String(value == null ? '' : value).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
      }

      function formatMediaTime(value) {
        if (value == null || value === '' || value === 'unknown' || value === 'unavailable') return '';
        const seconds = Math.max(0, Math.floor(Number(value) || 0));
        return `${Math.floor(seconds / 60)}:${pad(seconds % 60)}`;
      }

      function hassBadge(state) {
        state = String(state || '').toLowerCase();
        if (state === 'playing') return 'PLAYING';
        if (state === 'paused') return 'PAUSED';
        if (state === 'off' || state === 'standby') return 'OFF';
        return 'IDLE';
      }

      function hassMusicToDashboardData(state) {
        const attrs = state.attributes || {};
        const playerState = state.state || 'unknown';
        const device = attrs.friendly_name || state.entity_id || 'Home Assistant player';
        const source = attrs.app_name || attrs.source || 'Home Assistant';
        return {
          status: `Home Assistant • ${device}`,
          music: {
            entityId: state.entity_id,
            badge: hassBadge(playerState),
            summary: [device, source, playerState].filter(Boolean).join(' • '),
            state: playerState,
            track: attrs.media_title || device,
            artist: attrs.media_artist || attrs.media_album_artist || '',
            album: attrs.media_album_name || '',
            position: formatMediaTime(attrs.media_position),
            duration: formatMediaTime(attrs.media_duration),
            source,
            items: [
              { label: 'Device', value: device },
              { label: 'Player', value: source },
              { label: 'State', value: playerState },
              { label: 'Updated', value: String(state.last_updated || state.last_changed || '').replace('T', ' ').replace('+00:00', 'Z').slice(0, 19) }
            ]
          }
        };
      }

      function asArray(value) {
        if (!value) return [];
        return Array.isArray(value) ? value : [value];
      }

      function hassMailToDashboardData(state) {
        const attrs = state.attributes || {};
        const unread = Number(state.state) || Number(attrs.unread) || Number(attrs.unseen) || 0;
        const rawMessages = asArray(attrs.messages || attrs.message || attrs.emails || attrs.email || attrs.subjects || attrs.subject);
        const items = rawMessages.slice(0, 4).map((m, index) => {
          if (typeof m === 'string') return { from: 'Mail', subject: m, when: '' };
          return {
            from: m.from || m.sender || m.name || m.mailbox || 'Mail',
            subject: m.subject || m.title || m.summary || m.body || `Message ${index + 1}`,
            when: m.date || m.when || m.received || ''
          };
        });

        if (!items.length && unread > 0) {
          const cfg = window.HASS_CONFIG || {};
          const mailLabel = cfg.mailLabel || attrs.friendly_name || 'Mail';
          items.push({ from: mailLabel, subject: `${unread} unread message${unread === 1 ? '' : 's'}`, when: '' });
        }

        return {
          mail: {
            unread,
            summary: unread ? `${unread} unread from ` + ((window.HASS_CONFIG && window.HASS_CONFIG.mailLabel) || 'Mail') : 'Inbox quiet',
            items
          }
        };
      }

      function calendarEntities() {
        const cfg = window.HASS_CONFIG || {};
        const raw = cfg.calendarEntities || cfg.calendarEntity || ['calendar.it', 'calendar.calendario'];
        const parts = [];
        asArray(raw).forEach(v => String(v).split(',').forEach(part => parts.push(part)));
        return parts.map(v => v.trim()).filter(Boolean);
      }

      function formatEventTime(event) {
        const start = event.start || {};
        const raw = start.dateTime || start.date || event.start_time || event.start;
        if (!raw) return '';
        if (start.date || /^\d{4}-\d{2}-\d{2}$/.test(String(raw))) return 'all day';
        const date = new Date(raw);
        if (Number.isNaN(date.getTime())) return '';
        return `${pad(date.getHours())}:${pad(date.getMinutes())}`;
      }

      function hassCalendarToDashboardData(result) {
        const response = (result && result.response) || result || {};
        const events = [];
        Object.keys(response).forEach(entity => {
          const bucket = response[entity] || {};
          asArray(bucket.events).forEach(event => {
            events.push({
              time: formatEventTime(event),
              title: event.summary || event.title || 'Calendar event',
              detail: event.location || event.description || entity
            });
          });
        });
        events.sort((a, b) => String(a.time).localeCompare(String(b.time)));
        return {
          agendaSummary: events.length ? 'Home Assistant calendar' : 'No upcoming calendar events',
          events: events.slice(0, 4)
        };
      }

      function mergeHassData(partial) {
        window.KINDLE_HASS_DATA = window.KINDLE_HASS_DATA || {};
        if (partial.status) window.KINDLE_HASS_DATA.status = partial.status;
        if (partial.music) window.KINDLE_HASS_DATA.music = Object.assign({}, window.KINDLE_HASS_DATA.music || {}, partial.music);
        if (partial.mail) window.KINDLE_HASS_DATA.mail = Object.assign({}, window.KINDLE_HASS_DATA.mail || {}, partial.mail);
        if (partial.agendaSummary) window.KINDLE_HASS_DATA.agendaSummary = partial.agendaSummary;
        if (partial.events) window.KINDLE_HASS_DATA.events = partial.events;
        renderData();
      }

      let lastDataSignature = null;

      function toggleEntity(entityId, element) {
        if (element) {
          element.classList.add('active-tap');
          setTimeout(() => { element.classList.remove('active-tap'); }, 500);
        }

        if (!hassSocket || hassSocket.readyState !== WebSocket.OPEN) {
          console.warn("HA Socket not connected. Cannot toggle", entityId);
          return;
        }

        const domain = entityId.split('.')[0];
        
        hassSocket.send(JSON.stringify({
          id: hassMessageId++,
          type: 'call_service',
          domain: domain,
          service: 'toggle',
          target: { entity_id: entityId }
        }));
      }

      function renderData() {
        const sourceData = Object.assign({}, window.KINDLE_DASHBOARD_DATA || {});
        if (window.KINDLE_HASS_DATA) {
          sourceData.status = window.KINDLE_HASS_DATA.status || sourceData.status;
          sourceData.music = Object.assign({}, sourceData.music || {}, window.KINDLE_HASS_DATA.music || {});
          sourceData.mail = Object.assign({}, sourceData.mail || {}, window.KINDLE_HASS_DATA.mail || {});
          sourceData.agendaSummary = window.KINDLE_HASS_DATA.agendaSummary || sourceData.agendaSummary;
          sourceData.events = window.KINDLE_HASS_DATA.events || sourceData.events;
        }

        const dataSignature = JSON.stringify({ base: window.KINDLE_DASHBOARD_DATA || {}, hass: window.KINDLE_HASS_DATA || {} });
        if (dataSignature === lastDataSignature) return;
        lastDataSignature = dataSignature;

        const data = Object.assign({}, defaults, sourceData);
        const mail = Object.assign({}, defaults.mail, data.mail || {});
        const music = Object.assign({}, defaults.music, data.music || {});
        const events = Array.isArray(data.events) ? data.events : [];

        setText('statusLine', data.status || defaults.status);

        setText('mailCount', mail.unread || (mail.items ? mail.items.length : 0) || 0);
        setText('mailSubhead', mail.summary || defaults.mail.summary);
        document.getElementById('mailItems').innerHTML = (mail.items || []).slice(0, 2).map(m =>
          item(m.from || 'Mail', m.subject || m.detail || '', m.when || '')
        ).join('') || empty('Inbox quiet');

        setText('eventCount', events.length);
        document.getElementById('eventItems').innerHTML = events.slice(0, 2).map(e =>
          item(e.title || 'Event', e.detail || e.where || '', e.time || '')
        ).join('') || empty('No events today');

        const playing = String(music.state || '').toLowerCase() === 'playing';
        const musicBadgeEl = document.getElementById('musicBadge');
        if (musicBadgeEl) {
          musicBadgeEl.textContent = music.badge || (playing ? 'PLAYING' : 'IDLE');
          musicBadgeEl.onclick = () => toggleEntity(music.entityId, musicBadgeEl);
        }
        setText('musicSubhead', music.source || 'Music');

        const musicRows = [];
        if (music.track || music.artist) {
          const detail = [music.artist, music.album].filter(Boolean).join(' • ');
          const meta = [music.position, music.duration].filter(Boolean).join(' / ');
          musicRows.push(item(music.track || 'Unknown track', detail, meta, 'pc-line'));
        }
        document.getElementById('musicItems').innerHTML = musicRows.join('') || empty('Nothing playing');
      }

      function loadData() {
        const old = document.getElementById('data-js');
        if (old) old.remove();
        const script = document.createElement('script');
        script.id = 'data-js';
        script.src = 'data.js?ts=' + Date.now();
        script.onload = renderData;
        script.onerror = renderData;
        document.body.appendChild(script);
      }

      function hassWsUrl(rawUrl) {
        const url = String(rawUrl || '').replace(/\/$/, '');
        if (!url) return '';
        if (url.indexOf('https://') === 0) return 'wss://' + url.slice('https://'.length) + '/api/websocket';
        if (url.indexOf('http://') === 0) return 'ws://' + url.slice('http://'.length) + '/api/websocket';
        return url.replace(/^\/+/, '').replace(/^/, 'wss://') + '/api/websocket';
      }

      function handleHassState(state) {
        const cfg = window.HASS_CONFIG || {};
        const musicEntity = cfg.entity || cfg.musicEntity || 'media_player.googlehome1844';
        const mailEntity = cfg.mailEntity || 'sensor.imap_me_messages';
        const calendars = calendarEntities();
        if (!state || !state.entity_id) return;
        if (state.entity_id === musicEntity) mergeHassData(hassMusicToDashboardData(state));
        if (state.entity_id === mailEntity) mergeHassData(hassMailToDashboardData(state));
        if (calendars.indexOf(state.entity_id) !== -1) requestHassCalendarEvents(true);
      }

      function requestHassCalendarEvents(force) {
        if (!hassSocket || hassSocket.readyState !== WebSocket.OPEN) return;
        const nowMs = Date.now();
        if (!force && nowMs - lastCalendarRefresh < HASS_CALENDAR_RELOAD_MS) return;
        lastCalendarRefresh = nowMs;

        const entities = calendarEntities();
        if (!entities.length) return;
        const start = new Date();
        start.setHours(0, 0, 0, 0);
        const end = new Date(start);
        end.setDate(end.getDate() + 7);

        const id = hassMessageId++;
        hassPendingRequests[id] = 'calendar_events';
        hassSocket.send(JSON.stringify({
          id,
          type: 'call_service',
          domain: 'calendar',
          service: 'get_events',
          target: { entity_id: entities },
          service_data: {
            start_date_time: start.toISOString(),
            end_date_time: end.toISOString()
          },
          return_response: true
        }));
      }

      function checkApiKeyInfo() {
        const cfg = window.HASS_CONFIG || {};
        const token = cfg.token || cfg.HASS_TOKEN || '';
        if (!token) {
          setText('connApiInfo', 'API Key: Missing');
        } else if (token.indexOf('YOUR_') === 0 || token === 'placeholder') {
          setText('connApiInfo', 'API Key: Placeholder');
        } else {
          setText('connApiInfo', 'API Key: Present');
        }
      }

      function manualReconnectHass() {
        hassAuthFailed = false;
        if (hassSocket) {
          try { hassSocket.close(); } catch (_) {}
          hassSocket = null;
        }
        setText('connStatus', 'Connecting…');
        lastHassConnectAttempt = Date.now();
        connectHassWs();
      }

      function connectHassWs() {
        if (hassAuthFailed) {
          setText('connStatus', 'Auth Failed');
          return;
        }
        const cfg = window.HASS_CONFIG || {};
        const token = cfg.token || cfg.HASS_TOKEN || '';
        const wsUrl = hassWsUrl(cfg.url || cfg.HASS_URL || '');
        if (!wsUrl || !token || token.indexOf('YOUR_') === 0 || token === 'placeholder' || wsUrl.indexOf('your-') !== -1) {
          setText('connStatus', 'Config Missing');
          return;
        }
        if (hassSocket && (hassSocket.readyState === WebSocket.OPEN || hassSocket.readyState === WebSocket.CONNECTING)) return;

        try {
          setText('connStatus', 'Connecting…');
          hassSocket = new WebSocket(wsUrl);
        } catch (error) {
          setText('connStatus', 'Error');
          setText('musicSubhead', 'HA WebSocket unavailable; using local data.js');
          return;
        }

        hassSocket.onmessage = event => {
          let msg;
          try { msg = JSON.parse(event.data); } catch (_) { return; }

          if (msg.type === 'auth_required') {
            setText('connStatus', 'Authenticating…');
            hassSocket.send(JSON.stringify({ type: 'auth', access_token: token }));
            return;
          }

          if (msg.type === 'auth_ok') {
            setText('connStatus', 'Connected');
            setText('musicSubhead', 'Home Assistant connected');
            hassSocket.send(JSON.stringify({ id: hassMessageId++, type: 'get_states' }));
            hassSocket.send(JSON.stringify({ id: hassMessageId++, type: 'subscribe_events', event_type: 'state_changed' }));
            requestHassCalendarEvents(true);
            return;
          }

          if (msg.type === 'auth_invalid') {
            hassAuthFailed = true;
            setText('connStatus', 'Auth Failed');
            setText('musicSubhead', 'HA auth failed permanently');
            if (hassSocket) {
              try { hassSocket.close(); } catch (_) {}
            }
            return;
          }

          if (msg.type === 'result' && hassPendingRequests[msg.id] === 'calendar_events') {
            delete hassPendingRequests[msg.id];
            if (msg.success !== false) mergeHassData(hassCalendarToDashboardData(msg.result));
            return;
          }

          if (msg.type === 'result' && Array.isArray(msg.result)) {
            msg.result.forEach(handleHassState);
            return;
          }

          if (msg.type === 'event' && msg.event && msg.event.event_type === 'state_changed') {
            handleHassState(msg.event.data && msg.event.data.new_state);
          }
        };

        hassSocket.onclose = () => {
          hassSocket = null;
          if (hassAuthFailed) {
            setText('connStatus', 'Auth Failed');
          } else {
            setText('connStatus', 'Disconnected');
            setText('musicSubhead', 'HA disconnected; will reconnect');
          }
        };

        hassSocket.onerror = () => {
          setText('connStatus', 'Error');
          setText('musicSubhead', 'HA WebSocket error; using local data.js');
        };
      }

      function heartbeat() {
        tick();
        const now = Date.now();
        if (DATA_RELOAD_MS > 0 && now - lastDataLoad >= DATA_RELOAD_MS) {
          lastDataLoad = now;
          loadData();
        }
        if (now - lastHassConnectAttempt >= HASS_RECONNECT_MS) {
          lastHassConnectAttempt = now;
          connectHassWs();
        }
        loadLocalSettings(false);
        requestHassCalendarEvents(false);
      }

      // Initialize theme
      let savedTheme = 'light';
      try {
        savedTheme = localStorage.getItem('theme') || 'light';
      } catch (_) {}
      setTheme(savedTheme);

      checkApiKeyInfo();

      applyShellOrientation();
      syncSettingsDrawer(false, false);
      window.addEventListener('resize', () => {
        applyShellOrientation();
        syncSettingsDrawer(localDeviceState.settingsOpen, false);
      });
      document.getElementById('appShell').addEventListener('touchstart', handleSettingsTouchStart, { passive: true });
      document.getElementById('appShell').addEventListener('touchmove', handleSettingsTouchMove, { passive: false });
      document.getElementById('appShell').addEventListener('touchend', finishSettingsGesture, { passive: true });
      document.getElementById('appShell').addEventListener('touchcancel', finishSettingsGesture, { passive: true });
      updateOrientationUi(localDeviceState.orientation);
      renderData();
      loadData();
      loadLocalSettings(true);
      lastDataLoad = 0;
      lastHassConnectAttempt = 0;
      heartbeat();
      setInterval(heartbeat, APP_POLL_MS);
