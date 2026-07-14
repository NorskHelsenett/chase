package handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/norskhelsenett/chase/auth"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

func newAuthTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/login", HandleLogin)
	r.GET("/api/callback", HandleCallback)
	return r
}

func cookieValue(t *testing.T, resp *http.Response, name string) string {
	t.Helper()
	for _, c := range resp.Cookies() {
		if c.Name == name {
			return c.Value
		}
	}
	t.Fatalf("cookie %q not set", name)
	return ""
}

func TestHandleLoginSetsPKCEChallengeAndState(t *testing.T) {
	auth.Config = oauth2.Config{
		ClientID:    "test-client",
		RedirectURL: "http://localhost/api/callback",
		Endpoint:    oauth2.Endpoint{AuthURL: "https://idp.example/authorize"},
		Scopes:      []string{"openid"},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/login", nil)
	newAuthTestRouter().ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", w.Code)
	}

	location, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatalf("invalid redirect location: %v", err)
	}
	query := location.Query()

	if query.Get("code_challenge_method") != "S256" {
		t.Errorf("expected code_challenge_method=S256, got %q", query.Get("code_challenge_method"))
	}

	resp := w.Result()
	state := cookieValue(t, resp, stateCookie)
	verifier := cookieValue(t, resp, verifierCookie)

	if query.Get("state") != state {
		t.Errorf("state in redirect (%q) does not match state cookie (%q)", query.Get("state"), state)
	}

	sum := sha256.Sum256([]byte(verifier))
	wantChallenge := base64.RawURLEncoding.EncodeToString(sum[:])
	if query.Get("code_challenge") != wantChallenge {
		t.Errorf("code_challenge does not match S256 hash of verifier cookie")
	}
}

func TestHandleCallbackRejectsStateMismatch(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/callback?code=abc&state=attacker", nil)
	req.AddCookie(&http.Cookie{Name: stateCookie, Value: "legit-state"})
	req.AddCookie(&http.Cookie{Name: verifierCookie, Value: "some-verifier"})
	newAuthTestRouter().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for state mismatch, got %d", w.Code)
	}
}

func TestHandleCallbackRejectsMissingStateCookie(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/callback?code=abc&state=whatever", nil)
	newAuthTestRouter().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing state cookie, got %d", w.Code)
	}
}

func TestHandleCallbackSendsVerifierToTokenEndpoint(t *testing.T) {
	var gotVerifier string
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("failed to parse token request form: %v", err)
		}
		gotVerifier = r.FormValue("code_verifier")
		// Return an invalid token so the handler stops after the exchange step;
		// this test only cares about what reached the token endpoint.
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer tokenServer.Close()

	auth.Config = oauth2.Config{
		ClientID:    "test-client",
		RedirectURL: "http://localhost/api/callback",
		Endpoint:    oauth2.Endpoint{TokenURL: tokenServer.URL},
	}

	verifier := oauth2.GenerateVerifier()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/callback?code=abc&state=legit-state", nil)
	req.AddCookie(&http.Cookie{Name: stateCookie, Value: "legit-state"})
	req.AddCookie(&http.Cookie{Name: verifierCookie, Value: verifier})
	newAuthTestRouter().ServeHTTP(w, req)

	if gotVerifier != verifier {
		t.Errorf("token endpoint received code_verifier %q, want %q", gotVerifier, verifier)
	}

	// The auth-flow cookies must be cleared once the callback consumes them.
	for _, c := range w.Result().Cookies() {
		if (c.Name == stateCookie || c.Name == verifierCookie) && c.MaxAge >= 0 {
			t.Errorf("cookie %q was not cleared after callback", c.Name)
		}
	}
}
