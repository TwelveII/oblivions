package authcookie

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// The session cookie is the in-app fallback when a signed URL has expired.
// Wrong attributes break that fallback in subtle ways:
//   - SameSite=Strict/Lax → cookie not sent from Electron (file:// origin)
//   - Path=/ → cookie sent on every API mutation, expanding CSRF surface
//   - HttpOnly=false → JS can read the JWT, defeating the point
//   - Secure=false → cookie travels over plain HTTP
// These tests pin the contract so a future refactor can't quietly weaken it.

func TestSet_AttributesMatchContract(t *testing.T) {
	rr := httptest.NewRecorder()
	Set(rr, "jwt-token", time.Hour)

	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected exactly 1 cookie, got %d", len(cookies))
	}
	c := cookies[0]
	if c.Name != Name {
		t.Errorf("name: got %q want %q", c.Name, Name)
	}
	if c.Value != "jwt-token" {
		t.Errorf("value: got %q want %q", c.Value, "jwt-token")
	}
	if c.Path != Path {
		t.Errorf("path: got %q want %q (must be scoped to file endpoint only)", c.Path, Path)
	}
	if !c.HttpOnly {
		t.Error("HttpOnly must be true — JS access defeats cookie auth")
	}
	if !c.Secure {
		t.Error("Secure must be true — cookie carries access token, never plain HTTP")
	}
	if c.SameSite != http.SameSiteNoneMode {
		t.Errorf("SameSite: got %v want None — Electron renderer is file:// origin so anything stricter drops the cookie", c.SameSite)
	}
	if c.MaxAge != 3600 {
		t.Errorf("MaxAge: got %d want 3600 (1h)", c.MaxAge)
	}
}

func TestSet_MaxAgeMirrorsTokenTTL(t *testing.T) {
	cases := []struct {
		ttl  time.Duration
		want int
	}{
		{15 * time.Minute, 900},
		{1 * time.Hour, 3600},
		{7 * 24 * time.Hour, 7 * 86400},
	}
	for _, tc := range cases {
		rr := httptest.NewRecorder()
		Set(rr, "x", tc.ttl)
		got := rr.Result().Cookies()[0].MaxAge
		if got != tc.want {
			t.Errorf("ttl=%s: MaxAge got %d want %d", tc.ttl, got, tc.want)
		}
	}
}

func TestClear_ZeroValueAndNegativeMaxAge(t *testing.T) {
	rr := httptest.NewRecorder()
	Clear(rr)

	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	c := cookies[0]
	if c.Value != "" {
		t.Errorf("Clear must set empty value, got %q", c.Value)
	}
	if c.MaxAge >= 0 {
		t.Errorf("Clear must use negative MaxAge for immediate expiry, got %d", c.MaxAge)
	}
	// Path must match the Set path or browser keeps both cookies.
	if c.Path != Path {
		t.Errorf("Clear path %q must match Set path %q", c.Path, Path)
	}
}

func TestRead_ExtractsTokenFromRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/files/avatars/u/x.png", nil)
	req.AddCookie(&http.Cookie{Name: Name, Value: "the-jwt"})
	if got := Read(req); got != "the-jwt" {
		t.Errorf("Read: got %q want %q", got, "the-jwt")
	}
}

func TestRead_EmptyWhenAbsent(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/files/avatars/u/x.png", nil)
	if got := Read(req); got != "" {
		t.Errorf("Read with no cookie: got %q want empty", got)
	}
}

func TestRead_IgnoresUnrelatedCookies(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/files/avatars/u/x.png", nil)
	req.AddCookie(&http.Cookie{Name: "other_cookie", Value: "noise"})
	req.AddCookie(&http.Cookie{Name: Name, Value: "real-jwt"})
	if got := Read(req); got != "real-jwt" {
		t.Errorf("Read with other cookies present: got %q want %q", got, "real-jwt")
	}
}

func TestSetClear_RoundTripWithEncoding(t *testing.T) {
	// Tokens with characters that go through cookie value encoding (= and .)
	// must come back identical.
	rr := httptest.NewRecorder()
	jwt := "eyJhbGciOi.eyJ1c2VySWQi.signature_part="
	Set(rr, jwt, time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/api/files/x", nil)
	for _, c := range rr.Result().Cookies() {
		req.AddCookie(c)
	}
	if got := Read(req); got != jwt {
		t.Errorf("round-trip: got %q want %q", got, jwt)
	}
}

func TestSet_MultipleCallsOverwritePreviousCookie(t *testing.T) {
	rr := httptest.NewRecorder()
	Set(rr, "old", time.Hour)
	Set(rr, "new", time.Hour)

	// Both Set-Cookie headers will be sent; browser keeps only the newest.
	// We assert that the FIRST one we'd read back via header order is "new".
	headers := rr.Result().Header["Set-Cookie"]
	if len(headers) != 2 {
		t.Fatalf("expected 2 Set-Cookie headers (old + new), got %d", len(headers))
	}
	if !strings.Contains(headers[1], "=new") {
		t.Errorf("second Set-Cookie should carry new value, got %q", headers[1])
	}
}
