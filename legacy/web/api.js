(function(window) {
  window.apiModule = {
    fetchLocalText: async function(path, options) {
      const response = await fetch(window.APP_CONFIG.LOCAL_API_BASE + path, options || {});
      if (!response.ok) throw new Error('HTTP ' + response.status);
      return (await response.text()).trim();
    },

    loadLocalSettings: async function(force) {
      const now = Date.now();
      if (!force && now - window.appGlobals.lastLocalSettingsLoad < 60000) return;
      window.appGlobals.lastLocalSettingsLoad = now;
      try {
        const [brightnessText, maxText, orientationText, batteryLevelText, batteryStatusText] = await Promise.all([
          this.fetchLocalText('/brightness'),
          this.fetchLocalText('/brightness-max'),
          this.fetchLocalText('/orientation'),
          this.fetchLocalText('/battery-level'),
          this.fetchLocalText('/battery-status')
        ]);
        window.localDeviceState.brightness = Math.max(0, parseInt(brightnessText, 10) || 0);
        window.localDeviceState.brightnessMax = Math.max(1, parseInt(maxText, 10) || 2399);
        window.localDeviceState.batteryLevel = Math.max(0, parseInt(batteryLevelText, 10) || 0);
        window.localDeviceState.batteryStatus = String(batteryStatusText || '');
        window.localDeviceState.orientation = window.utils.normalizeOrientation(parseInt(orientationText, 10));
        window.localDeviceState.apiReady = true;
        
        window.uiModule.updateBrightnessUi(window.localDeviceState.brightness, window.localDeviceState.brightnessMax);
        window.uiModule.updateBatteryUi(window.localDeviceState.batteryLevel, window.localDeviceState.batteryStatus);
        window.uiModule.updateOrientationUi(window.localDeviceState.orientation);
        window.uiModule.applyShellOrientation();
        window.utils.setText('settingsStatus', 'On-device settings ready');
      } catch (_) {
        window.localDeviceState.apiReady = false;
        window.uiModule.updateOrientationUi(window.localDeviceState.orientation);
        window.uiModule.updateBatteryUi(null, '');
        window.utils.setText('brightnessQuickValue', '--');
        window.utils.setText('settingsStatus', 'Local device API unavailable');
      }
    },

    saveBrightness: async function(percentOverride) {
      const slider = document.getElementById('settingsBrightnessSlider');
      const button = document.getElementById('saveBrightnessBtn');
      let percent;
      if (percentOverride !== undefined && percentOverride !== null) {
        percent = window.utils.clamp(Number(percentOverride), 0, 100);
        if (slider) slider.value = percent;
      } else {
        percent = window.utils.clamp(Number(slider && slider.value || 0), 0, 100);
      }
      const raw = window.uiModule.percentToBrightness(percent, window.localDeviceState.brightnessMax);
      window.utils.setText('settingsStatus', 'Applying brightness…');
      if (button) button.disabled = true;
      try {
        const brightnessText = await this.fetchLocalText('/brightness?value=' + encodeURIComponent(raw), { method: 'POST' });
        window.localDeviceState.brightness = Math.max(0, parseInt(brightnessText, 10) || raw);
        window.localDeviceState.apiReady = true;
        window.uiModule.updateBrightnessUi(window.localDeviceState.brightness, window.localDeviceState.brightnessMax);
        window.utils.setText('settingsStatus', 'Brightness saved on device');
        window.hassModule.publishBrightnessToHass(percent);
      } catch (_) {
        window.utils.setText('settingsStatus', 'Failed to update brightness');
      }
      if (button) button.disabled = false;
    },

    saveOrientation: async function(value) {
      const orientation = window.utils.normalizeOrientation(value);
      window.utils.setText('settingsStatus', 'Applying rotation…');
      try {
        const orientationText = await this.fetchLocalText('/orientation?value=' + encodeURIComponent(orientation), { method: 'POST' });
        window.localDeviceState.orientation = window.utils.normalizeOrientation(parseInt(orientationText, 10));
        window.uiModule.updateOrientationUi(window.localDeviceState.orientation);
        window.uiModule.applyShellOrientation();
        window.utils.setText('settingsStatus', 'Rotation saved on device');
      } catch (_) {
        window.utils.setText('settingsStatus', 'Failed to update rotation');
      }
    }
  };
})(window);
