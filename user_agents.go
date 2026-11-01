/*
 * Copyright (C) 2020-2025. IntBoat <intboat@gmail.com> - All Rights Reserved.
 * Unauthorized copying or redistribution of this file in source and binary forms via any medium is strictly prohibited.
 * Proprietary and confidential
 *
 * @author    IntBoat <intboat@gmail.com>
 * @copyright 2020-2025. IntBoat <intboat@gmail.com>
 * @modified  2025-08-19 15:15:03
 */

package user_agents

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/carlmjohnson/requests"
	"github.com/spf13/viper"
)

var (
	UserAgentFileName = "user-agents"
	UserAgentFileType = "json"
	APIBase           = "https://www.whatismybrowser.com/guides/the-latest-user-agent/"
	DefaultUserAgent  = "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:141.0) Gecko/20100101 Firefox/141.0"

	//go:embed user-agents.json
	f []byte

	// 預編譯正則表達式以提高效能
	includePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)windows nt \d+\.\d+`),
		regexp.MustCompile(`(?i)macintosh`),
		regexp.MustCompile(`(?i)linux (x86_64|i686)`),
	}

	// 支援的瀏覽器列表
	supportedBrowsers = []string{"chrome", "firefox", "safari", "edge"}

	// 互斥鎖防止並發問題
	mu sync.RWMutex

	// 快取用戶代理列表
	userAgentsCache []string
	cacheTime       time.Time
	cacheValid      = 24 * time.Hour
)

// init initializes the package by reading the user agent configuration file and starting a ticker to update the user agents every 24 hours.
func init() {
	viper.SetConfigName(UserAgentFileName)
	viper.SetConfigType(UserAgentFileType)
	viper.AddConfigPath(".")

	if err := viper.ReadConfig(bytes.NewReader(f)); err != nil {
		log.Printf("Warning: Failed to read config: %v", err)
	} else {
		seedCacheFromViper()
	}

	// 啟動背景更新任務
	go startBackgroundUpdate()

	// 初始更新（失敗時保留已灌入的 embed 快取）
	if err := UpdateLatestUserAgents(false); err != nil {
		log.Printf("Initial update failed: %v", err)
	}
}

// seedCacheFromViper loads embedded/config user agents into the in-memory cache.
func seedCacheFromViper() {
	uas := viper.GetStringSlice("user-agent")
	if len(uas) == 0 {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	userAgentsCache = append([]string(nil), uas...)
	if t := viper.GetTime("last-update"); !t.IsZero() {
		cacheTime = t
	} else {
		cacheTime = time.Now()
	}
}

// startBackgroundUpdate starts a background goroutine that updates user agents every 24 hours
func startBackgroundUpdate() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		if err := UpdateLatestUserAgents(true); err != nil {
			log.Printf("Background update failed: %v", err)
		}
	}
}

// GetRandomUserAgent returns a random user agent from the latest user agents list.
// If the list is empty, it returns the default user agent.
func GetRandomUserAgent() string {
	mu.RLock()
	defer mu.RUnlock()

	if len(userAgentsCache) == 0 {
		return DefaultUserAgent
	}
	return userAgentsCache[rand.Intn(len(userAgentsCache))]
}

// UpdateLatestUserAgents updates the latest user agents from the provided API.
// It retrieves the latest user agents for Chrome, Firefox, Safari, and Edge browsers.
// It also checks if the last update was within the last 24 hours, and only updates if necessary.
// It returns an error if any occurs during the update process.
func UpdateLatestUserAgents(force bool) error {
	mu.RLock()
	skip := !force && !cacheTime.IsZero() && time.Since(cacheTime) < cacheValid
	mu.RUnlock()
	if skip {
		return nil
	}

	// 使用 context 設定超時
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	type fetchResult struct {
		uas []string
		err error
	}

	results := make(chan fetchResult, len(supportedBrowsers))
	var wg sync.WaitGroup

	for _, browser := range supportedBrowsers {
		wg.Add(1)
		go func(b string) {
			defer wg.Done()
			uas, err := fetchUserAgentsForBrowser(ctx, b)
			results <- fetchResult{uas: uas, err: err}
		}(browser)
	}

	wg.Wait()
	close(results)

	uaSet := make(map[string]bool)
	var firstErr error
	for r := range results {
		if r.err != nil && firstErr == nil {
			firstErr = r.err
		}
		for _, ua := range r.uas {
			uaSet[ua] = true
		}
	}
	if firstErr != nil {
		return firstErr
	}
	if len(uaSet) == 0 {
		return fmt.Errorf("no user agents fetched; keeping existing cache")
	}

	uas := make([]string, 0, len(uaSet))
	for ua := range uaSet {
		uas = append(uas, ua)
	}

	now := time.Now()

	mu.Lock()
	defer mu.Unlock()

	userAgentsCache = uas
	cacheTime = now
	viper.Set("user-agent", uas)
	viper.Set("last-update", now)

	return viper.WriteConfigAs(UserAgentFileName + "." + UserAgentFileType)
}

// fetchUserAgentsForBrowser fetches user agents for a specific browser.
// It returns a dedicated slice so concurrent callers do not share a map.
func fetchUserAgentsForBrowser(ctx context.Context, browser string) ([]string, error) {
	var html string

	mu.RLock()
	userAgent := DefaultUserAgent
	if len(userAgentsCache) > 0 {
		userAgent = userAgentsCache[rand.Intn(len(userAgentsCache))]
	}
	mu.RUnlock()

	err := requests.URL(APIBase + browser).
		UserAgent(userAgent).
		ToString(&html).
		Fetch(ctx)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	var uas []string
	doc.Find("td li span.code").Each(func(i int, s *goquery.Selection) {
		ua := strings.TrimSpace(s.Text())
		if ua == "" {
			return
		}

		for _, pattern := range includePatterns {
			if pattern.MatchString(ua) {
				uas = append(uas, ua)
				break
			}
		}
	})

	return uas, nil
}

// GetLatestUserAgents returns all cached user agents
func GetLatestUserAgents() []string {
	mu.RLock()
	defer mu.RUnlock()

	result := make([]string, len(userAgentsCache))
	copy(result, userAgentsCache)
	return result
}

// GetRandomUserAgentByOSAndBrowser returns a random user agent that matches the specified OS and browser.
// If no matching user agent is found, it returns the default user agent.
func GetRandomUserAgentByOSAndBrowser(os, browser string) string {
	mu.RLock()
	defer mu.RUnlock()

	if len(userAgentsCache) == 0 {
		return DefaultUserAgent
	}

	osLower := strings.ToLower(os)
	browserLower := strings.ToLower(browser)

	var matchedUA []string
	for _, agent := range userAgentsCache {
		agentLower := strings.ToLower(agent)
		if strings.Contains(agentLower, osLower) && strings.Contains(agentLower, browserLower) {
			matchedUA = append(matchedUA, agent)
		}
	}

	if len(matchedUA) == 0 {
		return DefaultUserAgent
	}
	return matchedUA[rand.Intn(len(matchedUA))]
}

// GetUserAgentCount returns the number of cached user agents
func GetUserAgentCount() int {
	mu.RLock()
	defer mu.RUnlock()
	return len(userAgentsCache)
}

// IsCacheValid returns whether the current cache is still valid
func IsCacheValid() bool {
	mu.RLock()
	defer mu.RUnlock()
	return !cacheTime.IsZero() && time.Since(cacheTime) < cacheValid
}
