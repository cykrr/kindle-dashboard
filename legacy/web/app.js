(function(window) {
  function loadData() {
    var old = document.getElementById('data-js');
    if (old) old.remove();
    var script = document.createElement('script');
    script.id = 'data-js';
    script.src = 'data.js?ts=' + Date.now();
    script.onload = function() { window.uiModule.renderData(); };
    script.onerror = function() { window.uiModule.renderData(); };
    document.body.appendChild(script);
  }

  function heartbeat() {
    window.timeModule.tick();
    var now = Date.now();
    
    if (window.APP_CONFIG.DATA_RELOAD_MS > 0 && now - window.appGlobals.lastDataLoad >= window.APP_CONFIG.DATA_RELOAD_MS) {
      window.appGlobals.lastDataLoad = now;
      loadData();
    }
    
    if (now - window.appGlobals.lastHassConnectAttempt >= window.APP_CONFIG.HASS_RECONNECT_MS) {
      window.appGlobals.lastHassConnectAttempt = now;
      window.hassModule.connect();
    }
    
    window.apiModule.loadLocalSettings(false);
    window.hassModule.requestCalendarEvents(false);
    checkUpdates();
  }

  async function checkUpdates() {
    try {
      var url = window.APP_CONFIG.LOCAL_API_BASE + '/version?ts=' + Date.now();
      var response = await fetch(url);
      if (!response.ok) return;
      var version = (await response.text()).trim();
      console.log("Check version: current=" + window.appGlobals.dashboardVersion + " remote=" + version);
      if (!window.appGlobals.dashboardVersion) {
        window.appGlobals.dashboardVersion = version;
      } else if (window.appGlobals.dashboardVersion !== version) {
        console.log("New version detected, reloading...");
        window.location.reload();
      }
    } catch (_) {}
  }

  function init() {
    // 6. Global action shortcuts for HTML onclicks
    window.openSettings = function() { window.uiModule.syncSettingsDrawer(true, true); window.apiModule.loadLocalSettings(true); };
    window.closeSettings = function(e) { if (!e || e.target.id === 'settingsBackdrop') window.uiModule.syncSettingsDrawer(false, true); };
    window.previewBrightness = function() {
      var slider = document.getElementById('settingsBrightnessSlider');
      var percent = window.utils.clamp(Number(slider && slider.value || 0), 0, 100);
      window.utils.setText('settingsBrightnessValue', percent + '%');
    };
    window.saveBrightness = function() { window.apiModule.saveBrightness(); };
    window.saveOrientation = function(v) { window.apiModule.saveOrientation(v); };
    window.setTheme = function(t) { window.uiModule.setTheme(t); };
    window.toggleEntity = function(id, el) { window.hassModule.toggleEntity(id, el); };
    
    window.pcPlaying = false;

    window.executePcMacro = function(action, el) {
      var pcIp = "10.20.0.2"; // PC IP from ipconfig
      var apiKey = "your-super-secret-key"; // Matches Go daemon
      var url = "http://" + pcIp + ":8080/execute?action=" + action + "&key=" + apiKey;
      console.log("Calling PC Macro: " + url);

      if (el) el.classList.add('active-tap');

      // Optimistic toggle for play/pause
      if (action === 'play_pause') {
        window.pcPlaying = !window.pcPlaying;
        window.updatePlayPauseIcon(window.pcPlaying);
      }
      
      fetch(url)
        .then(function(res) {
          if (!res.ok) throw new Error("PC unreachable");
          console.log("PC Macro executed: " + action);
        })
        .catch(function(err) {
          console.error("PC Macro error:", err.message || err);
          // Revert on failure
          if (action === 'play_pause') {
            window.pcPlaying = !window.pcPlaying;
            window.updatePlayPauseIcon(window.pcPlaying);
          }
        })
        .finally(function() {
          if (el) setTimeout(function() { el.classList.remove('active-tap'); }, 150);
        });
    };

    window.updatePlayPauseIcon = function(playing) {
      var playIcon = document.getElementById('pcPlayPauseIcon');
      if (!playIcon) return;
      if (playing) {
        playIcon.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="6" y="4" width="4" height="16"></rect><rect x="14" y="4" width="4" height="16"></rect></svg>';
      } else {
        playIcon.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"></polygon></svg>';
      }
    };

    window.confirmPcMacro = function(action, message, el) {
      if (confirm(message)) {
        window.executePcMacro(action, el);
      }
    };

    window.updatePcStatusUi = function(status) {
      // Gaming Mode Update
      var gamingEl = document.getElementById('gamingModeLabel');
      if (gamingEl) {
        gamingEl.textContent = (status.gaming_mode === 'power' ? 'Gaming: ON' : 'Gaming: OFF');
        var btn = gamingEl.closest('.macro-btn');
        if (btn) {
          btn.style.backgroundColor = (status.gaming_mode === 'power' ? 'var(--dark)' : 'var(--panel)');
          btn.querySelector('.macro-label').style.color = (status.gaming_mode === 'power' ? 'var(--bg)' : 'var(--ink)');
          btn.querySelector('.macro-icon').style.color = (status.gaming_mode === 'power' ? 'var(--bg)' : 'var(--ink)');
        }
      }
      
      // Monitor Toggle Update
      var monitorEl = document.getElementById('monitorToggleLabel');
      if (monitorEl) {
        monitorEl.textContent = (status.monitor_on ? 'Monitors: ON' : 'Monitors: OFF');
        var mBtn = monitorEl.closest('.macro-btn');
        if (mBtn) {
          mBtn.style.backgroundColor = (!status.monitor_on ? 'var(--dark)' : 'var(--panel)');
          mBtn.querySelector('.macro-label').style.color = (!status.monitor_on ? 'var(--bg)' : 'var(--ink)');
          mBtn.querySelector('.macro-icon').style.color = (!status.monitor_on ? 'var(--bg)' : 'var(--ink)');
        }
      }

      // Now Playing Update
      window.utils.setText('pcTrackTitle', status.track || 'Not Playing');
      window.utils.setText('pcTrackArtist', status.artist || (status.status === 'Idle' ? 'PC Idle' : 'PC'));

      // Play/Pause icon toggle
      var playIcon = document.getElementById('pcPlayPauseIcon');
      if (playIcon) {
        if (status.status === 'Playing') {
          playIcon.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="6" y="4" width="4" height="16"></rect><rect x="14" y="4" width="4" height="16"></rect></svg>';
        } else {
          playIcon.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"></polygon></svg>';
        }
      }
    };

    window.updatePcStatus = function() {
      var pcIp = "10.20.0.2";
      var apiKey = "your-super-secret-key";
      var url = "http://" + pcIp + ":8080/status?key=" + apiKey + "&ts=" + Date.now();

      fetch(url)
        .then(function(res) { return res.json(); })
        .then(function(status) { window.updatePcStatusUi(status); })
        .catch(function(err) { console.error("Status error:", err); });
    };

    // SSE connection for real-time PC status updates
    window.pcEventSource = null;

    window.connectPcEvents = function() {
      var pcIp = "10.20.0.2";
      var apiKey = "your-super-secret-key";
      var url = "http://" + pcIp + ":8080/events?key=" + apiKey;

      // Close any existing connection
      if (window.pcEventSource) {
        window.pcEventSource.close();
      }

      try {
        window.pcEventSource = new EventSource(url);

        window.pcEventSource.onmessage = function(event) {
          try {
            var status = JSON.parse(event.data);
            window.updatePcStatusUi(status);
          } catch (e) {
            console.error("SSE parse error:", e);
          }
        };

        window.pcEventSource.onerror = function() {
          console.log("SSE disconnected, falling back to polling");
          window.pcEventSource.close();
          window.pcEventSource = null;
          // Fallback: poll every 10 seconds
          if (!window._pcPollInterval) {
            window._pcPollInterval = setInterval(function() {
              window.updatePcStatus();
            }, 10000);
          }
        };

        window.pcEventSource.onopen = function() {
          console.log("SSE connected");
          // If we had a polling fallback running, clear it
          if (window._pcPollInterval) {
            clearInterval(window._pcPollInterval);
            window._pcPollInterval = null;
          }
        };

      } catch (e) {
        console.error("SSE not supported, polling:", e);
        if (!window._pcPollInterval) {
          window._pcPollInterval = setInterval(function() {
            window.updatePcStatus();
          }, 10000);
        }
      }
    };

    window.localDeviceState.orientation = window.utils.getQueryOrientation();
    
    var savedTheme = 'light';
    try { savedTheme = localStorage.getItem('theme') || 'light'; } catch (_) {}
    window.uiModule.setTheme(savedTheme);

    // 2. Initial UI Layout
    window.uiModule.applyShellOrientation();
    window.uiModule.syncSettingsDrawer(false, false);
    
    // 3. Global Event Listeners
    window.addEventListener('resize', function() {
      window.uiModule.applyShellOrientation();
      window.uiModule.syncSettingsDrawer(window.localDeviceState.settingsOpen, false);
    });

    var shell = document.getElementById('appShell');
    shell.addEventListener('touchstart', function(e) { window.uiModule.handleTouchStart(e); }, { passive: true });
    shell.addEventListener('touchmove', function(e) { window.uiModule.handleTouchMove(e); }, { passive: false });
    shell.addEventListener('touchend', function(e) { window.uiModule.finishGesture(e); }, { passive: true });
    shell.addEventListener('touchcancel', function(e) { window.uiModule.finishGesture(e); }, { passive: true });

    // 4. Initial Load
    window.uiModule.updateOrientationUi(window.localDeviceState.orientation);
    
    var savedViewIndex = 1;
    try {
      var val = localStorage.getItem('currentViewIndex');
      if (val !== null) savedViewIndex = parseInt(val, 10);
    } catch (_) {}
    window.uiModule.switchView(savedViewIndex);
    
    window.uiModule.renderData();
    loadData();
    window.apiModule.loadLocalSettings(true);
    window.connectPcEvents();
    window.updatePcStatus();
    
    // 5. Start Heartbeat
    heartbeat();
    setInterval(heartbeat, window.APP_CONFIG.POLL_MS);

    window.manualReconnectHass = function() {
      window.appGlobals.hassAuthFailed = false;
      if (window.appGlobals.hassSocket) { try { window.appGlobals.hassSocket.close(); } catch (_) {} }
      window.appGlobals.hassSocket = null;
      window.appGlobals.lastHassConnectAttempt = Date.now();
      window.hassModule.connect();
    };
  }

  // Ensure config is present before starting
  function checkConfig() {
    var cfg = window.HASS_CONFIG || {};
    var token = cfg.token || cfg.HASS_TOKEN || '';
    if (!token) window.utils.setText('connApiInfo', 'API Key: Missing');
    else if (token.indexOf('YOUR_') === 0 || token === 'placeholder') window.utils.setText('connApiInfo', 'API Key: Placeholder');
    else window.utils.setText('connApiInfo', 'API Key: Present');
  }

  window.addEventListener('DOMContentLoaded', function() {
    checkConfig();
    init();
  });
})(window);
