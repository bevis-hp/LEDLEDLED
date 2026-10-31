// Package engine runs the render loop: every frame it evaluates each layer's
// effect, composites the layers onto the strips, encodes the pixels into
// DMX universes and sends them to the controllers. It also publishes
// low-rate preview frames for the web UI.
package engine

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"log/slog"
	"math"
	"sync"
	"time"

	"ledledled/internal/audio"
	"ledledled/internal/effects"
	"ledledled/internal/output"
	"ledledled/internal/pixel"
	"ledledled/internal/show"
	"ledledled/internal/tempo"
	"ledledled/internal/timeline"
)

// previewHz is the target rate of preview frames sent to the UI.
const previewHz = 20

// Engine owns the active show and the output socket.
type Engine struct {
	mu      sync.Mutex
	plan    *plan
	tx      *output.Transmitter
	txOpts  output.Options
	start   time.Time
	version uint64
	clock   *tempo.Clock
	tr      transport
	audio   *audio.Manager
	feat    audio.Features // latest audio analysis, refreshed every frame
	beats   uint64         // last seen audio beat count
	tlPos   float64        // timeline playhead at the last frame

	outputOn bool
	blackout bool
	identify map[string]time.Time // strip ID -> until

	stats  map[string]*ctrlStats // by controller ID
	fps    float64
	frames int
	fpsAt  time.Time

	subMu sync.Mutex
	subs  map[chan Message]struct{}

	reset chan struct{} // nudges Run to pick up a new FPS
}

type ctrlStats struct {
	packets uint64
	errors  uint64
	lastErr string
}

// New creates an engine for s, which must already have passed Prepare.
func New(s *show.Show) (*Engine, error) {
	e := &Engine{
		start:    time.Now(),
		fpsAt:    time.Now(),
		outputOn: true,
		identify: map[string]time.Time{},
		stats:    map[string]*ctrlStats{},
		subs:     map[chan Message]struct{}{},
		reset:    make(chan struct{}, 1),
		clock:    tempo.New(120),
		audio:    audio.NewManager(),
	}
	if err := e.Load(s); err != nil {
		return nil, err
	}
	return e, nil
}

// Load replaces the show like Apply, and also restores its saved tempo.
// Use it for startup and library loads; Apply is for live edits, which
// must not reset a tempo that was just tapped in.
func (e *Engine) Load(s *show.Show) error {
	if err := e.Apply(s); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.clock.SetBPM(e.now(), e.plan.show.Tempo.BPM)
	return nil
}

// now is the engine clock in seconds.
func (e *Engine) now() float64 { return time.Since(e.start).Seconds() }

// Apply swaps in a new show. On error the running show is unchanged.
func (e *Engine) Apply(s *show.Show) error {
	s = s.Clone()
	if err := Prepare(s); err != nil {
		return err
	}
	p, err := compile(s)
	if err != nil {
		return err
	}

	opts := output.Options{
		BindIP:     s.Settings.BindIP,
		SourceName: s.Settings.SourceName,
		Priority:   byte(s.Settings.Priority),
	}
	opts.CID, _ = show.ParseCID(s.Settings.CID)

	if err := e.swap(p, opts); err != nil {
		return err
	}
	a := s.Settings.Audio
	e.audio.Configure(audio.Config{Enabled: a.Enabled, Device: a.Device, Gain: a.Gain})
	return nil
}

// swap installs a compiled plan (and a new socket if output options changed).
func (e *Engine) swap(p *plan, opts output.Options) error {
	s := p.show
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.tx == nil || opts != e.txOpts {
		tx, err := output.NewTransmitter(opts)
		if err != nil {
			return err
		}
		if e.tx != nil {
			e.tx.Close()
		}
		e.tx, e.txOpts = tx, opts
	}
	oldFPS := 0
	if e.plan != nil {
		oldFPS = e.plan.show.Settings.FPS
	}
	e.plan = p
	e.version++
	if oldFPS != s.Settings.FPS {
		select {
		case e.reset <- struct{}{}:
		default:
		}
	}
	return nil
}

