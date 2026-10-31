# LEDLEDLED

A small, self-contained pixel-strip controller for Raspberry Pi. It renders
layered animations onto LED strips and streams them as **sACN (E1.31)** or
**Art-Net** to wireless pixel drivers such as the **SmartShow AirPixel Mini II**.
Phones, tablets and laptops join the Pi's own Wi-Fi network and control
everything from a browser.

- Single static Go binary with the web UI embedded. No runtime dependencies.
- Map any number of controllers and strips, mixing RGB and RGBW (GRB, GRBW, ...).
- **Groups**: drag across strips to pick pixel ranges. A group can span
  several strips and effects treat it as one continuous line.
- **Layers**: stack effects (solid, gradient, rainbow, noise, pulse, chase,
  sparkle, VU meter, spectrum) on groups with opacity and blend modes.
- **Tempo**: tap tempo, or lock to beats detected in the audio. Any layer
  can be tempo-synced so its speeds become "per beat".
- **Timeline**: a looping keyframe timeline measured in beats. Animate any
  colour or number (and layer opacity) with easing.
- **Audio reactivity** from a USB sound card or HAT: level, bass, mid,
  treble, beat and a 16-band spectrum drive modulators, audio effects and
  master "flash on beat".
- **Power estimation and limiting** per strip for 5 V and 12 V strips.
- **Library**: save and load named shows and reusable layer presets as JSON
  files on the Pi, and download or upload them.
- Live preview in the browser, multi-device sync, master dimmer, blackout,
  per-strip identify pattern.

---

## Contents

