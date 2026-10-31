package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"ledledled/internal/engine"
	"ledledled/internal/library"
	"ledledled/internal/show"
)

func newTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	sh := show.Default()
	for i := range sh.Controllers {
		sh.Controllers[i].IP = "127.0.0.1"
		sh.Controllers[i].Disabled = true
	}
	eng, err := engine.New(sh)
	if err != nil {
		t.Fatal(err)
	}
	ui := fstest.MapFS{"index.html": {Data: []byte("<html>ui</html>")}}
	dir := t.TempDir()
	lib, err := library.New(filepath.Join(dir, "library"))
	if err != nil {
		t.Fatal(err)
	}
	s := New(eng, filepath.Join(dir, "show.json"), lib, ui)
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return s, ts
}

func TestPutShowValidatesAndApplies(t *testing.T) {
	_, ts := newTestServer(t)

	var sh show.Show
	resp, err := http.Get(ts.URL + "/api/show")
	if err != nil {
		t.Fatal(err)
	}
	_ = json.NewDecoder(resp.Body).Decode(&sh)
	resp.Body.Close()

	// A new layer without ID or params gets them filled in.
	sh.Layers = append(sh.Layers, show.Layer{Name: "red", Effect: "solid"})
	body, _ := json.Marshal(sh)
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/show", bytes.NewReader(body))
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var got show.Show
	_ = json.NewDecoder(resp.Body).Decode(&got)
	resp.Body.Close()
	if resp.StatusCode != 200 || len(got.Layers) != 2 || got.Layers[1].ID == "" || got.Layers[1].Params["color"] == nil {
		t.Fatalf("status %d, layers %+v", resp.StatusCode, got.Layers)
	}

	// Unknown effect is rejected.
	sh.Layers[0].Effect = "nope"
	body, _ = json.Marshal(sh)
	req, _ = http.NewRequest(http.MethodPut, ts.URL+"/api/show", bytes.NewReader(body))
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("bad show status = %d", resp.StatusCode)
	}
}

func TestStaticFallsBackToIndex(t *testing.T) {
	_, ts := newTestServer(t)
	resp, err := http.Get(ts.URL + "/some/client/route")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestLibrarySaveLoadAndPresets(t *testing.T) {
	_, ts := newTestServer(t)
	do := func(method, path, body string) *http.Response {
		t.Helper()
		req, _ := http.NewRequest(method, ts.URL+path, strings.NewReader(body))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp
	}

	if r := do("POST", "/api/library/shows/Gig%201/save", ""); r.StatusCode != 204 {
		t.Fatalf("save status %d", r.StatusCode)
	}
	if r := do("POST", "/api/library/shows/Gig%201/load", ""); r.StatusCode != 200 {
		t.Fatalf("load status %d", r.StatusCode)
	}
	if r := do("POST", "/api/library/shows/nope/load", ""); r.StatusCode != 404 {
		t.Fatalf("missing load status %d", r.StatusCode)
	}
	if r := do("POST", "/api/library/shows/..%2Fx/save", ""); r.StatusCode != 400 {
		t.Fatalf("traversal name status %d", r.StatusCode)
	}
	if r := do("PUT", "/api/library/presets/Red", `{"layer":{"effect":"solid","params":{"color":"#ff0000"}}}`); r.StatusCode != 204 {
		t.Fatalf("preset put status %d", r.StatusCode)
	}
	if r := do("PUT", "/api/library/presets/Bad", `{"layer":{"effect":"nope"}}`); r.StatusCode != 422 {
		t.Fatalf("bad preset status %d", r.StatusCode)
	}

	resp, err := http.Get(ts.URL + "/api/library/presets")
	if err != nil {
		t.Fatal(err)
	}
	var list []library.Entry
	_ = json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	if len(list) != 1 || list[0].Name != "Red" {
		t.Fatalf("preset list = %+v", list)
	}
}