// Show returns a copy of the active show, including the live tempo.
func (e *Engine) Show() *show.Show {
	e.mu.Lock()
	defer e.mu.Unlock()
	s := e.plan.show.Clone()
	s.Tempo.BPM = e.clock.BPM()
	return s
}

// Tap registers a tap-tempo tap now.
func (e *Engine) Tap() {
	e.mu.Lock()
	e.clock.Tap(e.now())
	e.mu.Unlock()
}

// SetBPM sets the tempo directly.
func (e *Engine) SetBPM(bpm float64) {
	e.mu.Lock()
	e.clock.SetBPM(e.now(), bpm)
	e.mu.Unlock()
}

// SetOutput enables or disables network output. Disabling sends sACN
// stream-terminated packets so receivers release immediately, which also
// lets you open the AirPixel's own config page (it is blocked while
// receiving data).
func (e *Engine) SetOutput(on bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.outputOn && !on {
		e.terminateLocked()
	}
	e.outputOn = on
}

// SetBlackout forces all pixels to black while keeping the stream alive.
func (e *Engine) SetBlackout(on bool) {
	e.mu.Lock()
	e.blackout = on
	e.mu.Unlock()
}

// Identify shows a test pattern on a strip for d.
func (e *Engine) Identify(stripID string, d time.Duration) {
	e.mu.Lock()
	e.identify[stripID] = time.Now().Add(d)
	e.mu.Unlock()
}

// Run drives the render loop until ctx is cancelled.
func (e *Engine) Run(ctx context.Context) {
	for {
		e.mu.Lock()
		fps := e.plan.show.Settings.FPS
		e.mu.Unlock()

		ticker := time.NewTicker(time.Second / time.Duration(fps))
		previewEvery := max(1, fps/previewHz)
		n := 0
	loop:
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				e.mu.Lock()
				if e.outputOn {
					e.terminateLocked()
				}
				e.tx.Close()
				e.mu.Unlock()
				e.audio.Stop()
				return
			case <-e.reset:
				ticker.Stop()
				break loop
			case now := <-ticker.C:
				n++
				e.frame(now, n%previewEvery == 0)
			}
		}
	}
}

// frame renders and sends one frame.
func (e *Engine) frame(now time.Time, preview bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	p := e.plan
	t := now.Sub(e.start).Seconds()

	e.feat = e.audio.Latest()
	e.followAudioTempo(p, t)
	beat := e.clock.Beat(t)
	e.resolveParams(p, beat)
	e.render(p, t, beat, now)

	e.encode(p)
	if e.outputOn {
		e.send(p)
	}
	if preview {
		e.publish(Message{Data: e.encodePreview(p)})
		e.publish(Message{Text: true, Data: e.encodeLive(p, beat)})
	}

	e.frames++
	if el := now.Sub(e.fpsAt); el >= time.Second {
		e.fps = float64(e.frames) / el.Seconds()
		e.frames, e.fpsAt = 0, now
		rollPowerWindow(p)
	}
}

// followAudioTempo locks the beat clock to detected audio beats when
// auto-BPM is on: the tempo glides towards the detected BPM and each
// detected beat close to a predicted beat nudges the phase.
func (e *Engine) followAudioTempo(p *plan, t float64) {
	f := &e.feat
	newBeat := f.BeatCount != e.beats
	e.beats = f.BeatCount
	if !p.show.Settings.Audio.AutoBPM || f.BPM <= 0 {
		return
	}
	if cur := e.clock.BPM(); math.Abs(cur-f.BPM) > 0.05 {
		e.clock.SetBPM(t, cur+(f.BPM-cur)*0.05)
	}
	if newBeat {
		at := t - f.BeatAge
		b := e.clock.Beat(at)
		if math.Abs(b-math.Round(b)) < 0.25 { // ignore off-beat hits
			e.clock.Nudge(at, 0.2)
		}
	}
}

