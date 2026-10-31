// Package server exposes the engine over HTTP: a small JSON API, a
// WebSocket for live preview and multi-client sync, and the embedded UI.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

	"ledledled/internal/audio"
	"ledledled/internal/effects"
	"ledledled/internal/engine"
	"ledledled/internal/library"
	"ledledled/internal/output"
	"ledledled/internal/pixel"
	"ledledled/internal/show"
	"ledledled/internal/sysinfo"
	"ledledled/internal/timeline"
)

// Server wires HTTP handlers to the engine.
type Server struct {
	eng    *engine.Engine
	saver  *saver
	lib    *library.Store
	ui     fs.FS
	portal *portal
	sys    *sysinfo.Reader

	mu      sync.Mutex
	clients map[*client]struct{}
}

// client is one connected WebSocket.
type client struct {
	id   string
	text chan []byte // JSON messages (show updates)
}

// New creates a server. showPath is where changes are persisted; lib holds
// named shows and presets.
func New(eng *engine.Engine, showPath string, lib *library.Store, ui fs.FS) *Server {
	return &Server{
		eng:     eng,
		saver:   newSaver(showPath, 2*time.Second),
		lib:     lib,
		ui:      ui,
		clients: map[*client]struct{}{},
	}
}

// SetPortal enables captive-portal handling, redirecting unknown hosts and
// OS connectivity probes to url (e.g. "http://10.42.0.1/"). "" disables it.
func (s *Server) SetPortal(url string) { s.portal = newPortal(url) }

// SetSysInfo adds host health (temperature, uptime, Pi power flags) to
// status responses.
func (s *Server) SetSysInfo(r *sysinfo.Reader) { s.sys = r }

// statusPayload is the engine status plus host health.
type statusPayload struct {
	engine.Status
	System *sysinfo.Info `json:"system,omitempty"`
}

func (s *Server) status() statusPayload {
	p := statusPayload{Status: s.eng.Status()}
	if s.sys != nil {
		info := s.sys.Read()
		p.System = &info
	}
	return p
}

// Close flushes any unsaved changes.
func (s *Server) Close() { s.saver.Flush() }

// Handler returns the root HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/show", s.getShow)
	mux.HandleFunc("PUT /api/show", s.putShow)
	mux.HandleFunc("GET /api/meta", s.getMeta)
	mux.HandleFunc("GET /api/status", s.getStatus)
	mux.HandleFunc("POST /api/output", s.postToggle(s.eng.SetOutput))
	mux.HandleFunc("POST /api/blackout", s.postToggle(s.eng.SetBlackout))
	mux.HandleFunc("POST /api/identify", s.postIdentify)
	mux.HandleFunc("POST /api/tempo/tap", s.postTap)
	mux.HandleFunc("POST /api/tempo", s.postTempo)
	mux.HandleFunc("POST /api/transport", s.postTransport)
	mux.HandleFunc("GET /api/audio/devices", s.getAudioDevices)
	mux.HandleFunc("GET /api/library/{kind}", s.libList)
	mux.HandleFunc("GET /api/library/{kind}/{name}", s.libGet)
	mux.HandleFunc("PUT /api/library/{kind}/{name}", s.libPut)
	mux.HandleFunc("DELETE /api/library/{kind}/{name}", s.libDelete)
	mux.HandleFunc("POST /api/library/shows/{name}/save", s.libSaveCurrent)
	mux.HandleFunc("POST /api/library/shows/{name}/load", s.libLoad)
	mux.HandleFunc("GET /ws", s.ws)
	mux.Handle("/", s.static())
	return logRequests(s.portal.wrap(mux))
}

func (s *Server) getShow(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.eng.Show())
}

// putShow replaces the whole show. The UI always sends the complete show,
// which keeps the API tiny and makes "last write wins" easy to reason about.
func (s *Server) putShow(w http.ResponseWriter, r *http.Request) {
	var sh show.Show
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&sh); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.eng.Apply(&sh); err != nil {
		httpError(w, http.StatusUnprocessableEntity, err)
		return
	}
	applied := s.eng.Show()
	s.saver.Schedule(applied)
	s.broadcastShow(applied, r.Header.Get("X-Client-Id"))
	writeJSON(w, http.StatusOK, applied)
}

// meta is static information the UI needs to build its forms.
type meta struct {
	Effects     []*effects.Def    `json:"effects"`
	ColorOrders []string          `json:"colorOrders"`
	BlendModes  []string          `json:"blendModes"`
	Protocols   []output.Protocol `json:"protocols"`
	Eases       []string          `json:"eases"`
	ModSources  []string          `json:"modSources"`
	Palettes    []show.Palette    `json:"builtinPalettes"`
}

