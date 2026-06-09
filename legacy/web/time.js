(function(window) {
  window.timeModule = {
    tick: function() {
      var now = new Date();
      var h = now.getHours();
      var minuteKey = h + ':' + now.getMinutes();
      
      if (minuteKey !== window.appGlobals.lastMinuteKey) {
        window.appGlobals.lastMinuteKey = minuteKey;
        var greeting = h < 12 ? 'Good morning' : h < 18 ? 'Good afternoon' : 'Good evening';
        window.utils.setText('greeting', greeting);
        window.utils.setText('hour', window.utils.pad(h));
        window.utils.setText('minute', window.utils.pad(now.getMinutes()));
      }

      var dateKey = now.toDateString();
      if (dateKey !== window.appGlobals.lastDateKey) {
        window.appGlobals.lastDateKey = dateKey;
        window.utils.setText('dateLine', now.toLocaleDateString(undefined, { weekday: 'long', month: 'long', day: 'numeric' }));
        window.utils.setText('monthName', now.toLocaleDateString(undefined, { month: 'long' }));
        window.utils.setText('yearName', String(now.getFullYear()));
        this.renderWeek(now);
      }
    },

    renderWeek: function(now) {
      var target = document.getElementById('days');
      if (!target) return;
      var weekKey = now.getFullYear() + '-' + now.getMonth() + '-' + now.getDate();
      if (weekKey === window.appGlobals.lastWeekKey) return;
      window.appGlobals.lastWeekKey = weekKey;
      
      var start = new Date(now);
      var mondayOffset = (start.getDay() + 6) % 7;
      start.setDate(start.getDate() - mondayOffset);
      
      target.innerHTML = '';
      for (var i = 0; i < 7; i++) {
        var d = new Date(start);
        d.setDate(start.getDate() + i);
        var el = document.createElement('div');
        el.className = 'day' + (d.toDateString() === now.toDateString() ? ' today' : '');
        el.innerHTML = '<b>' + d.toLocaleDateString(undefined, { weekday: 'short' }) + '</b><span>' + d.getDate() + '</span>';
        target.appendChild(el);
      }
    }
  };
})(window);