// modSource returns a modulation source's current value (0..1).
func (e *Engine) modSource(name string, beat float64) float64 {
	switch name {
	case "pulse":
		return math.Exp(-(beat - math.Floor(beat)) * 6)
	case "phase":
		return beat - math.Floor(beat)
	}
	return e.feat.Get(name)
}

// resolveParams sets this frame's parameter overrides: timeline keyframes
// at the current playhead, then modulators on top, then master modulation.
func (e *Engine) resolveParams(p *plan, beat float64) {
	for i := range p.layers {
		p.layers[i].params.ClearOverrides()
		p.layers[i].opNow = p.layers[i].opacity
	}
	tl := &p.show.Timeline
	e.tlPos = e.tr.position(beat, tl.LengthBeats, tl.OneShot)
	for i := range p.tracks {
		tp := &p.tracks[i]
		num, col, ok := tp.track.At(e.tlPos)
		if !ok {
			continue
		}
		l := &p.layers[tp.layer]
		switch {
		case tp.opacity:
			l.opNow = float32(math.Max(0, math.Min(1, num)))
		case tp.track.Kind == timeline.Color:
			l.params.OverrideColor(tp.param, col)
		default:
			l.params.OverrideNum(tp.param, num)
		}
	}
	for i := range p.layers {
		l := &p.layers[i]
		for _, m := range l.mods {
			v := m.Min + (m.Max-m.Min)*e.modSource(m.Source, beat)
			if m.Param == "opacity" {
				l.opNow = float32(math.Max(0, math.Min(1, v)))
			} else {
				l.params.OverrideNum(m.Param, v)
			}
		}
	}
	p.masterK = 1
	if mm := p.show.Settings.MasterMod; mm.Source != "" {
		p.masterK = float32(1 - mm.Amount*(1-e.modSource(mm.Source, beat)))
	}
}

// render evaluates and composites all layers into the strip buffers.
func (e *Engine) render(p *plan, t, beat float64, now time.Time) {
	for i := range p.strips {
		clear(p.strips[i].px)
	}
	if !e.blackout {
		for i := range p.layers {
			l := &p.layers[i]
			ctx := effects.Context{Time: t, Beat: beat, Audio: &e.feat, Params: l.params}
			if l.tempoSync {
				ctx.Time = beat
			}
			l.def.Render(&ctx, l.buf)
			for j, ref := range l.targets {
				dst := &p.strips[ref.strip].px[ref.idx]
				*dst = l.blend(*dst, l.buf[j], l.opNow)
			}
		}
	}
	for id, until := range e.identify {
		if now.After(until) {
			delete(e.identify, id)
			continue
		}
		for si, st := range p.show.Strips {
			if st.ID == id {
				identifyPattern(p.strips[si].px, now)
			}
		}
	}
}

// identifyPattern helps check mapping: first pixel green, last pixel red,
// every 10th pixel white, everything else a dim blinking blue.
func identifyPattern(px []pixel.Color, now time.Time) {
	blink := float32(0.15)
	if now.UnixMilli()/250%2 == 0 {
		blink = 0.4
	}
	for i := range px {
		switch {
		case i == 0:
			px[i] = pixel.Color{G: 1}
		case i == len(px)-1:
			px[i] = pixel.Color{R: 1}
		case i%10 == 0:
			px[i] = pixel.Color{R: 1, G: 1, B: 1}
		default:
			px[i] = pixel.Color{B: blink}
		}
	}
}

