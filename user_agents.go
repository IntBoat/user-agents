/*
 * Copyright (C) 2020-2023. IntBoat <intboat@gmail.com> - All Rights Reserved.
 *  Unauthorized copying of this file, via any medium is strictly prohibited
 *  Proprietary and confidential
 *
 * @package   user-agents\main.go
 * @author    IntBoat <intboat@gmail.com>
 * @copyright 2020-2023. IntBoat <intboat@gmail.com>
 * @modified  2/15/23, 12:44 AM
 */

package user_agents

import (
	"context"
	"github.com/PuerkitoBio/goquery"
	"github.com/carlmjohnson/requests"
	"github.com/spf13/viper"
	"math/rand"
	"regexp"
	"strings"
	"time"
)

var (
	UserAgentFileName = "user-agents.json"
	APIBase           = "https://www.whatismybrowser.com/guides/the-latest-user-agent/"
	IncludePatterns   = []string{
		`(?i)windows nt \d+\.\d+`,
		`(?i)macintosh`,
		`(?i)linux (x86_64|i686)`,
	}
)

func init() {
	viper.SetConfigName(UserAgentFileName)
	viper.SetConfigType(strings.Split(UserAgentFileName, ".")[1])
	viper.AddConfigPath(".")
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {
		viper.SafeWriteConfig()
	}
	viper.WatchConfig()

	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for {
			select {
			case <-ticker.C:
				_ = UpdateLatestUserAgents()
			}
		}
	}()
	UpdateLatestUserAgents()
}

func GetRandomUserAgent() string {
	ua := viper.GetStringSlice("user-agent")
	if len(ua) == 0 {
		return "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/109.0"
	}
	return ua[rand.Intn(len(ua))]
}

func UpdateLatestUserAgents() error {
	lastUpdate := viper.GetTime("last-update")
	if !lastUpdate.Before(time.Now().Add(-24 * time.Hour)) {
		return nil
	}

	var uas []string
	for _, v := range []string{"chrome", "firefox", "safari", "edge"} {
		var html string
		err := requests.
			URL(APIBase + v).
			UserAgent(GetRandomUserAgent()).
			ToString(&html).
			Fetch(context.Background())
		if err != nil {
			return err
		}

		// Load the HTML document
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
		if err != nil {
			return err
		}

		// Find the review items
		doc.Find("td li span.code").Each(func(i int, s *goquery.Selection) {
			for _, pattern := range IncludePatterns {
				var include = regexp.MustCompile(pattern)
				if include.MatchString(s.Text()) {
					uas = append(uas, s.Text())
				}
			}
		})
	}
	viper.Set("user-agent", uas)
	viper.Set("last-update", time.Now())
	err := viper.WriteConfig()
	return err
}

func GetLatestUserAgents() []string {
	return viper.GetStringSlice("user-agent")
}
