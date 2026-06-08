(function(window) {
  window.APP_DEFAULTS = {
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

  window.APP_CONFIG = {
    POLL_MS: 15000,
    DATA_RELOAD_MS: 0,
    HASS_RECONNECT_MS: 60000,
    HASS_CALENDAR_RELOAD_MS: 900000,
    LOCAL_API_BASE: 'http://127.0.0.1:8177',
    SETTINGS_EDGE_PX: 36
  };

  window.localDeviceState = {
    brightness: 0,
    brightnessMax: 2399,
    batteryLevel: null,
    batteryStatus: '',
    orientation: 270, // will be updated from query or local storage
    currentViewIndex: 1, // 0: calendar, 1: dashboard, 2: launcher
    apiReady: false,
    settingsOpen: false,
    settingsOffset: 0
  };

  window.appGlobals = {
    lastMinuteKey: '',
    lastDateKey: '',
    lastWeekKey: '',
    lastDataLoad: 0,
    lastHassConnectAttempt: 0,
    lastLocalSettingsLoad: 0,
    hassSocket: null,
    hassAuthFailed: false,
    hassMessageId: 1,
    hassPendingRequests: {},
    lastCalendarRefresh: 0,
    lastDataSignature: null,
    dashboardVersion: null
  };
})(window);
