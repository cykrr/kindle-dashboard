window.HASS_CONFIG = {
  url: "http://homeassistant.local:8123",
  token: "YOUR_LONG_LIVED_ACCESS_TOKEN",

  // Optional Home Assistant cards/widgets. Leave omitted to disable.
  mailEntity: "sensor.mail_unread",
  mailLabel: "Mail",
  calendarEntities: ["calendar.family"],
  lightEntities: ["light.office", "cover.garage_door"],
  brightnessEntity: "number.kindle_frontlight",

  // Optional Windows macro daemon integration.
  pcMacroUrl: "http://YOUR_PC_IP:8765",
  pcMacroKey: "YOUR_PC_MACRO_KEY",

  // Optional launcher personalization. If omitted, the app uses its built-in
  // default launcher. Set [] for an empty launcher.
  launcherButtons: [
    { action: "pc_mode_toggle", icon: "pc_mode_toggle", label: "Mode" },
    { action: "mute_mic", icon: "mute_mic", label: "Mic" },
    { action: "monitor_toggle", icon: "monitor_toggle", label: "Monitor" },
    { action: "launch_chrome", icon: "launch_chrome", label: "Browser" },
    { action: "launch_mail", icon: "launch_mail", label: "Mail" },
    { action: "sleep", icon: "sleep", label: "Sleep" }
  ]
};
