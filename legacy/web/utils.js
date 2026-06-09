(function(window) {
  window.utils = {
    pad: function(n) { return String(n).padStart(2, '0'); },
    
    setText: function(id, value) {
      var el = document.getElementById(id);
      var text = String(value);
      if (el && el.textContent !== text) el.textContent = text;
    },

    clamp: function(value, min, max) {
      return Math.min(max, Math.max(min, value));
    },

    escapeHtml: function(value) {
      return String(value == null ? '' : value).replace(/[&<>"']/g, function(c) {
        return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c];
      });
    },

    normalizeOrientation: function(value) {
      var numeric = Number(value);
      if (!Number.isFinite(numeric)) return 270;
      var normalized = ((numeric % 360) + 360) % 360;
      return normalized === 0 || normalized === 90 || normalized === 180 || normalized === 270 ? normalized : 270;
    },

    getQueryOrientation: function() {
      var params = new URLSearchParams(location.search || '');
      return this.normalizeOrientation(params.get('orientation'));
    },

    formatMediaTime: function(value) {
      if (value == null || value === '' || value === 'unknown' || value === 'unavailable') return '';
      var seconds = Math.max(0, Math.floor(Number(value) || 0));
      return Math.floor(seconds / 60) + ':' + this.pad(seconds % 60);
    }
  };
})(window);
