/*
 * Copyright (C) 2020-2024. IntBoat <intboat@gmail.com> - All Rights Reserved.
 *  Unauthorized copying of this file, via any medium is strictly prohibited
 *  Proprietary and confidential
 *
 * @package   user-agents\user_agents_test.go
 * @author    IntBoat <intboat@gmail.com>
 * @copyright 2020-2024. IntBoat <intboat@gmail.com>
 * @modified  5/20/24, 11:35 AM
 */

package user_agents

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func sampleUAHTML(ua string) string {
	return `<html><body><table><tr><td><ul><li><span class="code">` + ua + `</span></li></ul></td></tr></table></body></html>`
}

func withTempAPI(t *testing.T, handler http.HandlerFunc) (restore func()) {
	t.Helper()
	srv := httptest.NewServer(handler)
	prevBase := APIBase
	APIBase = srv.URL + "/"
	return func() {
		APIBase = prevBase
		srv.Close()
	}
}

func withTempWorkDir(t *testing.T) (restore func()) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	return func() {
		_ = os.Chdir(prev)
	}
}

func setCacheForTest(t *testing.T, uas []string, at time.Time) (restore func()) {
	t.Helper()
	mu.Lock()
	prevUAs := append([]string(nil), userAgentsCache...)
	prevTime := cacheTime
	userAgentsCache = append([]string(nil), uas...)
	cacheTime = at
	mu.Unlock()
	return func() {
		mu.Lock()
		userAgentsCache = prevUAs
		cacheTime = prevTime
		mu.Unlock()
	}
}

// TestUpdateLatestUserAgents exercises the live renew path used by CI.
// It writes user-agents.json into the process working directory on success.
// Skips when the upstream site is unreachable so unit verification still passes.
func TestUpdateLatestUserAgents(t *testing.T) {
	err := UpdateLatestUserAgents(true)
	if err != nil {
		t.Skipf("live update unavailable: %v", err)
	}
	if GetUserAgentCount() == 0 {
		t.Fatal("expected non-empty cache after successful live update")
	}
}

func TestEmbedSeededCache(t *testing.T) {
	if GetUserAgentCount() == 0 {
		t.Fatal("expected embed/config seed to populate cache when live update fails")
	}
	if GetRandomUserAgent() == "" {
		t.Fatal("random user agent should not be empty")
	}
	uas := GetLatestUserAgents()
	if len(uas) == 0 {
		t.Fatal("GetLatestUserAgents should return seeded agents")
	}
	// Defensive copy: mutating return value must not affect cache.
	uas[0] = "mutated"
	if GetLatestUserAgents()[0] == "mutated" {
		t.Fatal("GetLatestUserAgents must return a copy")
	}
}

func TestGetRandomUserAgent(t *testing.T) {
	if GetUserAgentCount() == 0 {
		t.Fatal("cache empty; cannot assert random selection")
	}
	ua := GetRandomUserAgent()
	if ua == "" {
		t.Fatal("expected non-empty user agent")
	}
}

func TestGetRandomUserAgentByOSAndBrowser(t *testing.T) {
	restore := setCacheForTest(t, []string{
		"Mozilla/5.0 (X11; Linux x86_64; rv:158.0) Gecko/20100101 Firefox/158.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36",
	}, time.Now())
	defer restore()

	ua := GetRandomUserAgentByOSAndBrowser("linux", "firefox")
	if !strings.Contains(strings.ToLower(ua), "linux") || !strings.Contains(strings.ToLower(ua), "firefox") {
		t.Fatalf("expected linux firefox UA, got %q", ua)
	}

	ua = GetRandomUserAgentByOSAndBrowser("nonexistent-os", "nonexistent-browser")
	if ua != DefaultUserAgent {
		t.Fatalf("expected default fallback, got %q", ua)
	}
}

