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

func TestUpdateLatestUserAgents(t *testing.T) {
	err := UpdateLatestUserAgents(true)
	if err != nil {
		t.Errorf("Error occured: %s", err.Error())
	}
}

func TestGetLatestUserAgents(t *testing.T) {
	if len(GetLatestUserAgents()) == 0 {
		t.Errorf("can not get latest user agents")
	}
}

func TestGetRandomUserAgent(t *testing.T) {
	if GetRandomUserAgent() == DefaultUserAgent {
		t.Errorf("can not get random user agent")
	}
}
