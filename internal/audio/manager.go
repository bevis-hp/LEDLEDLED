package audio

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Config selects and tunes the audio input.
type Config struct {
	Enabled bool
	Device  string  // ALSA device (e.g. "default", "plughw:1,0") or "sim"
	Gain    float64 // sensitivity multiplier after auto-gain, default 1
}

// Status reports the capture state for the UI.
type Status struct {
	Running bool   `json:"running"`
	Device  string `json:"device"`
	Error   string `json:"error,omitempty"`
}

// Manager runs capture + analysis in the background and keeps the latest
// features. It restarts capture if the device fails (e.g. a USB sound card
// is unplugged and replugged).
type Manager struct {
	mu      sync.Mutex
	cfg     Config
	cancel  context.CancelFunc
	done    chan struct{}
	feat    Features
	beatAt  time.Time
	status  Status
	gainSet chan float64
}

// NewManager returns an idle manager.
func NewManager() *Manager { return &Manager{} }

// Configure applies cfg, (re)starting or stopping capture as needed.
func (m *Manager) Configure(cfg Config) {
	if cfg.Gain <= 0 {
		cfg.Gain = 1
	}
	m.mu.Lock()
	restart := cfg.Enabled != m.cfg.Enabled || cfg.Device != m.cfg.Device
	gainOnly := !restart && cfg.Gain != m.cfg.Gain
	m.cfg = cfg
	gainCh := m.gainSet
	m.mu.Unlock()

	if gainOnly && gainCh != nil {
		select {
		case gainCh <- cfg.Gain:
		default:
		}
	}
	if !restart {
		return
	}
	m.Stop()
	if !cfg.Enabled {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	gainCh = make(chan float64, 1)
	m.mu.Lock()
	m.cancel, m.done, m.gainSet = cancel, done, gainCh
	m.status = Status{Device: cfg.Device}
	m.mu.Unlock()
	go m.run(ctx, cfg, gainCh, done)
}

// Stop ends capture and waits for it to finish.
func (m *Manager) Stop() {
	m.mu.Lock()
	cancel, done := m.cancel, m.done
	m.cancel, m.done, m.gainSet = nil, nil, nil
	m.feat = Features{}
	m.status = Status{Device: m.cfg.Device}
	m.mu.Unlock()
	if cancel != nil {
		cancel()
		<-done
	}
}

// Latest returns the most recent features. BeatAge is filled relative to now.
func (m *Manager) Latest() Features {
	m.mu.Lock()
	defer m.mu.Unlock()
	f := m.feat
	if !m.beatAt.IsZero() {
		f.BeatAge = time.Since(m.beatAt).Seconds()
	}
	return f
}

// Status returns the capture state.
func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status
}

func (m *Manager) run(ctx context.Context, cfg Config, gainCh <-chan float64, done chan struct{}) {
	defer close(done)
	a := newAnalyzer()
	a.gain = cfg.Gain
	onBlock := func(samples []float64, at time.Time) {
		select {
		case g := <-gainCh:
			a.gain = g
		default:
		}
		prevBeats := a.out.BeatCount
		f := a.process(samples, at)
		m.mu.Lock()
		m.feat = f
		if f.BeatCount != prevBeats {
			m.beatAt = at
		}
		m.status.Running, m.status.Error = true, ""
		m.mu.Unlock()
	}

	for {
		var err error
		if cfg.Device == SimDevice {
			err = captureSim(ctx, onBlock)
		} else {
			err = captureArecord(ctx, cfg.Device, onBlock)
		}
		if ctx.Err() != nil {
			return
		}
		msg := "capture stopped"
		if err != nil {
			msg = err.Error()
		}
		slog.Warn("audio capture failed, retrying", "device", cfg.Device, "err", msg)
		m.mu.Lock()
		m.status.Running, m.status.Error = false, msg
		m.feat = Features{}
		m.mu.Unlock()
		select {
		case <-ctx.Done():
			return
		case <-time.After(3 * time.Second):
		}
	}
}