// encode converts every strip into its controller's channel stream,
// applying brightness, gamma, white extraction and the power limit.
func (e *Engine) encode(p *plan) {
	master := float32(p.show.Settings.Brightness) * p.masterK
	for ci := range p.controllers {
		c := &p.controllers[ci]
		for _, sp := range c.strips {
			sb := &p.strips[sp.StripIndex]
			src := sb.px
			if sb.reverse {
				for i, v := range sb.px {
					sb.scratch[len(sb.px)-1-i] = v
				}
				src = sb.scratch
			}
			n := len(src) * sp.Order.Channels()
			out := c.stream[sp.ChannelOffset : sp.ChannelOffset+n]
			sp.Order.Encode(out, src, pixel.EncodeOptions{
				Brightness:   master * sb.brightness,
				Gamma:        p.gamma,
				ExtractWhite: sb.white,
			})
			req, got, scale := sb.power.Limit(out, len(src), sb.maxMA)
			sb.powered = true
			sb.ma, sb.scale = got, scale
			sb.winMA = max(sb.winMA, got)
			sb.winReqMA = max(sb.winReqMA, req)
			sb.winLimited = sb.winLimited || scale < 1
		}
	}
}

// rollPowerWindow publishes the last second's power peaks.
func rollPowerWindow(p *plan) {
	for i := range p.strips {
		s := &p.strips[i]
		s.peakMA, s.peakReqMA, s.limited = s.winMA, s.winReqMA, s.winLimited
		s.winMA, s.winReqMA, s.winLimited = 0, 0, false
	}
}

// send transmits each enabled controller's universes.
func (e *Engine) send(p *plan) {
	for ci := range p.controllers {
		c := &p.controllers[ci]
		if c.disabled || c.universes == 0 {
			continue
		}
		st := e.statsFor(c.id)
		for u := range c.universes {
			err := e.tx.Send(c.proto, c.dst, uint16(c.startUniverse+u), c.universeData(u))
			if err != nil {
				st.errors++
				if st.lastErr != err.Error() {
					slog.Warn("send failed", "controller", c.id, "err", err)
				}
				st.lastErr = err.Error()
				continue
			}
			st.packets++
			st.lastErr = ""
		}
	}
}

// terminateLocked sends sACN stream-terminated packets to every universe.
func (e *Engine) terminateLocked() {
	for _, c := range e.plan.controllers {
		for u := range c.universes {
			_ = e.tx.Terminate(c.proto, c.dst, uint16(c.startUniverse+u), c.universeData(u))
		}
	}
}

func (e *Engine) statsFor(id string) *ctrlStats {
	st, ok := e.stats[id]
	if !ok {
		st = &ctrlStats{}
		e.stats[id] = st
	}
	return st
}

// encodePreview packs strip colours for the UI:
//
//	u8 type (1) | u16 strip count | u16 pixels per strip... | RGB bytes...
//
// Colours include master and strip brightness but not gamma, since the
// browser display applies its own.
func (e *Engine) encodePreview(p *plan) []byte {
	total := 0
	for _, s := range p.strips {
		total += len(s.px)
	}
	buf := make([]byte, 3+2*len(p.strips)+3*total)
	buf[0] = 1
	binary.LittleEndian.PutUint16(buf[1:], uint16(len(p.strips)))
	off := 3
	for _, s := range p.strips {
		binary.LittleEndian.PutUint16(buf[off:], uint16(len(s.px)))
		off += 2
	}
	master := float32(p.show.Settings.Brightness) * p.masterK
	for _, s := range p.strips {
		k := master * s.brightness
		if s.scale < 1 {
			// Bytes were scaled after gamma; undo gamma to approximate it.
			k *= float32(math.Pow(s.scale, 1/p.show.Settings.Gamma))
		}
		for _, c := range s.px {
			c = c.Scale(k).Clamp()
			buf[off], buf[off+1], buf[off+2] = byte(c.R*255+0.5), byte(c.G*255+0.5), byte(c.B*255+0.5)
			off += 3
		}
	}
	return buf
}

// Message is a frame for UI subscribers: binary preview pixels, or a JSON
// "live" text message (beat, playhead, audio levels).
type Message struct {
	Text bool
	Data []byte
}

