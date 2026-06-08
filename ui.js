(function(window) {
  window.uiModule = {
    views: ['view-calendar', 'view-dashboard', 'view-launcher'],
    settingsGesture: {
      active: false,
      mode: '', // 'open', 'close', or 'viewSwipe'
      startX: 0,
      startY: 0,
      startOffset: 0
    },

    applyShellOrientation: function() {
      var shell = document.getElementById('appShell');
      if (!shell) return;
      var orientation = window.localDeviceState.orientation;
      var viewportWidth = Math.max(window.innerWidth || 0, document.documentElement.clientWidth || 0);
      var viewportHeight = Math.max(window.innerHeight || 0, document.documentElement.clientHeight || 0);
      var rotated = orientation === 90 || orientation === 270;
      shell.style.width = (rotated ? viewportHeight : viewportWidth) + 'px';
      shell.style.height = (rotated ? viewportWidth : viewportHeight) + 'px';

      if (orientation === 0) shell.style.transform = 'none';
      else if (orientation === 90) shell.style.transform = 'translateX(' + viewportWidth + 'px) rotate(90deg)';
      else if (orientation === 180) shell.style.transform = 'translate(' + viewportWidth + 'px, ' + viewportHeight + 'px) rotate(180deg)';
      else shell.style.transform = 'translateY(' + viewportHeight + 'px) rotate(270deg)';

      this.applySettingsLayout();
      this.setSettingsDrawerOffset(window.localDeviceState.settingsOpen ? 0 : this.settingsPanelSize(), false);
    },

    brightnessToPercent: function(value, max) {
      if (!max) return 0;
      return window.utils.clamp(Math.round((Number(value || 0) / Number(max || 1)) * 100), 0, 100);
    },

    percentToBrightness: function(percent, max) {
      return window.utils.clamp(Math.round((Number(percent || 0) / 100) * Number(max || 1)), 0, Number(max || 1));
    },

    updateBrightnessUi: function(value, max) {
      var percent = this.brightnessToPercent(value, max);
      var slider = document.getElementById('settingsBrightnessSlider');
      if (slider) slider.value = percent;
      window.utils.setText('brightnessQuickValue', percent + '%');
      window.utils.setText('settingsBrightnessValue', percent + '%');
    },

    updateBatteryUi: function(level, status) {
      var pct = level == null || Number.isNaN(Number(level)) ? '--' : (String(level) + '%');
      var suffix = status ? (' • ' + status) : '';
      window.utils.setText('deviceBatteryValue', pct + suffix);
    },

    updateOrientationUi: function(value) {
      var orientation = window.utils.normalizeOrientation(value);
      window.utils.setText('settingsOrientationValue', orientation + '°');
      [0, 90, 180, 270].forEach(function(candidate) {
        var el = document.getElementById('orientationBtn' + candidate);
        if (el) el.classList.toggle('active', candidate === orientation);
      });
    },

    settingsSide: function() { return 'right'; },

    settingsPanelSize: function() {
      var panel = document.getElementById('settingsPanel');
      if (!panel) return 360;
      var side = this.settingsSide();
      return side === 'left' || side === 'right' ? panel.offsetWidth : panel.offsetHeight;
    },

    applySettingsLayout: function() {
      var edge = document.getElementById('settingsEdgeZone');
      var panel = document.getElementById('settingsPanel');
      if (!edge || !panel) return;
      var side = this.settingsSide();

      edge.style.left = 'auto'; edge.style.right = 'auto'; edge.style.top = 'auto'; edge.style.bottom = 'auto';
      edge.style.width = ''; edge.style.height = '';

      panel.style.left = 'auto'; panel.style.right = 'auto'; panel.style.top = 'auto'; panel.style.bottom = 'auto';
      panel.style.width = ''; panel.style.height = ''; panel.style.maxWidth = ''; panel.style.maxHeight = '';
      panel.style.borderLeft = '2px solid var(--dark)'; panel.style.borderRight = '2px solid var(--dark)';
      panel.style.borderTop = '2px solid var(--dark)'; panel.style.borderBottom = '2px solid var(--dark)';

      if (side === 'right') {
        edge.style.top = '0'; edge.style.right = '0'; edge.style.width = '28px'; edge.style.height = '100%';
        panel.style.top = '0'; panel.style.right = '0'; panel.style.width = '360px'; panel.style.maxWidth = 'calc(100% - 28px)'; panel.style.height = '100%';
        panel.style.borderRight = '0'; panel.style.borderRadius = '22px 0 0 22px'; panel.style.boxShadow = '-5px 5px 0 var(--dark)';
      }
    },

    setSettingsDrawerOffset: function(offset, animated) {
      var backdrop = document.getElementById('settingsBackdrop');
      var panel = document.getElementById('settingsPanel');
      if (!backdrop || !panel) return;
      var side = this.settingsSide();
      var size = this.settingsPanelSize();
      var clamped = window.utils.clamp(Number(offset || 0), 0, size);
      window.localDeviceState.settingsOffset = clamped;
      panel.style.transition = animated ? 'transform 180ms ease' : 'none';
      if (side === 'right') panel.style.transform = 'translateX(' + clamped + 'px)';
      var openness = 1 - (clamped / Math.max(size, 1));
      backdrop.classList.toggle('open', openness > 0.001);
      backdrop.style.background = 'rgba(23, 23, 23, ' + (0.28 * openness) + ')';
    },

    syncSettingsDrawer: function(open, animated) {
      window.localDeviceState.settingsOpen = !!open;
      this.setSettingsDrawerOffset(open ? 0 : this.settingsPanelSize(), animated !== false);
    },

    switchView: function(newIndex) {
      if (newIndex < 0 || newIndex >= this.views.length) return;
      
      // Hide current
      var currentId = this.views[window.localDeviceState.currentViewIndex];
      var currentEl = document.getElementById(currentId);
      if (currentEl) {
        currentEl.classList.remove('active');
        currentEl.classList.add('hidden');
      }

      // Update index and show new
      window.localDeviceState.currentViewIndex = newIndex;
      try { localStorage.setItem('currentViewIndex', newIndex); } catch (_) {}
      
      var newId = this.views[newIndex];
      var newEl = document.getElementById(newId);
      if (newEl) {
        newEl.classList.remove('hidden');
        newEl.classList.add('active');
      }

      // Update indicators
      var indicators = document.getElementById('viewIndicators');
      if (indicators) {
        var dots = indicators.getElementsByClassName('dot');
        for (var i = 0; i < dots.length; i++) {
          dots[i].classList.toggle('active', i === newIndex);
        }
      }
    },

    handleTouchStart: function(event) {
      var panel = document.getElementById('settingsPanel');
      var edge = document.getElementById('settingsEdgeZone');
      if (!panel || !edge || !event.touches || !event.touches.length) return;
      var touch = event.touches[0];
      var panelRect = panel.getBoundingClientRect();
      var edgeRect = edge.getBoundingClientRect();
      var inEdge = touch.clientX >= edgeRect.left && touch.clientX <= edgeRect.right && touch.clientY >= edgeRect.top && touch.clientY <= edgeRect.bottom;

      if (!window.localDeviceState.settingsOpen) {
        if (!inEdge) {
          // If not in edge, start a view swipe gesture
          this.beginGesture('viewSwipe', touch);
          return;
        }
        this.syncSettingsDrawer(false, false);
        this.beginGesture('open', touch);
        return;
      }
      if (event.target && event.target.closest && event.target.closest('button,input,select,textarea,label,a')) return;
      if (touch.clientX >= panelRect.left && touch.clientX <= panelRect.right && touch.clientY >= panelRect.top && touch.clientY <= panelRect.bottom) {
        this.beginGesture('close', touch);
      }
    },

    beginGesture: function(mode, touch) {
      this.settingsGesture.active = true;
      this.settingsGesture.mode = mode;
      this.settingsGesture.startX = touch.clientX;
      this.settingsGesture.startY = touch.clientY;
      this.settingsGesture.startOffset = window.localDeviceState.settingsOffset;
    },

    handleTouchMove: function(event) {
      if (!this.settingsGesture.active || !event.touches || !event.touches.length) return;
      var touch = event.touches[0];
      var dx = touch.clientX - this.settingsGesture.startX;
      var dy = touch.clientY - this.settingsGesture.startY;
      
      var primary = (window.localDeviceState.orientation === 90 || window.localDeviceState.orientation === 270) ? Math.abs(dy) : Math.abs(dx);
      var secondary = (window.localDeviceState.orientation === 90 || window.localDeviceState.orientation === 270) ? Math.abs(dx) : Math.abs(dy);

      if (this.settingsGesture.mode === 'viewSwipe') {
        // For view swipe, we don't necessarily need to track movement visually, 
        // but we might want to prevent default if it's mostly horizontal.
        if (primary > secondary + 8) event.preventDefault();
        return;
      }

      if (secondary > primary + 8) return;

      var delta = dx; // Simplified for right-side drawer default
      if (window.localDeviceState.orientation === 90) delta = dy;
      else if (window.localDeviceState.orientation === 180) delta = -dx;
      else if (window.localDeviceState.orientation === 270) delta = -dy;

      var size = this.settingsPanelSize();
      var offset = this.settingsGesture.startOffset;
      if (this.settingsGesture.mode === 'open') offset = window.utils.clamp(size + delta, 0, size);
      if (this.settingsGesture.mode === 'close') offset = window.utils.clamp(delta, 0, size);
      this.setSettingsDrawerOffset(offset, false);
      event.preventDefault();
    },

    finishGesture: function(event) {
      if (!this.settingsGesture.active) return;
      
      if (this.settingsGesture.mode === 'viewSwipe') {
        var touch = event && event.changedTouches && event.changedTouches[0];
        if (touch) {
          var dx = touch.clientX - this.settingsGesture.startX;
          var dy = touch.clientY - this.settingsGesture.startY;
          var delta = dx;
          if (window.localDeviceState.orientation === 90) delta = dy;
          else if (window.localDeviceState.orientation === 180) delta = -dx;
          else if (window.localDeviceState.orientation === 270) delta = -dy;

          var swipeThreshold = 60;
          if (delta < -swipeThreshold) {
            // Swiped Left -> Move Right
            this.switchView(window.localDeviceState.currentViewIndex + 1);
          } else if (delta > swipeThreshold) {
            // Swiped Right -> Move Left
            this.switchView(window.localDeviceState.currentViewIndex - 1);
          }
        }
        this.settingsGesture.active = false;
        return;
      }

      var size = this.settingsPanelSize();
      var shouldOpen = window.localDeviceState.settingsOffset < size * 0.45;
      this.syncSettingsDrawer(shouldOpen, true);
      this.settingsGesture.active = false;
    },

    mergeHassData: function(partial) {
      window.KINDLE_HASS_DATA = window.KINDLE_HASS_DATA || {};
      if (partial.status) window.KINDLE_HASS_DATA.status = partial.status;
      if (partial.music) window.KINDLE_HASS_DATA.music = Object.assign({}, window.KINDLE_HASS_DATA.music || {}, partial.music);
      if (partial.mail) window.KINDLE_HASS_DATA.mail = Object.assign({}, window.KINDLE_HASS_DATA.mail || {}, partial.mail);
      if (partial.agendaSummary) window.KINDLE_HASS_DATA.agendaSummary = partial.agendaSummary;
      if (partial.events) window.KINDLE_HASS_DATA.events = partial.events;
      this.renderData();
    },

    renderData: function() {
      var sourceData = Object.assign({}, window.KINDLE_DASHBOARD_DATA || {});
      if (window.KINDLE_HASS_DATA) {
        sourceData.status = window.KINDLE_HASS_DATA.status || sourceData.status;
        sourceData.music = Object.assign({}, sourceData.music || {}, window.KINDLE_HASS_DATA.music || {});
        sourceData.mail = Object.assign({}, sourceData.mail || {}, window.KINDLE_HASS_DATA.mail || {});
        sourceData.agendaSummary = window.KINDLE_HASS_DATA.agendaSummary || sourceData.agendaSummary;
        sourceData.events = window.KINDLE_HASS_DATA.events || sourceData.events;
      }

      var dataSignature = JSON.stringify({ base: window.KINDLE_DASHBOARD_DATA || {}, hass: window.KINDLE_HASS_DATA || {} });
      if (dataSignature === window.appGlobals.lastDataSignature) return;
      window.appGlobals.lastDataSignature = dataSignature;

      var data = Object.assign({}, window.APP_DEFAULTS, sourceData);
      var mail = Object.assign({}, window.APP_DEFAULTS.mail, data.mail || {});
      var music = Object.assign({}, window.APP_DEFAULTS.music, data.music || {});
      var events = Array.isArray(data.events) ? data.events : [];

      window.utils.setText('statusLine', data.status || window.APP_DEFAULTS.status);
      window.utils.setText('mailCount', mail.unread || (mail.items ? mail.items.length : 0) || 0);
      window.utils.setText('mailSubhead', mail.summary || window.APP_DEFAULTS.mail.summary);
      
      var mailHtml = (mail.items || []).slice(0, 2).map(function(m) {
        return window.uiModule.renderItem(m.from || 'Mail', m.subject || m.detail || '', m.when || '');
      }).join('');
      document.getElementById('mailItems').innerHTML = mailHtml || '<div class="empty">Inbox quiet</div>';

      window.utils.setText('eventCount', events.length);
      var eventHtml = events.slice(0, 2).map(function(e) {
        return window.uiModule.renderItem(e.title || 'Event', e.detail || e.where || '', e.time || '');
      }).join('');
      document.getElementById('eventItems').innerHTML = eventHtml || '<div class="empty">No events today</div>';

      // Update Full Calendar View
      window.utils.setText('fullEventCount', events.length);
      var fullEventHtml = events.map(function(e) {
        return window.uiModule.renderItem(e.title || 'Event', e.detail || e.where || '', e.time || '');
      }).join('');
      var fullListEl = document.getElementById('fullEventItems');
      if (fullListEl) {
        fullListEl.innerHTML = fullEventHtml || '<div class="empty">Your agenda is clear for the week</div>';
      }

      var musicBadgeEl = document.getElementById('musicBadge');
      if (musicBadgeEl) {
        musicBadgeEl.textContent = music.badge || 'IDLE';
        musicBadgeEl.onclick = function() { window.hassModule.toggleEntity(music.entityId, musicBadgeEl); };
      }
      window.utils.setText('musicSubhead', music.source || 'Music');

      var musicRows = [];
      if (music.track || music.artist) {
        var detail = [music.artist, music.album].filter(Boolean).join(' • ');
        var meta = [music.position, music.duration].filter(Boolean).join(' / ');
        musicRows.push(this.renderItem(music.track || 'Unknown track', detail, meta, 'pc-line'));
      }
      document.getElementById('musicItems').innerHTML = musicRows.join('') || '<div class="empty">Nothing playing</div>';
    },

    renderItem: function(title, detail, meta, extraClass) {
      return '<div class="item ' + (extraClass || '') + '"><div class="item-main"><div class="item-title">' + window.utils.escapeHtml(title) + '</div><div class="item-detail">' + window.utils.escapeHtml(detail || '') + '</div></div><div class="item-meta">' + window.utils.escapeHtml(meta || '') + '</div></div>';
    },

    setTheme: function(theme) {
      var isDark = theme === 'dark';
      document.body.classList.toggle('dark-theme', isDark);
      window.utils.setText('settingsThemeValue', isDark ? 'Dark' : 'Light');
      var btnLight = document.getElementById('themeBtnLight');
      var btnDark = document.getElementById('themeBtnDark');
      if (btnLight) btnLight.classList.toggle('active', !isDark);
      if (btnDark) btnDark.classList.toggle('active', isDark);
      try { localStorage.setItem('theme', theme); } catch (_) {}
    }
  };
})(window);
