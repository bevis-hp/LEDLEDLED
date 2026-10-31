package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"ledledled/internal/effects"
	"ledledled/internal/engine"
	"ledledled/internal/library"
	"ledledled/internal/show"
)

// Library endpoints. {kind} is "shows" or "presets".

func (s *Server) libList(w http.ResponseWriter, r *http.Request) {
	list, err := s.lib.List(library.Kind(r.PathValue("kind")))
	if err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) libGet(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	b, err := s.lib.Read(library.Kind(r.PathValue("kind")), name)
	if err != nil {
		libError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if r.URL.Query().Has("download") {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name+".json"))
	}
	_, _ = w.Write(b)
}

// libPut stores an uploaded show or preset after validating it.
func (s *Server) libPut(w http.ResponseWriter, r *http.Request) {
	kind := library.Kind(r.PathValue("kind"))
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4<<20))
	if err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	var clean any
	switch kind {
	case library.Shows:
		var sh show.Show
		if err := json.Unmarshal(body, &sh); err != nil {
			httpError(w, http.StatusBadRequest, err)
			return
		}
		if err := engine.Prepare(&sh); err != nil {
			httpError(w, http.StatusUnprocessableEntity, err)
			return
		}
		clean = &sh
	case library.Presets:
		var p library.Preset
		if err := json.Unmarshal(body, &p); err != nil {
			httpError(w, http.StatusBadRequest, err)
			return
		}
		if _, ok := effects.Get(p.Layer.Effect); !ok {
			httpError(w, http.StatusUnprocessableEntity, fmt.Errorf("unknown effect %q", p.Layer.Effect))
			return
		}
		if p.Tracks == nil {
			p.Tracks = []show.Track{}
		}
		clean = &p
	default:
		httpError(w, http.StatusBadRequest, fmt.Errorf("unknown library section %q", kind))
		return
	}
	s.writeLib(w, kind, r.PathValue("name"), clean)
}

// libSaveCurrent saves the running show (including live BPM) by name.
func (s *Server) libSaveCurrent(w http.ResponseWriter, r *http.Request) {
	s.writeLib(w, library.Shows, r.PathValue("name"), s.eng.Show())
}

func (s *Server) writeLib(w http.ResponseWriter, kind library.Kind, name string, v any) {
	b, _ := json.MarshalIndent(v, "", "  ")
	if err := s.lib.Write(kind, name, b); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// libLoad makes a stored show the running show, for every client.
func (s *Server) libLoad(w http.ResponseWriter, r *http.Request) {
	b, err := s.lib.Read(library.Shows, r.PathValue("name"))
	if err != nil {
		libError(w, err)
		return
	}
	var sh show.Show
	if err := json.Unmarshal(b, &sh); err != nil {
		httpError(w, http.StatusUnprocessableEntity, err)
		return
	}
	if err := s.eng.Load(&sh); err != nil {
		httpError(w, http.StatusUnprocessableEntity, err)
		return
	}
	applied := s.eng.Show()
	s.saver.Schedule(applied)
	s.broadcastShow(applied, "") // everyone, including the caller
	writeJSON(w, http.StatusOK, applied)
}

func (s *Server) libDelete(w http.ResponseWriter, r *http.Request) {
	if err := s.lib.Delete(library.Kind(r.PathValue("kind")), r.PathValue("name")); err != nil {
		libError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func libError(w http.ResponseWriter, err error) {
	if errors.Is(err, library.ErrNotFound) {
		httpError(w, http.StatusNotFound, err)
		return
	}
	httpError(w, http.StatusBadRequest, err)
}
