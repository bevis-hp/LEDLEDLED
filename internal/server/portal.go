package server

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// portal makes phones pop up the UI when they join the Wi-Fi.
//
// With wildcard DNS on the UI network (see deploy/setup-network.sh) every
// hostname resolves to the Pi, so the operating systems' connectivity
// probes land here:
//
//   - A device that hasn't opened the UI yet is redirected to it, which
//     makes the OS show its "sign in to network" sheet with the controller.
//   - Once a device has loaded the UI, its probes get the normal "online"
//     answer. iOS then offers "Done" instead of "Cancel" and Android stops
//     nagging, so the phone stays on the network.
//
// Requests for the Pi itself (any IP literal, the portal host, led.lan,
// *.local, localhost, the machine's hostname) are never intercepted.
type portal struct {
	url  string // where to send devices, e.g. "http://10.42.0.1/"
	host string // hostname part of url

	mu       sync.Mutex
	accepted map[string]time.Time // client IP -> when it last used the UI
}

// acceptFor is how long a device counts as "has seen the UI".
const acceptFor = 12 * time.Hour

// probeAnswers are the responses each OS expects when internet works.
var probeAnswers = map[string]struct {
	code int
	body string
}{
	"/hotspot-detect.html":       {200, "<HTML><HEAD><TITLE>Success</TITLE></HEAD><BODY>Success</BODY></HTML>"}, // Apple
	"/library/test/success.html": {200, "<HTML><HEAD><TITLE>Success</TITLE></HEAD><BODY>Success</BODY></HTML>"}, // Apple (old)
	"/generate_204":              {204, ""},                                                                     // Android / Chrome
	"/gen_204":                   {204, ""},                                                                     // Android
	"/connecttest.txt":           {200, "Microsoft Connect Test"},                                               // Windows 10+
	"/ncsi.txt":                  {200, "Microsoft NCSI"},                                                       // Windows (old)
	"/success.txt":               {200, "success\n"},                                                            // Firefox
	"/check_network_status.txt":  {200, "NetworkManager is online\n"},                                           // Linux NM
	"/canonical.html":            {200, "<meta http-equiv=\"refresh\" content=\"0;url=https://support.mozilla.org/kb/captive-portal\"/>"},
	"/kindle-wifi/wifistub.html": {200, "81ce4465-7167-4dcb-835b-dcc9e44c112a"}, // Kindle
	"/mobile/status.php":         {204, ""},                                     // Android (some OEMs)
	"/connectivity-check.html":   {204, ""},                                     // Ubuntu / misc
}

func newPortal(rawURL string) *portal {
	if rawURL == "" {
		return nil
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return nil
	}
	return &portal{url: rawURL, host: u.Hostname(), accepted: map[string]time.Time{}}
}

// wrap applies captive-portal handling in front of next. A nil portal is a
// no-op.
func (p *portal) wrap(next http.Handler) http.Handler {
	if p == nil {
		return next
	}
	self, _ := os.Hostname()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		host = strings.ToLower(strings.TrimSuffix(host, "."))
		client, _, _ := net.SplitHostPort(r.RemoteAddr)

		ours := net.ParseIP(host) != nil || host == p.host || host == "led.lan" || host == "localhost" ||
			strings.HasSuffix(host, ".local") || (self != "" && strings.EqualFold(host, self))
		if ours {
			p.accept(client)
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Cache-Control", "no-store")
		if ans, isProbe := probeAnswers[r.URL.Path]; isProbe && p.isAccepted(client) {
			if ans.code != http.StatusNoContent {
				w.Header().Set("Content-Type", "text/html")
			}
			w.WriteHeader(ans.code)
			_, _ = w.Write([]byte(ans.body))
			return
		}
		http.Redirect(w, r, p.url, http.StatusFound)
	})
}

func (p *portal) accept(ip string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	p.accepted[ip] = now
	for k, t := range p.accepted { // tiny map; prune opportunistically
		if now.Sub(t) > acceptFor {
			delete(p.accepted, k)
		}
	}
}

func (p *portal) isAccepted(ip string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	t, ok := p.accepted[ip]
	return ok && time.Since(t) < acceptFor
}
