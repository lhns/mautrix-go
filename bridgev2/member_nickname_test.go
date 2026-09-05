// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package bridgev2

import (
	"testing"

	"maunium.net/go/mautrix/bridgev2/networkid"
	"maunium.net/go/mautrix/event"
)

func TestMemberDisplayname(t *testing.T) {
	tests := []struct {
		name        string
		current     *event.MemberEventContent
		nickname    string
		hasNickname bool
		want        string
	}{
		{
			name:        "nickname overrides what the room has",
			current:     &event.MemberEventContent{Displayname: "Global Name"},
			nickname:    "Per-room Name",
			hasNickname: true,
			want:        "Per-room Name",
		},
		{
			// The zero-churn property: with no nickname the comparison in syncParticipants
			// collapses to the original membership-only check, so a connector that never sets
			// ChatMember.Nickname produces no extra member events.
			name:    "no nickname keeps what the room has",
			current: &event.MemberEventContent{Displayname: "Global Name"},
			want:    "Global Name",
		},
		{
			name:        "an empty nickname is a value, not an absence",
			current:     &event.MemberEventContent{Displayname: "Global Name"},
			nickname:    "",
			hasNickname: true,
			want:        "",
		},
		{
			name: "no current member event and no nickname",
			want: "",
		},
		{
			name:        "no current member event but a nickname",
			nickname:    "Per-room Name",
			hasNickname: true,
			want:        "Per-room Name",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := memberDisplayname(test.current, test.nickname, test.hasNickname)
			if got != test.want {
				t.Errorf("memberDisplayname() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestPickNicknameOwner(t *testing.T) {
	tests := []struct {
		name     string
		receiver networkid.UserLoginID
		logins   []networkid.UserLoginID
		want     networkid.UserLoginID
	}{
		{
			name:     "a receiver owns the portal outright",
			receiver: "login2",
			logins:   []networkid.UserLoginID{"login1", "login2"},
			want:     "login2",
		},
		{
			name:     "a receiver wins even when it is not in the login list",
			receiver: "login9",
			logins:   []networkid.UserLoginID{"login1"},
			want:     "login9",
		},
		{
			name:   "a shared portal takes the lowest login ID",
			logins: []networkid.UserLoginID{"login2", "login1", "login3"},
			want:   "login1",
		},
		{
			// The property that actually prevents flapping: whichever login's sync runs, and
			// in whatever order the logins come back, the same one owns the names.
			name:   "the pick does not depend on list order",
			logins: []networkid.UserLoginID{"login3", "login1", "login2"},
			want:   "login1",
		},
		{
			name:   "no logins means no owner",
			logins: nil,
			want:   "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := pickNicknameOwner(test.receiver, test.logins)
			if got != test.want {
				t.Errorf("pickNicknameOwner() = %q, want %q", got, test.want)
			}
		})
	}
}
