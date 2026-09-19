package aegis

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func testServer(t *testing.T, target string, rpm, burst int) *Server { t.Helper(); s,err:=NewServer(Config{Routes:[]Route{{Prefix:"",Target:mustURL(t,target)}},RequestsPerMin:rpm,Burst:burst},slog.New(slog.NewTextHandler(io.Discard,nil))); if err!=nil { t.Fatal(err) }; return s }
func mustURL(t *testing.T, raw string) *url.URL { t.Helper(); u,err:=url.Parse(raw); if err!=nil {t.Fatal(err)}; return u }
func TestProxyAddsRequestID(t *testing.T) { upstream:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){ if r.Header.Get("X-Request-ID")=="" { t.Error("request id not forwarded") }; _,_=w.Write([]byte("proxied")) })); defer upstream.Close(); s:=testServer(t,upstream.URL,60,2); r:=httptest.NewRecorder(); s.Handler().ServeHTTP(r,httptest.NewRequest(http.MethodGet,"/anything",nil)); if r.Code!=http.StatusOK || r.Body.String()!="proxied" || r.Header().Get("X-Request-ID")=="" { t.Fatalf("unexpected response: %d %q",r.Code,r.Body.String()) } }
func TestRateLimitByIP(t *testing.T) { upstream:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){})); defer upstream.Close(); s:=testServer(t,upstream.URL,1,1); h:=s.Handler(); for i:=0;i<2;i++ { r:=httptest.NewRecorder(); req:=httptest.NewRequest(http.MethodGet,"/",nil); req.RemoteAddr="192.0.2.10:1234"; h.ServeHTTP(r,req); if i==0 && r.Code!=200 {t.Fatal(r.Code)}; if i==1 && r.Code!=429 {t.Fatal(r.Code)} } }
func TestHealthAndMetrics(t *testing.T) { s:=testServer(t,"http://example.com",60,1); for _,path:=range []string{"/health","/metrics"} { r:=httptest.NewRecorder(); s.Handler().ServeHTTP(r,httptest.NewRequest(http.MethodGet,path,nil)); if r.Code!=200 {t.Fatalf("%s: %d",path,r.Code)} } }
