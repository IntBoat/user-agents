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
	"log"
	"testing"
)

// TestUpdateLatestUserAgents tests the UpdateLatestUserAgents function.
// It checks if the function successfully updates the latest user agents.
func TestUpdateLatestUserAgents(t *testing.T) {
	err := UpdateLatestUserAgents(true)
	if err != nil {
		t.Errorf("Error occured: %s", err.Error())
	}
}

// TestGetLatestUserAgents tests the GetLatestUserAgents function.
// It checks if the function returns a list of latest user agents, not an empty list.
func TestGetLatestUserAgents(t *testing.T) {
	// If the returned list of user agents is empty,
	// it means the function failed to return a list of latest user agents.
	if len(GetLatestUserAgents()) == 0 {
		t.Errorf("can not get latest user agents")
	}
}

// TestGetRandomUserAgent tests the GetRandomUserAgent function.
// It checks if the function returns a random user agent, not the default one.
func TestGetRandomUserAgent(t *testing.T) {
	// If the returned user agent is equal to the default user agent,
	// it means the function failed to return a random user agent.
	if GetRandomUserAgent() == DefaultUserAgent {
		t.Errorf("can not get random user agent")
	}
}

func TestGetRandomUserAgentByOSAndBrowser(t *testing.T) {
	userAgent := GetRandomUserAgentByOSAndBrowser("linux", "firefox")
	if userAgent == "" {
		t.Errorf("can not get random user agent by OS and browser")
	}
	log.Println(userAgent)
}
