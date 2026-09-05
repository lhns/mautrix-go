// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package bridgev2

import (
	"context"
	"testing"
	"time"

	"maunium.net/go/mautrix/bridgev2/bridgeconfig"
	"maunium.net/go/mautrix/bridgev2/database"
	"maunium.net/go/mautrix/bridgev2/networkid"
	"maunium.net/go/mautrix/id"
)

// dmPortal builds the minimum Portal that UpdateInfoFromGhost will act on. MXID is empty
// so sendRoomMeta returns before touching Matrix, which is what lets these run without a
// bridge harness.
func dmPortal(name string, nameSet bool) *Portal {
	return &Portal{
		Portal: &database.Portal{
			PortalKey:   networkid.PortalKey{ID: "chat", Receiver: "login"},
			Name:        name,
			NameSet:     nameSet,
			OtherUserID: "otheruser",
			RoomType:    database.RoomTypeDM,
		},
		Bridge: &Bridge{
			Config: &bridgeconfig.BridgeConfig{PrivateChatPortalMeta: true},
		},
	}
}

// UpdateInfoFromGhost mutates portal.Name and reports changed=true, and
// lockedUpdateInfoFromGhost -- the caller a ghost rename reaches via
// Ghost.updateDMPortals -- persists that, so the row keeps up with the room.
func TestUpdateInfoFromGhostPersistsPortalRename(t *testing.T) {
	ctx := context.Background()
	portal := dmPortal("Old Name", true)
	ghost := &Ghost{Ghost: &database.Ghost{ID: "otheruser", Name: "New Name"}}

	if changed := portal.UpdateInfoFromGhost(ctx, ghost); !changed {
		t.Fatal("UpdateInfoFromGhost reported no change; expected it to report the rename")
	}
	if portal.Name != "New Name" {
		t.Fatalf("portal.Name = %q, want the ghost name", portal.Name)
	}

	// The same rename through the ghost path reaches UpdateBridgeInfo and Save, which
	// need a bridge this harness does not have -- reaching them at all is the assertion.
	portal = dmPortal("Old Name", true)
	if !reachesBridgeDependency(func() { portal.lockedUpdateInfoFromGhost(ctx, ghost) }) {
		t.Fatal("lockedUpdateInfoFromGhost did not act on the rename")
	}
}

// updateName trusts the row: Name plus NameSet is taken to describe what the room shows.
// Once a dropped write has left them stale, a later correction to the recorded name is
// skipped and the room keeps the wrong name indefinitely.
func TestUpdateNameSkipsWhenRowClaimsTheNameIsAlreadySet(t *testing.T) {
	ctx := context.Background()

	// The row as a dropped write leaves it: it records a name the room no longer shows.
	portal := dmPortal("Old Name", true)
	portal.MXID = id.RoomID("!room:example.org")

	if changed := portal.updateName(ctx, "Old Name", nil, time.Time{}, false); changed {
		t.Fatal("updateName reported a change; expected the stale row to short-circuit it")
	}

	// Clearing NameSet is what recovers such a room. The same call then reaches
	// sendRoomMeta, which needs a Matrix connector this harness does not have --
	// reaching it at all is the assertion.
	portal.NameSet = false
	if !reachesBridgeDependency(func() { portal.updateName(ctx, "Old Name", nil, time.Time{}, false) }) {
		t.Fatal("updateName still short-circuited after NameSet was cleared")
	}
}

// reachesBridgeDependency reports whether fn got as far as a bridge dependency this
// harness does not have, which panics on the nil field. bridgev2 ships no test harness.
func reachesBridgeDependency(fn func()) (reached bool) {
	defer func() {
		if r := recover(); r != nil {
			reached = true
		}
	}()
	fn()
	return false
}
