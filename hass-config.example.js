window.HASS_CONFIG = {
  "url": "https://your-home-assistant-url",
  "token": "YOUR_LONG_LIVED_ACCESS_TOKEN",
  "entity": "media_player.your_speaker",
  "musicEntity": "media_player.your_speaker",
  "mailEntity": "sensor.imap_your_email",
  "calendarEntities": [
    "calendar.your_calendar"
  ],
  "lightEntities": [
    "light.your_main_light",
    "light.your_lamp"
  ],
  "pcMacroUrl": "http://10.20.0.2:8080",
  "pcMacroKey": "your-super-secret-key",
  "brightnessEntity": "input_number.kindle_brightness",
  "insecureSkipVerify": false
};
