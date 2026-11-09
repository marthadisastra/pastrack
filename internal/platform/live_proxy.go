package platform

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

// Live-linked local UI reads the same SQL database. Writes, report files and SSE
// use the authoritative server so all clients share notifications and file storage.
func (s *Server) proxyLive(w http.ResponseWriter, r *http.Request) bool {
	upstream := os.Getenv("PASTRACK_LIVE_UPSTREAM")
	if upstream == "" {
		return false
	}
	path := r.URL.Path
	mutation := r.Method != "GET" && r.Method != "HEAD" && path != "/api/v1/logout"
	if !strings.HasPrefix(path, "/api/v1/") || (!mutation && path != "/api/v1/realtime" && !strings.HasPrefix(path, "/api/v1/reports") && !strings.HasSuffix(path, "/remote-output")) {
		return false
	}
	target, err := url.Parse(upstream)
	if err != nil || target.Scheme != "https" || target.Host != "toko.azlaundry.my.id" || target.Path != "" {
		fail(w, 503, "invalid_live_upstream")
		return true
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	original := proxy.Director
	proxy.Director = func(request *http.Request) {
		original(request)
		request.Host = target.Host
		if request.Header.Get("Origin") != "" {
			request.Header.Set("Origin", upstream)
		}
		request.Header.Del("Referer")
	}
	proxy.FlushInterval = -1
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, e error) { fail(w, 502, "live_server_unavailable") }
	proxy.ServeHTTP(w, r)
	return true
}
