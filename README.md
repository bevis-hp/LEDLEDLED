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
  sparkle) on groups with opacity and blend modes.
- Live preview in the browser, multi-device sync, master dimmer, blackout,
  per-strip identify pattern.
- Effects are pure functions of `(time, params)`, so a keyframe timeline can
  be added later without reworking them (see TODO).

---

## Contents

1. [How it fits together](#how-it-fits-together)
2. [Concepts](#concepts)
3. [Developing on your computer](#developing-on-your-computer)
4. [Setting up the Raspberry Pi](#setting-up-the-raspberry-pi)
5. [Setting up the AirPixels](#setting-up-the-airpixels)
6. [First hardware test checklist](#first-hardware-test-checklist)
7. [Troubleshooting](#troubleshooting)
8. [Project layout](#project-layout)
9. [HTTP API](#http-api)
10. [A note on Wails](#a-note-on-wails)
11. [TODO](#todo)

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

1. renders each **layer's** effect over its **group** of pixels,
2. composites the layers onto the **strips** (opacity + blend mode),
3. applies master/strip brightness, gamma and RGBW white extraction,
4. packs each controller's strips into consecutive DMX **universes** and
   sends them by unicast UDP,
5. sends a ~20 fps preview to connected browsers over a WebSocket.

## Concepts

| Thing          | What it is |
|----------------|------------|
| **Controller** | A network pixel driver (one AirPixel). Has an IP, a protocol (sACN or Art-Net), a **start universe** and a **universe size**. An AirPixel Mini II takes 4 consecutive universes and outputs them as one continuous pixel stream. |
| **Strip**      | A run of pixels on a controller: pixel count, colour order (`GRB` for WS2812, `GRBW` for SK6812 RGBW, ...), brightness trim, and `reversed` if data enters at the far end. Several strips on one controller are chained in list order. |
| **Group**      | An ordered list of **sections** (strip + pixel range `from`–`to`, optionally reversed). Effects run through the sections in order as if they were one strip. That's how you make "the left half of strip 1 plus all of strip 3" behave as a single line. |
| **Layer**      | An **effect** applied to a group (or to *All pixels*), with params, opacity, blend mode (`normal`, `add`, `multiply`, `screen`, `max`) and mute. Layers are listed bottom to top. |

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

## Troubleshooting

| Symptom | Things to check |
|---|---|
| AirPixel LED **solid blue** (connected, but not receiving matching data) | Protocol and start universe match on both sides? Output ON? Controller enabled? IP correct? |
| AirPixel LED flashing red | Not joined to `LEDLEDLED-PIX`: check SSID/password and that the pixel AP is up (`nmcli con show --active`). |
| Errors on a controller in Setup | Usually "network unreachable": the pixel AP is down or the IP isn't on 10.43.0.x. |
| UI says *Disconnected* | Phone left the `LEDLEDLED` network. It reconnects by itself when back. |
| Can't open the AirPixel config page | Turn **Output OFF** first. The AirPixel blocks its page while streaming. |
| See the packets | On the Pi: `sudo tcpdump -ni any udp port 5568` (sACN) or `6454` (Art-Net). |
| Choppy on Wi-Fi | Lower the FPS in Setup (25–30 is plenty), move the pixel AP channel (`PIX_CHANNEL=1` or `11`), keep the USB stick away from the Pi's metal case/USB3 ports. |

## Project layout

```
cmd/ledledled/        main: flags, wiring, graceful shutdown
internal/output/      sACN + Art-Net packet encoders and UDP transmitter
internal/pixel/       colours, colour orders, RGBW white extraction, gamma
internal/show/        show data model, validation, universe patching, persistence
internal/effects/     effect registry, parameter schemas, built-in effects, noise
internal/engine/      render loop: compile plan, composite layers, encode, send, preview
internal/server/      HTTP API, WebSocket, debounced saving
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
| `GET /ws`             | binary preview frames + JSON `status`/`show` messages |

## A note on Wails

The plan started with [Wails](https://wails.io), but Wails v2 shows its UI in
a native window *on the machine running it*. It can't serve the UI to phones
on the network, and it would need a desktop environment on the Pi. So the core
is a plain Go HTTP server, built the way a Wails app is: Go backend plus an
embedded Svelte frontend. If you later want a desktop app (e.g. a laptop
"designer" version), a Wails shell can import `internal/engine` and
`internal/server` and point its webview at the same handler.

## TODO

Near term (after the first hardware test):

- [ ] Confirm AirPixel universe-size behaviour (510 vs 512) and document the result
- [ ] Confirm sACN vs Art-Net behaviour/latency on the AirPixels
- [ ] Optionally send only on change (with a 1 s keep-alive) to save airtime
- [ ] Zoomable strip picker for precise selection on phones
- [ ] Captive portal so phones open the UI automatically on joining the network
- [ ] Simple PIN/password for the UI

Features:

- [ ] **Keyframes / timeline**: per-layer parameter tracks (time → value with
      easing), evaluated before each frame. Effects already take all inputs via
      `effects.Params`, so a track system only needs to override values per frame
- [ ] Scenes/presets: save and recall layer stacks, crossfade between them
- [ ] Palettes (multi-stop gradients) usable by any effect
- [ ] More effects: fire, meteor, strobe, twinkle palettes, waves, image scroll
- [ ] BPM / tap tempo, global speed and pause
- [ ] 2D layout: give strips positions so effects can be spatial
- [ ] Power limiting per strip (estimated current)
- [ ] Art-Net poll / discovery of controllers
- [ ] sACN multicast option and universe synchronisation
- [ ] Undo/redo, and a patch-style API to avoid last-write-wins conflicts
- [ ] Optional Wails desktop shell reusing the engine
- [ ] Frontend tests and CI