func (s *Server) getMeta(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, meta{
		Effects:     effects.All(),
		ColorOrders: pixel.SupportedOrders,
		BlendModes:  show.BlendModes,
		Protocols:   []output.Protocol{output.SACN, output.ArtNet},
		Eases:       timeline.Eases,
		ModSources:  show.ModSources,
		Palettes:    show.BuiltinPalettes,
	})
}

func (s *Server) getAudioDevices(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, audio.Devices())
}

func (s *Server) getStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.status())
}

func (s *Server) postToggle(set func(bool)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			On bool `json:"on"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpError(w, http.StatusBadRequest, err)
			return
		}
		set(body.On)
		writeJSON(w, http.StatusOK, s.status())
	}
}

func (s *Server) postIdentify(w http.ResponseWriter, r *http.Request) {
	var body struct {
		StripID string  `json:"stripId"`
		Seconds float64 `json:"seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	if body.Seconds <= 0 || body.Seconds > 60 {
		body.Seconds = 5
	}
	s.eng.Identify(body.StripID, time.Duration(body.Seconds*float64(time.Second)))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) postTap(w http.ResponseWriter, r *http.Request) {
	s.eng.Tap()
	s.saver.Schedule(s.eng.Show()) // persist the tapped BPM
	writeJSON(w, http.StatusOK, s.status())
}

func (s *Server) postTempo(w http.ResponseWriter, r *http.Request) {
	var body struct {
		BPM float64 `json:"bpm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	s.eng.SetBPM(body.BPM)
	s.saver.Schedule(s.eng.Show())
	writeJSON(w, http.StatusOK, s.status())
}

func (s *Server) postTransport(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Action string  `json:"action"` // play, pause, stop, seek
		Beat   float64 `json:"beat"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.eng.Transport(body.Action, body.Beat); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ws streams binary preview frames and JSON show updates to the browser.
// Status is pushed once a second so the UI can show fps/send errors.
func (s *Server) ws(w http.ResponseWriter, r *http.Request) {
	// Same-origin in production; localhost:* allows the Vite dev server.
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{"localhost:*"}})
	if err != nil {
		return
	}
	defer conn.CloseNow()
	ctx := conn.CloseRead(r.Context()) // we don't expect client messages

	c := &client{id: r.URL.Query().Get("client"), text: make(chan []byte, 8)}
	s.mu.Lock()
	s.clients[c] = struct{}{}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.clients, c)
		s.mu.Unlock()
	}()

	frames, cancel := s.eng.Subscribe()
	defer cancel()
	statusTick := time.NewTicker(time.Second)
	defer statusTick.Stop()

	write := func(typ websocket.MessageType, b []byte) error {
		wctx, done := context.WithTimeout(ctx, 2*time.Second)
		defer done()
		return conn.Write(wctx, typ, b)
	}
	for {
		var err error
		select {
		case <-ctx.Done():
			return
		case f := <-frames:
			typ := websocket.MessageBinary
			if f.Text {
				typ = websocket.MessageText
			}
			err = write(typ, f.Data)
		case msg := <-c.text:
			err = write(websocket.MessageText, msg)
		case <-statusTick.C:
			b, _ := json.Marshal(map[string]any{"type": "status", "status": s.status()})
			err = write(websocket.MessageText, b)
		}
		if err != nil {
			return
		}
	}
}

// broadcastShow tells other clients the show changed. The client that made
// the change is skipped so its in-progress edits (e.g. a slider drag)
// aren't overwritten by its own echo.
func (s *Server) broadcastShow(sh *show.Show, originID string) {
	b, _ := json.Marshal(map[string]any{"type": "show", "show": sh})
	s.mu.Lock()
	defer s.mu.Unlock()
	for c := range s.clients {
		if originID != "" && c.id == originID {
			continue
		}
		select {
		case c.text <- b:
		default:
		}
	}
}

// static serves the embedded UI, falling back to index.html for unknown
// paths so client-side routes work.
func (s *Server) static() http.Handler {
	files := http.FileServerFS(s.ui)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(s.ui, p); err != nil {
			if _, err := fs.Stat(s.ui, "index.html"); err != nil {
				http.Error(w, "frontend not built: run `make frontend` and rebuild", http.StatusNotFound)
				return
			}
			r.URL.Path = "/"
		}
		files.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, code int, err error) {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		code = http.StatusRequestEntityTooLarge
	}
	writeJSON(w, code, map[string]string{"error": err.Error()})
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			slog.Debug("http", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr)
		}
		next.ServeHTTP(w, r)
	})
}
