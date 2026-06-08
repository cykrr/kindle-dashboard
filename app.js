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
      var response = await fetch('version.txt?ts=' + Date.now());
      if (!response.ok) return;
      var version = (await response.text()).trim();
      if (!window.appGlobals.dashboardVersion) {
        window.appGlobals.dashboardVersion = version;
      } else if (window.appGlobals.dashboardVersion !== version) {
        console.log("New version detected, reloading...");
        window.location.reload();
      }
    } catch (_) {}
  }

  function init() {
    // 1. Initial State
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
    shell.addEventListener('touchend', function() { window.uiModule.finishGesture(); }, { passive: true });
    shell.addEventListener('touchcancel', function() { window.uiModule.finishGesture(); }, { passive: true });

    // 4. Initial Load
    window.uiModule.updateOrientationUi(window.localDeviceState.orientation);
    window.uiModule.renderData();
    loadData();
    window.apiModule.loadLocalSettings(true);
    
    // 5. Start Heartbeat
    heartbeat();
    setInterval(heartbeat, window.APP_CONFIG.POLL_MS);

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