1. [How it fits together](#how-it-fits-together)
2. [Concepts](#concepts)
3. [Developing on your computer](#developing-on-your-computer)
4. [Setting up the Raspberry Pi](#setting-up-the-raspberry-pi)
5. [Setting up the AirPixels](#setting-up-the-airpixels)
6. [First hardware test checklist](#first-hardware-test-checklist)
7. [Tempo and the timeline](#tempo-and-the-timeline)
8. [Audio reactivity](#audio-reactivity)
9. [Power estimation and limiting](#power-estimation-and-limiting)
10. [Saving shows and presets](#saving-shows-and-presets)
11. [Troubleshooting](#troubleshooting)
12. [Project layout](#project-layout)
13. [HTTP API](#http-api)
14. [A note on Wails](#a-note-on-wails)
15. [TODO](#todo)

---

## How it fits together

```
 phones / laptops                     Raspberry Pi 4                         AirPixel Mini IIs
 ───────────────                 ─────────────────────────               ──────────────────
                 Wi-Fi "LEDLEDLED"  ┌──────────────────────┐ Wi-Fi "LEDLEDLED-PIX"
   browser  ◄──────────────────────►│ wlan (onboard)       │
  http://10.42.0.1                  │   10.42.0.1  5 GHz   │
                                    │                      │
                                    │  ledledled           │    sACN / Art-Net
                                    │   web UI + API (:80) │    unicast UDP
                                    │   render engine 40fps│ ───────────────────►  10.43.0.11  strip 1
                                    │   sACN/Art-Net out   │ ───────────────────►  10.43.0.12  strip 2
                                    │                      │ ───────────────────►  10.43.0.13  strip 3
                                    │ wlan (USB stick)     │
                                    │   10.43.0.1  2.4 GHz │
                                    └──────────────────────┘
```

Two separate access points keep the phones' chatter off the pixel network.
The AirPixels are 2.4 GHz only, so the pixel AP runs on the USB stick at
2.4 GHz. The Pi 4's onboard radio hosts the UI network on 5 GHz.

Every frame (40 fps by default) the engine:

1. reads the latest **audio** analysis and advances the **beat clock**,
2. resolves each layer's parameters: stored value, then **timeline
   keyframes** at the playhead, then **modulators**,
3. renders each **layer's** effect over its **group** of pixels,
4. composites the layers onto the **strips** (opacity + blend mode),
5. applies master/strip brightness, gamma and RGBW white extraction, then
   the per-strip **power limit**,
6. packs each controller's strips into consecutive DMX **universes** and
   sends them by unicast UDP,
7. sends a ~20 fps preview (pixels, beat, playhead, audio levels) to
   connected browsers over a WebSocket.

## Concepts

| Thing          | What it is |
|----------------|------------|
| **Controller** | A network pixel driver (one AirPixel). Has an IP, a protocol (sACN or Art-Net), a **start universe** and a **universe size**. An AirPixel Mini II takes 4 consecutive universes and outputs them as one continuous pixel stream. |
| **Strip**      | A run of pixels on a controller: pixel count, colour order (`GRB` for WS2812, `GRBW` for SK6812 RGBW, ...), brightness trim, and `reversed` if data enters at the far end. Several strips on one controller are chained in list order. |
| **Group**      | An ordered list of **sections** (strip + pixel range `from`–`to`, optionally reversed). Effects run through the sections in order as if they were one strip. That's how you make "the left half of strip 1 plus all of strip 3" behave as a single line. |
| **Layer**      | An **effect** applied to a group (or to *All pixels*), with params, opacity, blend mode (`normal`, `add`, `multiply`, `screen`, `max`) and mute. Layers are listed bottom to top. |
| **Tempo**      | The beat clock (BPM + phase), set by tap tempo, typed in, or locked to the audio. *Tempo-synced* layers receive beats instead of seconds. |
| **Timeline**   | One looping sequence, N beats long. **Tracks** hold **keyframes** (beat, value, easing) for one layer parameter each. |
| **Modulator**  | Maps a signal (`level`, `bass`, `mid`, `treble`, `beat` from audio; `pulse`, `phase` from the beat clock) onto a numeric layer parameter: `value = min + (max − min) × signal`. |

**Universe maths.** A universe holds 512 channels. RGB pixels use 3 channels
each, so by default LEDLEDLED sends **510 channels (170 pixels) per universe**
for RGB and **512 channels (128 pixels)** for RGBW. That way no pixel is split
across two universes. A 200-pixel RGB strip uses 2 universes (600 channels).
4 universes gives at most 680 RGB or 512 RGBW pixels per AirPixel, and the
Setup page warns you if you go over.

## Developing on your computer

Requirements: Go 1.24+ and Node 20+.

```sh
make frontend   # npm install + build the Svelte UI into internal/webui/dist
make run        # run on http://localhost:8080 using ./dev-config.json
make test       # Go unit tests (packet layouts, patching, effects, engine, API)
```

For UI work with hot reload, run `make run` in one terminal and `make dev`
in another, then open the Vite URL it prints (http://localhost:5173). API and
WebSocket calls are proxied to the Go server.

Running locally sends real sACN to whatever IPs are configured. That's harmless,
but you can untick **enabled** on controllers, or turn **Output OFF** in the header.

Command-line flags:

```
-config show.json   path to the show file (created with defaults if missing)
-listen :8080       HTTP listen address
-library DIR        saved shows/presets (default: <config dir>/library)
-debug              verbose logging
```

## Setting up the Raspberry Pi

Tested target: **Raspberry Pi 4, Raspberry Pi OS Bookworm 64-bit (Lite is fine)**,
plus a USB Wi-Fi stick that supports **AP mode**.

1. **Flash the SD card** with Raspberry Pi Imager. In its settings, set the
   hostname to `ledpi`, enable SSH, create your user, and set the Wi-Fi country.
   Connect the Pi to your router by **Ethernet** for setup, because the
   network script takes over both Wi-Fi radios.

2. **Install the app** from your computer:

   ```sh
   make install-pi PI_HOST=youruser@ledpi.local
   ```

   This cross-compiles for `linux/arm64`, copies the binary and systemd unit,
   and enables the `ledledled` service. The show file lives in
   `/var/lib/ledledled/show.json`.

3. **Set up the two access points** (USB stick plugged in):

   ```sh
   make network-pi PI_HOST=youruser@ledpi.local
   ssh youruser@ledpi.local
   sudo UI_PASS='choose-a-password' PIX_PASS='another-password' ./setup-network.sh
   ```

   Defaults (all overridable with environment variables, see the top of
   [deploy/setup-network.sh](deploy/setup-network.sh)):

   | Network          | SSID            | Password    | Pi address  | Radio / band            |
   |------------------|-----------------|-------------|-------------|-------------------------|
   | UI               | `LEDLEDLED`     | `ledledled` | `10.42.0.1` | onboard, 5 GHz ch 36    |
   | Pixels           | `LEDLEDLED-PIX` | `ledpixels` | `10.43.0.1` | USB stick, 2.4 GHz ch 6 |

   If a phone can't see the 5 GHz UI network, re-run with `UI_BAND=bg UI_CHANNEL=1`.
   `sudo ./setup-network.sh --remove` undoes it.

4. **Open the UI**: join `LEDLEDLED` on your phone and browse to
   **http://10.42.0.1** (or **http://led.lan**).

Updating later is just `make deploy PI_HOST=...`. Logs are available with
`journalctl -u ledledled -f`.

## Setting up the AirPixels

Summarised from the AirPixel Mini II manual. Do this once per unit.

1. Power the AirPixel. It appears as Wi-Fi network **`Airpixel xxyy`**
   (password **`smartshow`**). Connect a laptop or phone to it and browse to
   **http://192.168.4.1**.
2. Under **Device Settings & IP Configuration** set:
   - **Device IP**: `10.43.0.11` (then `.12`, `.13` for the others)
   - **Subnet mask**: `255.255.255.0`
   - press **Save Device Settings**.
3. Under **Network SSID & Password** enter `LEDLEDLED-PIX` and its password,
   press **Save SSID & Password**, then **Connect to Network**. The LED turns
   **solid blue** when it has joined.
4. Open the AirPixel's **browser configuration page**. To reach it, join a
   laptop to `LEDLEDLED-PIX` and browse to the device IP (e.g. http://10.43.0.11).
   This page **only opens while no data is streaming**, so press
   **Output ON → OFF** in the LEDLEDLED header first. Then set:
   - **Protocol**: sACN (E1.31). Use Art-Net if you changed it in LEDLEDLED.
   - **Starting universe**: must match the controller in LEDLEDLED's Setup
     tab. The defaults are **1, 5 and 9** for AirPixel 1, 2 and 3.
   - **Universe sizes**: **510** for each of the four universes (RGB strips),
     or 512 for RGBW. This must match the "Sending … × N ch" figure on the
     Setup page.
   - **Pixel type**: WS/SK for WS2812/SK6812.
5. Turn LEDLEDLED's **Output** back ON. The AirPixel's blue LED goes **off**
   while it receives matching data.

## First hardware test checklist

1. Setup tab: each controller should show packets increasing and **0 errors**.
2. Press **Identify** on a strip. You should see: **first pixel green, last
   pixel red, every 10th pixel white**, the rest blinking dim blue.
   - Wrong colours (e.g. first pixel red): change the strip's **colour order**
     (WS2812 is usually `GRB`).
   - Green at the far end: tick **reversed**.
   - Pattern stops or shifts colour **after pixel 170**: the universe-size
     setting on the AirPixel doesn't match LEDLEDLED. Make both 510, or set
     the controller's universe size to 512 in LEDLEDLED to match the AirPixel.
   - Nothing at all: check the blue LED (see Troubleshooting).
3. Set the pixel count to the real strip length and confirm the last pixel is red.
4. Live tab: try a few effects, then build a group in the Groups tab and add
   a colour layer for it.

## Tempo and the timeline

**Tempo.** The header shows the BPM, a **TAP** button and four beat dots.
Tap two or more times on the beat to set the tempo. A single tap re-syncs
the downbeat without changing the tempo. You can also type a BPM. The tempo
is saved with the show.

**Tempo sync.** Tick *Tempo sync ♩* on a layer and its effect runs on beats
instead of seconds. For example, a Pulse with *rate 1* pulses once per beat,
and a Chase with *speed 0.25* crosses the group once a bar. Everything
follows tap tempo and auto-BPM.

**Timeline.** One looping sequence per show, measured in beats (default 16
= 4 bars), so it also follows the tempo. Use ▶ / ❚❚ / ■ in the header or the
Timeline tab.

To animate a parameter:

1. Move the playhead: drag the ruler on the Timeline tab, or pause where you want.
2. On the Live tab, press **◇** next to the parameter. That stores a key with
   the current value at the playhead, snapped to the grid (Snap: ¼ beat by
   default).
3. Move the playhead and change the slider. Once a parameter has keys,
   editing it **records a key at the playhead**, as in After Effects. Its
   label turns amber and the slider shows the animated value.
4. On the Timeline tab, drag ◆ to retime keys, and tap one to edit its beat,
   value or **easing** (`linear`, `smooth`, `in`, `out`, `step`). Easing
   applies from that key to the next.

Colours, numbers and layer opacity can be keyed. If a layer is deleted or
its effect is changed, tracks for parameters that no longer exist are
removed automatically.

## Audio reactivity

**Hardware.** The Raspberry Pi 4 has **no audio input**; its 3.5 mm jack is
output only. Use a **USB sound card with line-in** (cheap and reliable), an
audio HAT with an ADC (e.g. HiFiBerry DAC+ADC), or a USB microphone. Feed it
a line level signal, e.g. from the mixing desk.

Audio is captured with ALSA's `arecord` (part of `alsa-utils`, normally
preinstalled on Pi OS; otherwise `sudo apt install alsa-utils`). The
systemd service runs in the `audio` group so it can open the device.

**Setup.**

1. Plug in the sound card and list devices: `arecord -l`.
2. Setup → *Audio & tempo*: tick **Audio input enabled** and pick the
   device. `plughw:CARD=...` / `plughw:1,0` style names are best because
   they convert the sample rate and channels for you. Press ↻ to rescan.
3. Play music. The meters and spectrum should move. If they barely move,
   raise the capture level with `alsamixer` (F6 to pick the card, F4 for
   capture) or adjust **Sensitivity**.

Signals auto-gain to 0..1 against their recent peaks, so you don't need to
calibrate them. A noise gate holds everything at 0 in silence.

**Using it.**

- **Modulators** (layer card → *Modulators*): e.g. `Opacity ← bass, 0 → 1`
  makes the layer pump with the kick. `Speed ← level, 0.1 → 2` speeds up
  with the music.
- **Effects**: *VU meter* (a bar following one signal, optionally mirrored
  from the centre) and *Spectrum* (16 bands spread along the group).
- **Master modulation** (Setup): master brightness follows a signal.
  `beat` or `pulse` at 80% flashes on the beat; `level` breathes with the music.
- **Auto BPM**: the tempo locks to detected beats. It glides towards the
  detected BPM, and the phase is nudged only by hits close to the predicted
  beat. Detection folds tempos into 80–160 BPM, so very fast or slow music
  may read at double or half speed. Tap to correct if needed.

**No hardware yet?** Choose the **`sim`** device. It synthesises a 120 BPM
kick, hats and a tone, so you can build audio-reactive looks on any machine.

## Power estimation and limiting

Each strip has a simple power model (Setup → strip → *Power*):

| Field | Meaning |
|---|---|
| Supply voltage | 5 V or 12 V (or 24 V). Only used to show watts. |
| mA per channel | Current of **one colour channel at full** on one *pixel* (addressable unit). |
| Idle mA per pixel | Quiescent draw of the driver chip with LEDs off. |
| Max amps | Limit for this strip. 0 = no limit. |

Presets fill in typical values:

| Strip | mA / channel | Idle | Full white per pixel |
|---|---|---|---|
| WS2812B 5 V | 20 | 1 | ~60 mA |
| SK6812 RGBW 5 V | 20 | 1 | ~80 mA (RGB+W) |
| WS2811 12 V (3 LEDs per pixel) | 20 | 2 | ~60 mA at 12 V |
| WS2815 12 V | 12 | 2 | ~36 mA at 12 V |

The estimate is computed from the **final bytes sent**, after gamma, master
and strip brightness, so it reflects what the LEDs actually draw. When it
exceeds *Max amps*, that strip's output is scaled down to fit and marked
**limited** (amber) on the Live and Setup tabs. The preview dims to match.

Set *Max amps* to the lowest of: PSU rating (with ~20% headroom), what your
power injection and cable can carry, and the strip's own copper (a 5 V strip
fed from one end usually manages only ~3–4 A before the far end goes orange).
For better accuracy, measure your strip at full white with a meter and
adjust *mA per channel*.

## Saving shows and presets

Everything you change is saved automatically to the running show
(`/var/lib/ledledled/show.json` on the Pi). On top of that, Setup → *Library*
keeps named copies as JSON files under `/var/lib/ledledled/library/`:

- **Shows** (`library/shows/<name>.json`): *Save current* stores the whole
  show, including the current BPM. **Load** makes it the running show on
  every connected device and stops the timeline at beat 0. ⤓ downloads the
  file; *Upload show .json* adds a file from your device to the library.
- **Layer presets** (`library/presets/<name>.json`): *Save as preset* on a
  layer stores its effect, settings, modulators and keyframe tracks. *Add from
  preset…* on the Live tab adds it to any show.

Names may contain letters, digits, spaces and `_ . ( ) -`. You can also copy
files in and out of the library folder directly over SSH.

## Troubleshooting

| Symptom | Things to check |
|---|---|
| AirPixel LED **solid blue** (connected, but not receiving matching data) | Protocol and start universe match on both sides? Output ON? Controller enabled? IP correct? |
| AirPixel LED flashing red | Not joined to `LEDLEDLED-PIX`: check SSID/password and that the pixel AP is up (`nmcli con show --active`). |
| Errors on a controller in Setup | Usually "network unreachable": the pixel AP is down or the IP isn't on 10.43.0.x. |
| UI says *Disconnected* | Phone left the `LEDLEDLED` network. It reconnects by itself when back. |
| Can't open the AirPixel config page | Turn **Output OFF** first. The AirPixel blocks its page while streaming. |
| See the packets | On the Pi: `sudo tcpdump -ni any udp port 5568` (sACN) or `6454` (Art-Net). |
| Audio: "arecord: ... No such file or directory" / device busy | Wrong device name (check `arecord -L`), card unplugged, or another program holds it. Capture retries every 3 s. |
| Audio meters flat | Raise capture gain in `alsamixer`, check the input is line-level and the right input is selected; try the `sim` device to rule out the software. |
| Strip shows "limited" all the time | Max amps lower than the look needs: raise it (if your supply allows) or lower brightness. |
| Choppy on Wi-Fi | Lower the FPS in Setup (25–30 is plenty), move the pixel AP channel (`PIX_CHANNEL=1` or `11`), keep the USB stick away from the Pi's metal case/USB3 ports. |

## Project layout

```
cmd/ledledled/        main: flags, wiring, graceful shutdown
internal/output/      sACN + Art-Net packet encoders and UDP transmitter
internal/pixel/       colours, colour orders, RGBW white extraction, gamma, power model
internal/show/        show data model, validation, universe patching, persistence
internal/effects/     effect registry, parameter schemas, built-in + audio effects, noise
internal/engine/      render loop: params (keys + mods), composite, encode, limit, send, preview
internal/tempo/       beat clock: BPM, tap tempo, phase nudging
internal/timeline/    keyframe track compilation and evaluation (easing)
internal/audio/       arecord capture / sim source, FFT analysis, beat + BPM detection
internal/library/     named show/preset JSON files
internal/fsutil/      atomic file writes
internal/server/      HTTP API, WebSocket, debounced saving, library endpoints
internal/webui/       go:embed of the built frontend
frontend/             Svelte 5 + Vite UI
deploy/               systemd unit, installer, two-AP network script
```

**Adding an effect.** Register it in
[internal/effects/builtin.go](internal/effects/builtin.go) with a name, label,
parameter schema and a render function. The UI builds its controls from the
schema automatically, with no frontend changes needed.

## HTTP API

| Method & path         | Body / result |
|-----------------------|---------------|
| `GET /api/show`       | the full show JSON |
| `PUT /api/show`       | replace the show. Validated, applied live, saved after 2 s idle. `422` with `{"error": ...}` if invalid |
| `GET /api/meta`       | effects with parameter schemas, colour orders, blend modes, protocols |
| `GET /api/status`     | fps, output/blackout state, per-controller packets/errors/warnings |
| `POST /api/output`    | `{"on": bool}` start/stop sending (stop = sACN stream-terminate) |
| `POST /api/blackout`  | `{"on": bool}` |
| `POST /api/identify`  | `{"stripId": "...", "seconds": 6}` |
| `POST /api/tempo/tap` | register a tap |
| `POST /api/tempo`     | `{"bpm": 128}` |
| `POST /api/transport` | `{"action": "play"\|"pause"\|"stop"\|"seek", "beat": 8}` |
| `GET /api/audio/devices` | ALSA capture devices (plus `sim`) |
| `GET /api/library/{shows\|presets}` | list entries |
| `GET\|PUT\|DELETE /api/library/{kind}/{name}` | read (`?download` for a file), store (validated) or delete an entry |
| `POST /api/library/shows/{name}/save` | save the running show under a name |
| `POST /api/library/shows/{name}/load` | load a saved show for everyone |
| `GET /ws`             | binary preview frames + JSON `live` (20 Hz: beat, playhead, audio), `status` (1 Hz) and `show` messages |

## A note on Wails

The plan started with [Wails](https://wails.io), but Wails v2 shows its UI in
a native window *on the machine running it*. It can't serve the UI to phones
on the network, and it would need a desktop environment on the Pi. So the core
is a plain Go HTTP server, built the way a Wails app is: Go backend plus an
embedded Svelte frontend. If you later want a desktop app (e.g. a laptop
"designer" version), a Wails shell can import `internal/engine` and
`internal/server` and point its webview at the same handler.

## TODO

Done since the first version: ~~keyframe timeline~~, ~~BPM / tap tempo~~,
~~power limiting per strip~~, ~~audio reactivity~~, ~~show library and
layer presets~~.

Near term (after the first hardware test):

- [ ] Confirm AirPixel universe-size behaviour (510 vs 512) and document the result
- [ ] Confirm sACN vs Art-Net behaviour/latency on the AirPixels
- [ ] Test audio capture with the actual USB sound card / HAT; tune the
      beat-detection threshold if needed
- [ ] Optionally send only on change (with a 1 s keep-alive) to save airtime
- [ ] Zoomable strip picker for precise selection on phones
- [ ] Captive portal so phones open the UI automatically on joining the network
- [ ] Simple PIN/password for the UI

Features:

- [ ] Multiple timelines / cues with a cue list and crossfades
- [ ] Keyframe copy/paste, multi-select, and "record" mode while playing
- [ ] Bezier easing curves; per-key ease editing in the lane
- [ ] Power limit per controller / PSU (shared across strips), not just per strip
- [ ] Palettes (multi-stop gradients) usable by any effect
- [ ] More effects: fire, meteor, strobe, twinkle palettes, waves, image scroll
- [ ] Global speed / pause; bar counter and phrase-aware auto-BPM
- [ ] Audio: modulate colours (hue shift by band), onset per band (kick/snare/hat)
- [ ] 2D layout: give strips positions so effects can be spatial
- [ ] Art-Net poll / discovery of controllers
- [ ] sACN multicast option and universe synchronisation
- [ ] Separate "rig" (controllers/strips) from "look" so shows can be swapped
      without touching hardware settings
- [ ] Undo/redo, and a patch-style API to avoid last-write-wins conflicts
- [ ] Optional Wails desktop shell reusing the engine
- [ ] Frontend tests and CI
