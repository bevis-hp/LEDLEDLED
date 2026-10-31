package server

import (
	"log/slog"
	"sync"
	"time"

	"ledledled/internal/show"
)

// saver writes the show to disk shortly after changes stop, so dragging a
// slider in the UI doesn't hammer the Pi's SD card with writes.
type saver struct {
	path  string
	delay time.Duration

	mu      sync.Mutex
	pending *show.Show
	timer   *time.Timer
}

func newSaver(path string, delay time.Duration) *saver {
	return &saver{path: path, delay: delay}
}

// Schedule queues s to be saved after the debounce delay.
func (sv *saver) Schedule(s *show.Show) {
	sv.mu.Lock()
	defer sv.mu.Unlock()
	sv.pending = s
	if sv.timer == nil {
		sv.timer = time.AfterFunc(sv.delay, sv.Flush)
	} else {
		sv.timer.Reset(sv.delay)
	}
}

// Flush writes any pending show immediately.
func (sv *saver) Flush() {
	sv.mu.Lock()
	s := sv.pending
	sv.pending = nil
	sv.mu.Unlock()
	if s == nil {
		return
	}
	if err := show.Save(sv.path, s); err != nil {
		slog.Error("save show", "path", sv.path, "err", err)
		return
	}
	slog.Info("show saved", "path", sv.path)
}