func TestUpdateLatestUserAgentsMockMergeAndWrite(t *testing.T) {
	restoreDir := withTempWorkDir(t)
	defer restoreDir()

	chromeUA := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	firefoxUA := "Mozilla/5.0 (X11; Linux x86_64; rv:120.0) Gecko/20100101 Firefox/120.0"

	restoreAPI := withTempAPI(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/chrome"):
			_, _ = w.Write([]byte(sampleUAHTML(chromeUA)))
		case strings.HasSuffix(r.URL.Path, "/firefox"):
			_, _ = w.Write([]byte(sampleUAHTML(firefoxUA)))
		default:
			// safari/edge: empty but successful responses
			_, _ = w.Write([]byte(`<html><body><table></table></body></html>`))
		}
	})
	defer restoreAPI()

	restoreCache := setCacheForTest(t, nil, time.Time{})
	defer restoreCache()

	if err := UpdateLatestUserAgents(true); err != nil {
		t.Fatalf("update: %v", err)
	}

	got := GetLatestUserAgents()
	if len(got) != 2 {
		t.Fatalf("expected 2 agents, got %d: %v", len(got), got)
	}
	joined := strings.Join(got, "\n")
	if !strings.Contains(joined, chromeUA) || !strings.Contains(joined, firefoxUA) {
		t.Fatalf("merged agents missing expected values: %v", got)
	}

	out := filepath.Join(".", UserAgentFileName+"."+UserAgentFileType)
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("expected config write at %s: %v", out, err)
	}
	if !IsCacheValid() {
		t.Fatal("cache should be valid after successful update")
	}
}

func TestUpdateKeepsCacheWhenFetchEmpty(t *testing.T) {
	restoreDir := withTempWorkDir(t)
	defer restoreDir()

	seed := []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/1.0 Safari/537.36"}
	restoreCache := setCacheForTest(t, seed, time.Now().Add(-48*time.Hour))
	defer restoreCache()

	restoreAPI := withTempAPI(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body><table></table></body></html>`))
	})
	defer restoreAPI()

	err := UpdateLatestUserAgents(true)
	if err == nil {
		t.Fatal("expected error when no agents parsed")
	}
	got := GetLatestUserAgents()
	if len(got) != 1 || got[0] != seed[0] {
		t.Fatalf("cache should be preserved on empty fetch, got %v", got)
	}
}

func TestUpdateKeepsCacheWhenFetchErrors(t *testing.T) {
	restoreDir := withTempWorkDir(t)
	defer restoreDir()

	seed := []string{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 Version/17.0 Safari/605.1.15"}
	restoreCache := setCacheForTest(t, seed, time.Now().Add(-48*time.Hour))
	defer restoreCache()

	restoreAPI := withTempAPI(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "denied", http.StatusForbidden)
	})
	defer restoreAPI()

	if err := UpdateLatestUserAgents(true); err == nil {
		t.Fatal("expected error on HTTP failure")
	}
	got := GetLatestUserAgents()
	if len(got) != 1 || got[0] != seed[0] {
		t.Fatalf("cache should be preserved on fetch error, got %v", got)
	}
}

func TestUpdateLatestUserAgentsNoConcurrentMapPanic(t *testing.T) {
	restoreDir := withTempWorkDir(t)
	defer restoreDir()

	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	restoreAPI := withTempAPI(t, func(w http.ResponseWriter, r *http.Request) {
		// Respond with many matching entries to increase write pressure.
		var b strings.Builder
		b.WriteString(`<html><body><table><tr><td><ul>`)
		for i := 0; i < 50; i++ {
			b.WriteString(`<li><span class="code">` + ua + `</span></li>`)
		}
		b.WriteString(`</ul></td></tr></table></body></html>`)
		_, _ = w.Write([]byte(b.String()))
	})
	defer restoreAPI()

	restoreCache := setCacheForTest(t, nil, time.Time{})
	defer restoreCache()

	var wg sync.WaitGroup
	errCh := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errCh <- UpdateLatestUserAgents(true)
		}()
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent update failed: %v", err)
		}
	}
	if GetUserAgentCount() == 0 {
		t.Fatal("expected cache populated after concurrent updates")
	}
}

func TestGetRandomUserAgentEmptyCacheFallback(t *testing.T) {
	restore := setCacheForTest(t, nil, time.Time{})
	defer restore()
	if got := GetRandomUserAgent(); got != DefaultUserAgent {
		t.Fatalf("expected default, got %q", got)
	}
}
