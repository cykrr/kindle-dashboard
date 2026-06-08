(function(window) {
  window.hassModule = {
    wsUrl: function(rawUrl) {
      var url = String(rawUrl || '').replace(/\/$/, '');
      if (!url) return '';
      if (url.indexOf('https://') === 0) return 'wss://' + url.slice('https://'.length) + '/api/websocket';
      if (url.indexOf('http://') === 0) return 'ws://' + url.slice('http://'.length) + '/api/websocket';
      return url.replace(/^\/+/, '').replace(/^/, 'wss://') + '/api/websocket';
    },

    connect: function() {
      if (window.appGlobals.hassAuthFailed) {
        window.utils.setText('connStatus', 'Auth Failed');
        return;
      }
      var cfg = window.HASS_CONFIG || {};
      var token = cfg.token || cfg.HASS_TOKEN || '';
      var wsUrl = this.wsUrl(cfg.url || cfg.HASS_URL || '');
      if (!wsUrl || !token || token.indexOf('YOUR_') === 0 || token === 'placeholder' || wsUrl.indexOf('your-') !== -1) {
        window.utils.setText('connStatus', 'Config Missing');
        return;
      }
      if (window.appGlobals.hassSocket && (window.appGlobals.hassSocket.readyState === WebSocket.OPEN || window.appGlobals.hassSocket.readyState === WebSocket.CONNECTING)) return;

      try {
        window.utils.setText('connStatus', 'Connecting…');
        window.appGlobals.hassSocket = new WebSocket(wsUrl);
      } catch (error) {
        window.utils.setText('connStatus', 'Error');
        window.utils.setText('musicSubhead', 'HA WebSocket unavailable; using local data.js');
        return;
      }

      window.appGlobals.hassSocket.onmessage = (event) => {
        let msg;
        try { msg = JSON.parse(event.data); } catch (_) { return; }

        if (msg.type === 'auth_required') {
          window.utils.setText('connStatus', 'Authenticating…');
          window.appGlobals.hassSocket.send(JSON.stringify({ type: 'auth', access_token: token }));
          return;
        }

        if (msg.type === 'auth_ok') {
          window.utils.setText('connStatus', 'Connected');
          window.utils.setText('musicSubhead', 'Home Assistant connected');
          window.appGlobals.hassSocket.send(JSON.stringify({ id: window.appGlobals.hassMessageId++, type: 'get_states' }));
          window.appGlobals.hassSocket.send(JSON.stringify({ id: window.appGlobals.hassMessageId++, type: 'subscribe_events', event_type: 'state_changed' }));
          this.requestCalendarEvents(true);
          return;
        }

        if (msg.type === 'auth_invalid') {
          window.appGlobals.hassAuthFailed = true;
          window.utils.setText('connStatus', 'Auth Failed');
          window.utils.setText('musicSubhead', 'HA auth failed permanently');
          if (window.appGlobals.hassSocket) {
            try { window.appGlobals.hassSocket.close(); } catch (_) {}
          }
          return;
        }

        if (msg.type === 'result' && window.appGlobals.hassPendingRequests[msg.id] === 'calendar_events') {
          delete window.appGlobals.hassPendingRequests[msg.id];
          if (msg.success !== false) window.uiModule.mergeHassData(this.parseCalendarData(msg.result));
          return;
        }

        if (msg.type === 'result' && Array.isArray(msg.result)) {
          msg.result.forEach(state => this.handleState(state));
          return;
        }

        if (msg.type === 'event' && msg.event && msg.event.event_type === 'state_changed') {
          this.handleState(msg.event.data && msg.event.data.new_state);
        }
      };

      window.appGlobals.hassSocket.onclose = () => {
        window.appGlobals.hassSocket = null;
        if (window.appGlobals.hassAuthFailed) {
          window.utils.setText('connStatus', 'Auth Failed');
        } else {
          window.utils.setText('connStatus', 'Disconnected');
          window.utils.setText('musicSubhead', 'HA disconnected; will reconnect');
        }
      };

      window.appGlobals.hassSocket.onerror = () => {
        window.utils.setText('connStatus', 'Error');
        window.utils.setText('musicSubhead', 'HA WebSocket error; using local data.js');
      };
    },

    handleState: function(state) {
      var cfg = window.HASS_CONFIG || {};
      var musicEntity = cfg.entity || cfg.musicEntity || 'media_player.googlehome1844';
      var mailEntity = cfg.mailEntity || 'sensor.imap_me_messages';
      var calendars = this.calendarEntities();
      var brightnessEntity = cfg.brightnessEntity;

      if (!state || !state.entity_id) return;
      if (state.entity_id === musicEntity) window.uiModule.mergeHassData(this.parseMusicData(state));
      if (state.entity_id === mailEntity) window.uiModule.mergeHassData(this.parseMailData(state));
      if (calendars.indexOf(state.entity_id) !== -1) this.requestCalendarEvents(true);
      if (brightnessEntity && state.entity_id === brightnessEntity) this.handleBrightnessState(state);
    },

    handleBrightnessState: function(state) {
      let percent = null;
      if (state.domain === 'light' || state.entity_id.startsWith('light.')) {
         if (state.state === 'off') {
           percent = 0;
         } else if (state.attributes && state.attributes.brightness !== undefined) {
           percent = Math.round((state.attributes.brightness / 255) * 100);
         }
      } else {
         const val = Number(state.state);
         if (!Number.isNaN(val)) percent = val;
      }

      if (percent !== null) {
         const currentPercent = window.uiModule.brightnessToPercent(window.localDeviceState.brightness, window.localDeviceState.brightnessMax);
         if (Math.abs(percent - currentPercent) > 2) { 
           window.apiModule.saveBrightness(percent);
         }
      }
    },

    publishBrightnessToHass: function(percent) {
      if (!window.appGlobals.hassSocket || window.appGlobals.hassSocket.readyState !== WebSocket.OPEN) return;
      var cfg = window.HASS_CONFIG || {};
      var entityId = cfg.brightnessEntity;
      if (!entityId) return;

      var domain = entityId.split('.')[0];
      if (domain === 'number' || domain === 'input_number') {
        window.appGlobals.hassSocket.send(JSON.stringify({
          id: window.appGlobals.hassMessageId++,
          type: 'call_service',
          domain: domain,
          service: 'set_value',
          target: { entity_id: entityId },
          service_data: { value: percent }
        }));
      } else if (domain === 'light') {
        window.appGlobals.hassSocket.send(JSON.stringify({
          id: window.appGlobals.hassMessageId++,
          type: 'call_service',
          domain: domain,
          service: 'turn_on',
          target: { entity_id: entityId },
          service_data: { brightness_pct: percent }
        }));
      }
    },

    toggleEntity: function(entityId, element) {
      if (element) {
        element.classList.add('active-tap');
        setTimeout(function() { element.classList.remove('active-tap'); }, 500);
      }

      if (!window.appGlobals.hassSocket || window.appGlobals.hassSocket.readyState !== WebSocket.OPEN) {
        console.warn("HA Socket not connected. Cannot toggle", entityId);
        return;
      }

      var domain = entityId.split('.')[0];
      window.appGlobals.hassSocket.send(JSON.stringify({
        id: window.appGlobals.hassMessageId++,
        type: 'call_service',
        domain: domain,
        service: 'toggle',
        target: { entity_id: entityId }
      }));
    },

    requestCalendarEvents: function(force) {
      if (!window.appGlobals.hassSocket || window.appGlobals.hassSocket.readyState !== WebSocket.OPEN) return;
      var nowMs = Date.now();
      if (!force && nowMs - window.appGlobals.lastCalendarRefresh < window.APP_CONFIG.HASS_CALENDAR_RELOAD_MS) return;
      window.appGlobals.lastCalendarRefresh = nowMs;

      var entities = this.calendarEntities();
      if (!entities.length) return;
      var start = new Date();
      start.setHours(0, 0, 0, 0);
      var end = new Date(start);
      end.setDate(end.getDate() + 7);

      var id = window.appGlobals.hassMessageId++;
      window.appGlobals.hassPendingRequests[id] = 'calendar_events';
      window.appGlobals.hassSocket.send(JSON.stringify({
        id: id,
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
    },

    calendarEntities: function() {
      var cfg = window.HASS_CONFIG || {};
      var raw = cfg.calendarEntities || cfg.calendarEntity || ['calendar.it', 'calendar.calendario'];
      var parts = [];
      var arr = Array.isArray(raw) ? raw : [raw];
      arr.forEach(function(v) { String(v).split(',').forEach(function(part) { parts.push(part); }); });
      return parts.map(function(v) { return v.trim(); }).filter(Boolean);
    },

    parseMusicData: function(state) {
      var attrs = state.attributes || {};
      var playerState = state.state || 'unknown';
      var device = attrs.friendly_name || state.entity_id || 'Home Assistant player';
      var source = attrs.app_name || attrs.source || 'Home Assistant';
      
      var badge = 'IDLE';
      var s = String(playerState).toLowerCase();
      if (s === 'playing') badge = 'PLAYING';
      else if (s === 'paused') badge = 'PAUSED';
      else if (s === 'off' || s === 'standby') badge = 'OFF';

      return {
        status: 'Home Assistant • ' + device,
        music: {
          entityId: state.entity_id,
          badge: badge,
          summary: [device, source, playerState].filter(Boolean).join(' • '),
          state: playerState,
          track: attrs.media_title || device,
          artist: attrs.media_artist || attrs.media_album_artist || '',
          album: attrs.media_album_name || '',
          position: window.utils.formatMediaTime(attrs.media_position),
          duration: window.utils.formatMediaTime(attrs.media_duration),
          source: source,
          items: [
            { label: 'Device', value: device },
            { label: 'Player', value: source },
            { label: 'State', value: playerState },
            { label: 'Updated', value: String(state.last_updated || state.last_changed || '').replace('T', ' ').replace('+00:00', 'Z').slice(0, 19) }
          ]
        }
      };
    },

    parseMailData: function(state) {
      var attrs = state.attributes || {};
      var unread = Number(state.state) || Number(attrs.unread) || Number(attrs.unseen) || 0;
      var rawMessages = attrs.messages || attrs.message || attrs.emails || attrs.email || attrs.subjects || attrs.subject || [];
      if (!Array.isArray(rawMessages)) rawMessages = [rawMessages];
      
      var items = rawMessages.slice(0, 4).map(function(m, index) {
        if (typeof m === 'string') return { from: 'Mail', subject: m, when: '' };
        return {
          from: m.from || m.sender || m.name || m.mailbox || 'Mail',
          subject: m.subject || m.title || m.summary || m.body || ('Message ' + (index + 1)),
          when: m.date || m.when || m.received || ''
        };
      });

      if (!items.length && unread > 0) {
        var cfg = window.HASS_CONFIG || {};
        var mailLabel = cfg.mailLabel || attrs.friendly_name || 'Mail';
        items.push({ from: mailLabel, subject: unread + ' unread message' + (unread === 1 ? '' : 's'), when: '' });
      }

      return {
        mail: {
          unread: unread,
          summary: unread ? (unread + ' unread from ' + ((window.HASS_CONFIG && window.HASS_CONFIG.mailLabel) || 'Mail')) : 'Inbox quiet',
          items: items
        }
      };
    },

    parseCalendarData: function(result) {
      var response = (result && result.response) || result || {};
      var events = [];
      var self = this;
      Object.keys(response).forEach(function(entity) {
        var bucket = response[entity] || {};
        var evts = bucket.events || [];
        if (!Array.isArray(evts)) evts = [evts];
        evts.forEach(function(event) {
          events.push({
            time: self.formatEventTime(event),
            title: event.summary || event.title || 'Calendar event',
            detail: event.location || event.description || entity
          });
        });
      });
      events.sort(function(a, b) { return String(a.time).localeCompare(String(b.time)); });
      return {
        agendaSummary: events.length ? 'Home Assistant calendar' : 'No upcoming calendar events',
        events: events.slice(0, 4)
      };
    },

    formatEventTime: function(event) {
      var start = event.start || {};
      var raw = start.dateTime || start.date || event.start_time || event.start;
      if (!raw) return '';
      if (start.date || /^\d{4}-\d{2}-\d{2}$/.test(String(raw))) return 'all day';
      var date = new Date(raw);
      if (Number.isNaN(date.getTime())) return '';
      return window.utils.pad(date.getHours()) + ':' + window.utils.pad(date.getMinutes());
    }
  };
})(window);
