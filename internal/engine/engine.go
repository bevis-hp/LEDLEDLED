// Package engine runs the render loop: every frame it evaluates each layer's
// effect, composites the layers onto the strips, encodes the pixels into
// DMX universes and sends them to the controllers. It also publishes
// low-rate preview frames for the web UI.
package engine

import (
	"context"
	"encoding/binary"
	"log/slog"
	"sync"
	"time"

	"ledledled/internal/effects"
	"ledledled/internal/output"
	"ledledled/internal/pixel"
	"ledledled/internal/show"
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

	outputOn bool
	blackout bool
	identify map[string]time.Time // strip ID -> until

	stats  map[string]*ctrlStats // by controller ID
	fps    float64
	frames int
	fpsAt  time.Time

	subMu sync.Mutex
	subs  map[chan []byte]struct{}

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
		outputOn: true,
		identify: map[string]time.Time{},
		stats:    map[string]*ctrlStats{},
		subs:     map[chan []byte]struct{}{},
		reset:    make(chan struct{}, 1),
	}
	if err := e.Apply(s); err != nil {
		return nil, err
	}
	return e, nil
}

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

// Show returns a copy of the active show.
func (e *Engine) Show() *show.Show {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.plan.show.Clone()
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

	e.render(p, t, now)

	if e.outputOn {
		e.send(p)
	}
	if preview {
		e.publish(e.encodePreview(p))
	}

	e.frames++
	if el := now.Sub(e.fpsAt); el >= time.Second {
		e.fps = float64(e.frames) / el.Seconds()
		e.frames, e.fpsAt = 0, now
	}
}

// render evaluates and composites all layers into the strip buffers.
func (e *Engine) render(p *plan, t float64, now time.Time) {
	for i := range p.strips {
		clear(p.strips[i].px)
	}
	if !e.blackout {
		for i := range p.layers {
			l := &p.layers[i]
			ctx := effects.Context{Time: t, Params: l.params}
			l.def.Render(&ctx, l.buf)
			for j, ref := range l.targets {
				dst := &p.strips[ref.strip].px[ref.idx]
				*dst = l.blend(*dst, l.buf[j], l.opacity)
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

// send encodes strips into each controller's stream and transmits it.
func (e *Engine) send(p *plan) {
	master := float32(p.show.Settings.Brightness)
	for ci := range p.controllers {
		c := &p.controllers[ci]
		if c.disabled || c.universes == 0 {
			continue
		}
		for _, sp := range c.strips {
			sb := &p.strips[sp.StripIndex]
			src := sb.px
			if sb.reverse {
				for i, v := range sb.px {
					sb.scratch[len(sb.px)-1-i] = v
				}
				src = sb.scratch
			}
			sp.Order.Encode(c.stream[sp.ChannelOffset:], src, pixel.EncodeOptions{
				Brightness:   master * sb.brightness,
				Gamma:        p.gamma,
				ExtractWhite: sb.white,
			})
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
	master := float32(p.show.Settings.Brightness)
	for _, s := range p.strips {
		k := master * s.brightness
		for _, c := range s.px {
			c = c.Scale(k).Clamp()
			buf[off], buf[off+1], buf[off+2] = byte(c.R*255+0.5), byte(c.G*255+0.5), byte(c.B*255+0.5)
			off += 3
		}
	}
	return buf
}

// Subscribe returns a channel of preview frames. Slow consumers drop frames
// rather than stalling the render loop. Call cancel when done.
func (e *Engine) Subscribe() (<-chan []byte, func()) {
	ch := make(chan []byte, 2)
	e.subMu.Lock()
	e.subs[ch] = struct{}{}
	e.subMu.Unlock()
	return ch, func() {
		e.subMu.Lock()
		delete(e.subs, ch)
		e.subMu.Unlock()
	}
}

func (e *Engine) publish(frame []byte) {
	e.subMu.Lock()
	defer e.subMu.Unlock()
	for ch := range e.subs {
		select {
		case ch <- frame:
		default:
		}
	}
}

// Status is a snapshot of runtime state for the UI.
type Status struct {
	Version     uint64             `json:"version"`
	OutputOn    bool               `json:"outputOn"`
	Blackout    bool               `json:"blackout"`
	FPS         float64            `json:"fps"`
	Controllers []ControllerStatus `json:"controllers"`
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
	s := Status{Version: e.version, OutputOn: e.outputOn, Blackout: e.blackout, FPS: e.fps}
	for _, c := range e.plan.controllers {
		st := e.statsFor(c.id)
		s.Controllers = append(s.Controllers, ControllerStatus{
			ID: c.id, Universes: c.universes, UniverseSize: c.universeSize, Channels: c.channels,
			Packets: st.packets, Errors: st.errors, LastError: st.lastErr, Warnings: c.warnings,
		})
	}
	return s
}
