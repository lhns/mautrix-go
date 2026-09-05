// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package database

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mau.fi/util/dbutil"

	"maunium.net/go/mautrix/bridgev2/networkid"
)

const testBridgeID networkid.BridgeID = "testbridge"

// _foreign_keys=on is explicit so the cascade assertion actually means something: sqlite
// enforces foreign keys per connection and defaults them off.
func getTestDB(t *testing.T) *Database {
	t.Helper()
	rawDB, err := sql.Open("sqlite3", ":memory:?_busy_timeout=5000&_foreign_keys=on")
	require.NoError(t, err, "Error opening raw database")
	wrapped, err := dbutil.NewWithDB(rawDB, "sqlite3")
	require.NoError(t, err, "Error creating database wrapper")
	db := New(testBridgeID, MetaTypes{}, wrapped)
	require.NoError(t, db.Upgrade(context.TODO()), "Error upgrading database")
	return db
}

// The member rows reference a portal and a ghost, so the parents have to exist first.
func makePortal(t *testing.T, db *Database, key networkid.PortalKey) {
	t.Helper()
	require.NoError(t, db.Portal.Insert(context.TODO(), &Portal{
		BridgeID:  testBridgeID,
		PortalKey: key,
		RoomType:  RoomTypeDefault,
	}))
}

func makeGhost(t *testing.T, db *Database, id networkid.UserID) {
	t.Helper()
	require.NoError(t, db.Ghost.Insert(context.TODO(), &Ghost{
		BridgeID:    testBridgeID,
		ID:          id,
		Identifiers: []string{},
	}))
}

func TestPortalMemberPutAndGet(t *testing.T) {
	ctx := context.TODO()
	db := getTestDB(t)
	portal := networkid.PortalKey{ID: "group1"}
	makePortal(t, db, portal)
	makeGhost(t, db, "ghost1")

	require.NoError(t, db.PortalMember.Put(ctx, &PortalMember{
		Portal: portal, GhostID: "ghost1", Nickname: "First Name",
	}))

	members, err := db.PortalMember.GetAllInPortal(ctx, portal)
	require.NoError(t, err)
	require.Len(t, members, 1)
	assert.Equal(t, "First Name", members[0].Nickname)
	assert.Equal(t, networkid.UserID("ghost1"), members[0].GhostID)
	assert.Equal(t, testBridgeID, members[0].BridgeID)
}

func TestPortalMemberPutUpdatesInsteadOfDuplicating(t *testing.T) {
	ctx := context.TODO()
	db := getTestDB(t)
	portal := networkid.PortalKey{ID: "group1"}
	makePortal(t, db, portal)
	makeGhost(t, db, "ghost1")

	require.NoError(t, db.PortalMember.Put(ctx, &PortalMember{Portal: portal, GhostID: "ghost1", Nickname: "First Name"}))
	require.NoError(t, db.PortalMember.Put(ctx, &PortalMember{Portal: portal, GhostID: "ghost1", Nickname: "Second Name"}))

	members, err := db.PortalMember.GetAllInPortal(ctx, portal)
	require.NoError(t, err)
	require.Len(t, members, 1, "the upsert should replace the row, not add one")
	assert.Equal(t, "Second Name", members[0].Nickname)
}

// The reverse lookup is what makes re-applying nicknames after a profile change possible, so
// it has to find every room the ghost is in, not just one.
func TestPortalMemberGetAllForGhostSpansPortals(t *testing.T) {
	ctx := context.TODO()
	db := getTestDB(t)
	portalA := networkid.PortalKey{ID: "group1"}
	portalB := networkid.PortalKey{ID: "group2"}
	makePortal(t, db, portalA)
	makePortal(t, db, portalB)
	makeGhost(t, db, "ghost1")
	makeGhost(t, db, "ghost2")

	require.NoError(t, db.PortalMember.Put(ctx, &PortalMember{Portal: portalA, GhostID: "ghost1", Nickname: "In A"}))
	require.NoError(t, db.PortalMember.Put(ctx, &PortalMember{Portal: portalB, GhostID: "ghost1", Nickname: "In B"}))
	require.NoError(t, db.PortalMember.Put(ctx, &PortalMember{Portal: portalA, GhostID: "ghost2", Nickname: "Other Ghost"}))

	members, err := db.PortalMember.GetAllForGhost(ctx, "ghost1")
	require.NoError(t, err)
	require.Len(t, members, 2)
	byPortal := map[networkid.PortalID]string{}
	for _, member := range members {
		byPortal[member.Portal.ID] = member.Nickname
	}
	assert.Equal(t, map[networkid.PortalID]string{"group1": "In A", "group2": "In B"}, byPortal)
}

func TestPortalMemberDelete(t *testing.T) {
	ctx := context.TODO()
	db := getTestDB(t)
	portal := networkid.PortalKey{ID: "group1"}
	makePortal(t, db, portal)
	makeGhost(t, db, "ghost1")
	require.NoError(t, db.PortalMember.Put(ctx, &PortalMember{Portal: portal, GhostID: "ghost1", Nickname: "First Name"}))

	require.NoError(t, db.PortalMember.Delete(ctx, portal, "ghost1"))

	members, err := db.PortalMember.GetAllInPortal(ctx, portal)
	require.NoError(t, err)
	assert.Empty(t, members)
}

// Cascades are why the feature needs no cleanup code of its own.
func TestPortalMemberCascadesWithPortal(t *testing.T) {
	ctx := context.TODO()
	db := getTestDB(t)
	portal := networkid.PortalKey{ID: "group1"}
	makePortal(t, db, portal)
	makeGhost(t, db, "ghost1")
	require.NoError(t, db.PortalMember.Put(ctx, &PortalMember{Portal: portal, GhostID: "ghost1", Nickname: "First Name"}))

	require.NoError(t, db.Portal.Delete(ctx, portal))

	members, err := db.PortalMember.GetAllForGhost(ctx, "ghost1")
	require.NoError(t, err)
	assert.Empty(t, members, "deleting the portal should take its member rows with it")
}