// Subscribe returns a channel of preview messages. Slow consumers drop
// messages rather than stalling the render loop. Call cancel when done.
func (e *Engine) Subscribe() (<-chan Message, func()) {
	ch := make(chan Message, 8)
	e.subMu.Lock()
	e.subs[ch] = struct{}{}
	e.subMu.Unlock()
	return ch, func() {
		e.subMu.Lock()
		delete(e.subs, ch)
		e.subMu.Unlock()
	}
}

func (e *Engine) publish(frame Message) {
	e.subMu.Lock()
	defer e.subMu.Unlock()
	for ch := range e.subs {
		select {
		case ch <- frame:
		default:
		}
	}
}

// liveMsg is the ~20 Hz JSON update for animated UI elements.
type liveMsg struct {
	Type     string          `json:"type"`
	BPM      float64         `json:"bpm"`
	Beat     float64         `json:"beat"`
	Timeline liveTLInfo      `json:"timeline"`
	Audio    *audio.Features `json:"audio,omitempty"` // only while audio is enabled
}

type liveTLInfo struct {
	Playing bool    `json:"playing"`
	Pos     float64 `json:"pos"`
	Length  float64 `json:"length"`
}

func (e *Engine) encodeLive(p *plan, beat float64) []byte {
	m := liveMsg{
		Type: "live", BPM: e.clock.BPM(), Beat: beat,
		Timeline: liveTLInfo{Playing: e.tr.playing, Pos: e.tlPos, Length: p.show.Timeline.LengthBeats},
	}
	if p.show.Settings.Audio.Enabled {
		m.Audio = &e.feat
	}
	b, _ := json.Marshal(m)
	return b
}

// Status is a snapshot of runtime state for the UI.
type Status struct {
	Version     uint64             `json:"version"`
	OutputOn    bool               `json:"outputOn"`
	Blackout    bool               `json:"blackout"`
	FPS         float64            `json:"fps"`
	BPM         float64            `json:"bpm"`
	Beat        float64            `json:"beat"`
	Controllers []ControllerStatus `json:"controllers"`
	Strips      []StripStatus      `json:"strips"`
	Audio       audio.Status       `json:"audio"`
}

// StripStatus reports estimated power over the last second.
type StripStatus struct {
	ID            string  `json:"id"`
	Amps          float64 `json:"amps"`          // peak draw after limiting
	RequestedAmps float64 `json:"requestedAmps"` // peak draw the effects asked for
	Watts         float64 `json:"watts"`
	Limited       bool    `json:"limited"`
}

// ControllerStatus reports per-controller patch and send health.
type ControllerStatus struct {
	ID           string   `json:"id"`
	Universes    int      `json:"universes"`
	UniverseSize int      `json:"universeSize"`
	Channels     int      `json:"channels"`
	Packets      uint64   `json:"packets"`
	Errors       uint64   `json:"errors"`
	LastError    string   `json:"lastError,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
}

// Status returns current runtime state.
func (e *Engine) Status() Status {
	e.mu.Lock()
	defer e.mu.Unlock()
	s := Status{Version: e.version, OutputOn: e.outputOn, Blackout: e.blackout, FPS: e.fps,
		BPM: e.clock.BPM(), Beat: e.clock.Beat(e.now()), Audio: e.audio.Status()}
	for _, c := range e.plan.controllers {
		st := e.statsFor(c.id)
		s.Controllers = append(s.Controllers, ControllerStatus{
			ID: c.id, Universes: c.universes, UniverseSize: c.universeSize, Channels: c.channels,
			Packets: st.packets, Errors: st.errors, LastError: st.lastErr, Warnings: c.warnings,
		})
	}
	for _, sp := range e.plan.strips {
		if !sp.powered {
			continue
		}
		s.Strips = append(s.Strips, StripStatus{
			ID: sp.id, Amps: sp.peakMA / 1000, RequestedAmps: sp.peakReqMA / 1000,
			Watts: sp.peakMA / 1000 * sp.power.Volts, Limited: sp.limited,
		})
	}
	return s
}
