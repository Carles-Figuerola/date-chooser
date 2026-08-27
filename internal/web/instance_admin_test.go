package web

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/cfiguerola/date-chooser/internal/store"
)

func TestInstanceAdmin_PageRedirectsToLoginWithoutSession(t *testing.T) {
	ts, _ := newTestServer(t)

	resp, err := noRedirectClient().Get(ts.URL + "/admin")
	if err != nil {
		t.Fatalf("GET /admin error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/admin/login" {
		t.Fatalf("expected redirect to /admin/login, got %q", loc)
	}
}

func TestInstanceAdmin_LoginRejectsWrongSecret(t *testing.T) {
	ts, _ := newTestServer(t)

	form := url.Values{}
	form.Set("secret", "definitely-wrong")

	resp, err := noRedirectClient().PostForm(ts.URL+"/admin/login", form)
	if err != nil {
		t.Fatalf("POST /admin/login error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 (re-render with error), got %d", resp.StatusCode)
	}
	if resp.Header.Get("Set-Cookie") != "" {
		t.Fatalf("expected no cookie set on a failed login, got Set-Cookie: %q", resp.Header.Get("Set-Cookie"))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body: %v", err)
	}
	if !strings.Contains(string(body), instanceAdminLoginErrorCopy) {
		t.Fatalf("expected the error banner %q in the response, got: %s", instanceAdminLoginErrorCopy, body)
	}
}

func TestInstanceAdmin_LoginWithCorrectSecretSetsSessionOnlyCookieAndUnlocksPage(t *testing.T) {
	ts, st := newTestServer(t)

	_, err := st.CreatePoll(context.Background(), store.NewPollInput{
		Title:    "Admin List Test",
		PollType: "all_day",
		Slots:    []store.NewSlotInput{{StartsAt: "2026-09-01"}},
	}, "ptok-instance-admin-1", "atok-instance-admin-1")
	if err != nil {
		t.Fatalf("CreatePoll() error: %v", err)
	}

	form := url.Values{}
	form.Set("secret", testInstanceAdminSecret)

	client := noRedirectClient()
	resp, err := client.PostForm(ts.URL+"/admin/login", form)
	if err != nil {
		t.Fatalf("POST /admin/login error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/admin" {
		t.Fatalf("expected redirect to /admin, got %q", loc)
	}

	setCookie := resp.Header.Get("Set-Cookie")
	if setCookie == "" {
		t.Fatalf("expected a Set-Cookie header on successful login, got none")
	}
	if strings.Contains(strings.ToLower(setCookie), "max-age") || strings.Contains(strings.ToLower(setCookie), "expires") {
		t.Fatalf("expected a session-only cookie (no Max-Age/Expires), got: %s", setCookie)
	}

	var sessionCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == instanceAdminCookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatalf("expected a %s cookie, got none", instanceAdminCookieName)
	}
	if sessionCookie.Value == testInstanceAdminSecret {
		t.Fatalf("expected the session cookie to be a fresh token distinct from the instance-admin secret, got the secret itself")
	}

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/admin", nil)
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	req.AddCookie(sessionCookie)

	pageResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET /admin error: %v", err)
	}
	defer pageResp.Body.Close()

	if pageResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", pageResp.StatusCode)
	}
	body, err := io.ReadAll(pageResp.Body)
	if err != nil {
		t.Fatalf("reading response body: %v", err)
	}
	if !strings.Contains(string(body), "Admin List Test") {
		t.Fatalf("expected the created poll's title in the admin page, got: %s", body)
	}
	if !strings.Contains(string(body), "ptok-instance-admin-1") {
		t.Fatalf("expected the poll's participant link in the admin page, got: %s", body)
	}
	if !strings.Contains(string(body), "atok-instance-admin-1") {
		t.Fatalf("expected the poll's admin link in the admin page, got: %s", body)
	}
}

func TestInstanceAdmin_ExpiredSessionCookieIsRejectedAndPruned(t *testing.T) {
	ts, st := newTestServer(t)
	ctx := context.Background()

	old := time.Now().UTC().Add(-25 * time.Hour).Format(time.RFC3339)
	if _, err := st.DB().ExecContext(ctx, "INSERT INTO admin_sessions (token, created_at) VALUES (?, ?)", "old-session-token", old); err != nil {
		t.Fatalf("seeding an expired session: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/admin", nil)
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: instanceAdminCookieName, Value: "old-session-token"})

	resp, err := noRedirectClient().Do(req)
	if err != nil {
		t.Fatalf("GET /admin error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected a 25h-old session to be rejected (303 to login), got %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/admin/login" {
		t.Fatalf("expected redirect to /admin/login, got %q", loc)
	}

	// GET /admin also prunes expired rows as a side effect — confirm the
	// row is actually gone, not just rejected for this one request.
	var count int
	if err := st.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM admin_sessions WHERE token = 'old-session-token'").Scan(&count); err != nil {
		t.Fatalf("querying admin_sessions: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected GET /admin to prune the expired session row, still found %d row(s)", count)
	}
}
