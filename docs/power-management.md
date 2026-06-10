# Power management findings (Kindle KHF)

## Current behavior

`scripts/kindle/launch.sh` calls:

```sh
lipc-set-prop -i com.lab126.powerd wakeUp 1
lipc-set-prop -i com.lab126.powerd preventScreenSaver 1
```

This forces the device fully awake (CPU + backlight) at all times. E-ink
holds its image with zero power, so this is *not* required just to keep
content visible — it's a real battery drain with no display benefit.

## Suspend-to-RAM test (2026-06-10)

Tested `echo mem > /sys/power/state` with an RTC wake alarm
(`/sys/class/rtc/rtc0/wakealarm`):

```sh
echo 0 > /sys/class/rtc/rtc0/wakealarm
echo +20 > /sys/class/rtc/rtc0/wakealarm
echo mem > /sys/power/state
```

- Suspend/resume cycle works, woken correctly by RTC after 20s.
- **WiFi fully powers down during suspend.** dmesg on resume shows:
  - `bcmsdh_sdmmc_resume`
  - `dhdsdio_download_code_file ... finished download fw_image_size=369119`
  - `Register interface [wlan0] MAC: 68:e4:7c`
- Resume of devices took ~424ms total (`PM: resume devices took 0.420 seconds`),
  but that's just driver re-init — WiFi still needs reassociation + DHCP/auth
  on top before network is usable again.

## `standby` test (2026-06-10)

`echo standby > /sys/power/state` with RTC wakealarm (+15s): **identical
behavior to `mem`** on this hardware — same `bcmsdh_sdmmc_suspend`/wifi GPIO
off/fw reload sequence, same ~424ms resume cost. No advantage over `mem`.

## ⚠️ `freeze` — CRASHED THE DEVICE (2026-06-10)

Tried `echo freeze > /sys/power/state` **without setting an RTC wakealarm**
first. Device went completely unresponsive over network/SSH (`No route to
host`) and got stuck in a boot loop — required hard power-button reset to
recover.

**Do not use `freeze` without a confirmed wake source, and treat it as
untested/risky on this hardware even with one.** `mem`/`standby` are the only
states confirmed safe so far (both require an RTC wakealarm set beforehand,
or the device will sleep forever with no way to wake it remotely).

## Wake-to-network timing (2026-06-10, post-reboot)

Measured via dmesg timestamps across one `mem` suspend/resume cycle
(RTC wakealarm +15s):

| Event | t (s) |
|---|---|
| `PM: suspend exit` (resume starts) | 256.924 |
| WiFi begins association (`Connecting with ... ssid`) | 257.068 |
| `wl_bss_connect_done succeeded` (associated) | 257.988 |

**~1.06s from resume to WiFi associated** (driver/fw reload ~424ms is
included in that). DHCP renewal time not captured but lease should
still be valid (short suspend), so likely minimal extra.

For a 60s wake cycle, ~1-1.5s WiFi overhead ≈ **2-2.5% duty cycle** —
seems acceptable.

## Implication for "sleep but keep polling" design

Each wake-from-suspend cycle pays a WiFi reconnect cost (firmware reload +
reassoc + DHCP/auth, likely 1s+) before HA/PC status polling can succeed.
Options to consider:

1. Accept the reconnect cost per wake (e.g. wake every 60s via RTC, poll,
   sleep again) — simplest, but adds latency + reconnect overhead each cycle.
2. Try `/sys/power/state` = `standby` or `freeze` instead of `mem` — these
   are lighter suspend states (`cat /sys/power/state` shows `freeze standby
   mem` all available) and may keep the WiFi chip powered/associated.
   **Not yet tested.**
3. Drop `wakeUp`/`preventScreenSaver` entirely and let powerd's normal
   idle/suspend handle it, only intervening to ensure RTC wake for clock
   ticks.

## Recommendation

A 60s `mem`-suspend/wake/poll/refresh cycle looks viable: ~2-2.5% duty
cycle for WiFi reinit, `mem`/`standby` confirmed safe with RTC wakealarm.

- Align RTC wakealarm to the next wall-clock minute boundary (not a
  fixed +60s from arbitrary suspend time), matching the existing clock
  goroutine logic in `main.go` — keeps the displayed clock accurate to
  the minute without extra wake cycles.
- Each wake: poll HA/PC status, redraw, set next wakealarm to next
  minute boundary, suspend (`mem`) again.
- Drop `wakeUp`/`preventScreenSaver` forcing entirely — they fight the
  suspend cycle.
- **Never call `freeze`** — confirmed to crash/hang this hardware with
  no safe recovery path other than a hard power-button reset.
