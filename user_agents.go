/*
 * Copyright (C) 2020-2024. IntBoat <intboat@gmail.com> - All Rights Reserved.
 *  Unauthorized copying of this file, via any medium is strictly prohibited
 *  Proprietary and confidential
 *
 * @package   user-agents\user_agents.go
 * @author    IntBoat <intboat@gmail.com>
 * @copyright 2020-2024. IntBoat <intboat@gmail.com>
 * @modified  5/20/24, 11:33 AM
 */

package user_agents

import (
	"bytes"
	"context"
	_ "embed"
	"github.com/PuerkitoBio/goquery"
	"github.com/carlmjohnson/requests"
	"github.com/spf13/viper"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"time"
)

var (
	UserAgentFileName = "user-agents"
	UserAgentFileType = "json"
	APIBase           = "https://www.whatismybrowser.com/guides/the-latest-user-agent/"
	IncludePatterns   = []string{
		`(?i)windows nt \d+\.\d+`,
		`(?i)macintosh`,
		`(?i)linux (x86_64|i686)`,
	}
	DefaultUserAgent = "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:126.0) Gecko/20100101 Firefox/126.0"

	//go:embed user-agents.json
	f []byte
)

// init initializes the package by reading the user agent configuration file and starting a ticker to update the user agents every 24 hours.
func init() {
	viper.New()
	viper.SetConfigName(UserAgentFileName)
	viper.SetConfigType(UserAgentFileType)
	viper.AddConfigPath(".")
	err := viper.ReadConfig(bytes.NewReader(f)) // Find and read the config file
	if err != nil {
	}

	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for {
			select {
			case <-ticker.C:
				_ = UpdateLatestUserAgents(true)
			}
		}
	}()
	err = UpdateLatestUserAgents(false)
	if err != nil {
		log.Println(err)
		return
	}
}

// GetRandomUserAgent returns a random user agent from the latest user agents list.
// If the list is empty, it returns the default user agent.
func GetRandomUserAgent() string {
	ua := viper.GetStringSlice("user-agent")
	if len(ua) == 0 {
		return DefaultUserAgent
	}
	return ua[rand.Intn(len(ua))]
}

// UpdateLatestUserAgents updates the latest user agents from the provided API.
// It retrieves the latest user agents for Chrome, Firefox, Safari, and Edge browsers.
// It also checks if the last update was within the last 24 hours, and only updates if necessary.
// It returns an error if any occurs during the update process.
func UpdateLatestUserAgents(force bool) error {
	// Get the last update time from the configuration file.
	lastUpdate := viper.GetTime("last-update")

	// Check if the update is forced or if the last update was within the last 24 hours.
	if !force && !lastUpdate.Before(time.Now().Add(-24*time.Hour)) {
		return nil
	}

	// Initialize an empty slice to store the latest user agents.
	var uas []string

	// Iterate over the supported browsers (Chrome, Firefox, Safari, and Edge).
	for _, v := range []string{"chrome", "firefox", "safari", "edge"} {
		// Initialize an empty string to store the HTML content.
		var html string

		// Make a request to the API endpoint for the specified browser.
		// Use the GetRandomUserAgent function as the UserAgent for the request.
		err := requests.URL(APIBase + v).UserAgent(GetRandomUserAgent()).ToString(&html).Fetch(context.Background())
		if err != nil {
			return err
		}

		// Parse the HTML content using the goquery library.
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
		// Check if an error occurred during the parsing.
		if err != nil {
			return err
		}

		// Find all the review items containing the latest user agents.
		doc.Find("td li span.code").Each(func(i int, s *goquery.Selection) {
			// Iterate over the include patterns.
			for _, pattern := range IncludePatterns {
				// Compile the include pattern as a regular expression.
				var include = regexp.MustCompile(pattern)

				// Check if the current user agent matches any of the include patterns.
				if include.MatchString(s.Text()) {
					// Append the user agent to the slice of latest user agents.
					uas = append(uas, s.Text())
				}
			}
		})
	}

	// Set the latest user agents in the configuration file.
	viper.Set("user-agent", uas)

	// Set the current time as the last update time in the configuration file.
	viper.Set("last-update", time.Now())

	// Write the updated configuration file.
	err := viper.WriteConfigAs(UserAgentFileName + "." + UserAgentFileType)
	return err
}

func GetLatestUserAgents() []string {
	return viper.GetStringSlice("user-agent")
}

// GetRandomUserAgentByOSAndBrowser returns a random user agent from the latest user agents list that matches the specified OS and browser.
// If no matching user agent is found, it returns the default user agent.
//
// Parameters:
//   - os (string): The operating system to filter the user agents by.
//   - browser (string): The browser to filter the user agents by.
//
// Returns:
//   - (string): A random user agent that matches the specified OS and browser, or the default user agent if no matching user agent is found.
func GetRandomUserAgentByOSAndBrowser(os, browser string) string {
	ua := viper.GetStringSlice("user-agent")
	var matchedUA []string
	for _, agent := range ua {
		if strings.Contains(strings.ToLower(agent), strings.ToLower(os)) && strings.Contains(strings.ToLower(agent), strings.ToLower(browser)) {
			matchedUA = append(matchedUA, agent)
		}
	}
	if len(matchedUA) == 0 {
		return DefaultUserAgent
	}
	return matchedUA[rand.Intn(len(matchedUA))]
}
