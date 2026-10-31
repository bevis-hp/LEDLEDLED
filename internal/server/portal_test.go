package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCaptivePortalFlow(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(299) })
	h := newPortal("http://10.42.0.1/").wrap(ok)

	get := func(host, path, client string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "http://"+host+path, nil)
		r.RemoteAddr = client + ":5555"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}

	// New phone: Apple probe is redirected to the UI.
	w := get("captive.apple.com", "/hotspot-detect.html", "10.42.0.50")
	if w.Code != http.StatusFound || w.Header().Get("Location") != "http://10.42.0.1/" {
		t.Fatalf("probe before UI: %d %q", w.Code, w.Header().Get("Location"))
	}
	// Any other foreign host also lands on the UI.
	if w := get("example.com", "/anything", "10.42.0.50"); w.Code != http.StatusFound {
		t.Fatalf("foreign host: %d", w.Code)
	}
	// Loading the UI (by IP) passes through and marks the phone.
	if w := get("10.42.0.1", "/", "10.42.0.50"); w.Code != 299 {
		t.Fatalf("own host: %d", w.Code)
	}
	// Now probes get the "online" answers.
	if w := get("captive.apple.com", "/hotspot-detect.html", "10.42.0.50"); w.Code != 200 || w.Body.String() == "" {
		t.Fatalf("apple probe after UI: %d", w.Code)
	}
	if w := get("connectivitycheck.gstatic.com", "/generate_204", "10.42.0.50"); w.Code != 204 {
		t.Fatalf("android probe after UI: %d", w.Code)
	}
	// A different phone is still redirected.
	if w := get("connectivitycheck.gstatic.com", "/generate_204", "10.42.0.51"); w.Code != http.StatusFound {
		t.Fatalf("other phone: %d", w.Code)
	}
	// Local names are never intercepted.
	for _, host := range []string{"led.lan", "ledpi.local", "localhost:8080", "192.168.1.20:80"} {
		if w := get(host, "/api/status", "192.168.1.9"); w.Code != 299 {
			t.Errorf("%s intercepted: %d", host, w.Code)
		}
	}
}

func TestPortalDisabled(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(299) })
	h := newPortal("").wrap(ok)
	r := httptest.NewRequest("GET", "http://captive.apple.com/hotspot-detect.html", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 299 {
		t.Fatalf("disabled portal intercepted: %d", w.Code)
	}
}
