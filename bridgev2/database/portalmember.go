// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package database

import (
	"context"

	"go.mau.fi/util/dbutil"

	"maunium.net/go/mautrix/bridgev2/networkid"
)

type PortalMemberQuery struct {
	BridgeID networkid.BridgeID
	*dbutil.QueryHelper[*PortalMember]
}

// PortalMember is the bridge's own per-room state for a ghost in a portal. The nickname is
// the displayname the ghost's member event carries in that room, which the homeserver
// overwrites with the global profile whenever that changes.
type PortalMember struct {
	BridgeID networkid.BridgeID
	Portal   networkid.PortalKey
	GhostID  networkid.UserID
	Nickname string
}

const (
	getPortalMemberBaseQuery = `
		SELECT bridge_id, portal_id, portal_receiver, ghost_id, nickname
		FROM portal_member
	`
	getAllPortalMembersInPortalQuery = getPortalMemberBaseQuery + `
		WHERE bridge_id=$1 AND portal_id=$2 AND portal_receiver=$3
	`
	// Served by portal_member_ghost_idx. This is the only way to find the rooms a ghost is
	// in, so it is what makes re-applying nicknames after a profile change possible at all.
	getAllPortalMembersForGhostQuery = getPortalMemberBaseQuery + `
		WHERE bridge_id=$1 AND ghost_id=$2
	`
	upsertPortalMemberQuery = `
		INSERT INTO portal_member (bridge_id, portal_id, portal_receiver, ghost_id, nickname)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (bridge_id, portal_id, portal_receiver, ghost_id) DO UPDATE
			SET nickname=excluded.nickname
	`
	deletePortalMemberQuery = `
		DELETE FROM portal_member WHERE bridge_id=$1 AND portal_id=$2 AND portal_receiver=$3 AND ghost_id=$4
	`
)

func (pmq *PortalMemberQuery) GetAllInPortal(ctx context.Context, portal networkid.PortalKey) ([]*PortalMember, error) {
	return pmq.QueryMany(ctx, getAllPortalMembersInPortalQuery, pmq.BridgeID, portal.ID, portal.Receiver)
}

func (pmq *PortalMemberQuery) GetAllForGhost(ctx context.Context, ghostID networkid.UserID) ([]*PortalMember, error) {
	return pmq.QueryMany(ctx, getAllPortalMembersForGhostQuery, pmq.BridgeID, ghostID)
}

func (pmq *PortalMemberQuery) Put(ctx context.Context, pm *PortalMember) error {
	ensureBridgeIDMatches(&pm.BridgeID, pmq.BridgeID)
	return pmq.Exec(ctx, upsertPortalMemberQuery, pm.sqlVariables()...)
}

func (pmq *PortalMemberQuery) Delete(ctx context.Context, portal networkid.PortalKey, ghostID networkid.UserID) error {
	return pmq.Exec(ctx, deletePortalMemberQuery, pmq.BridgeID, portal.ID, portal.Receiver, ghostID)
}

func (pm *PortalMember) Scan(row dbutil.Scannable) (*PortalMember, error) {
	err := row.Scan(&pm.BridgeID, &pm.Portal.ID, &pm.Portal.Receiver, &pm.GhostID, &pm.Nickname)
	if err != nil {
		return nil, err
	}
	return pm, nil
}

func (pm *PortalMember) sqlVariables() []any {
	return []any{pm.BridgeID, pm.Portal.ID, pm.Portal.Receiver, pm.GhostID, pm.Nickname}
}
