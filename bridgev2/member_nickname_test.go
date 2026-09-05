// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package bridgev2

import (
	"testing"

	"maunium.net/go/mautrix/event"
)

func TestMemberDisplayname(t *testing.T) {
	nickname := "Per-room Name"
	empty := ""
	tests := []struct {
		name    string
		current *event.MemberEventContent
		member  ChatMember
		want    string
	}{
		{
			name:    "nickname overrides what the room has",
			current: &event.MemberEventContent{Displayname: "Global Name"},
			member:  ChatMember{Nickname: &nickname},
			want:    "Per-room Name",
		},
		{
			name:    "no nickname keeps what the room has",
			current: &event.MemberEventContent{Displayname: "Global Name"},
			member:  ChatMember{},
			want:    "Global Name",
		},
		{
			name:    "an empty nickname is a value, not an absence",
			current: &event.MemberEventContent{Displayname: "Global Name"},
			member:  ChatMember{Nickname: &empty},
			want:    "",
		},
		{
			name:    "no current member event and no nickname",
			current: nil,
			member:  ChatMember{},
			want:    "",
		},
		{
			name:    "no current member event but a nickname",
			current: nil,
			member:  ChatMember{Nickname: &nickname},
			want:    "Per-room Name",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := memberDisplayname(test.current, test.member); got != test.want {
				t.Errorf("memberDisplayname() = %q, want %q", got, test.want)
			}
		})
	}
}

// syncParticipants skips a member whose membership is unchanged. A nickname change alone
// leaves membership untouched, so without the displayname in that condition it would
// never be sent.
func TestMemberSyncIsNotSkippedForANicknameChange(t *testing.T) {
	nickname := "Per-room Name"
	current := &event.MemberEventContent{
		Membership:  event.MembershipJoin,
		Displayname: "Global Name",
	}
	member := ChatMember{Membership: event.MembershipJoin, Nickname: &nickname}

	membershipUnchanged := current.Membership == member.Membership
	displaynameUnchanged := current.Displayname == memberDisplayname(current, member)
	if !membershipUnchanged {
		t.Fatal("test setup: membership should be unchanged")
	}
	if displaynameUnchanged {
		t.Fatal("skip condition still holds; the nickname change would never be sent")
	}
}
