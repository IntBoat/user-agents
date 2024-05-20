/*
 * Copyright (C) 2020-2023. IntBoat <intboat@gmail.com> - All Rights Reserved.
 *  Unauthorized copying of this file, via any medium is strictly prohibited
 *  Proprietary and confidential
 *
 * @package   user-agents\user_agents_test.go
 * @author    IntBoat <intboat@gmail.com>
 * @copyright 2020-2023. IntBoat <intboat@gmail.com>
 * @modified  2/15/23, 2:41 AM
 */

package user_agents

import (
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
