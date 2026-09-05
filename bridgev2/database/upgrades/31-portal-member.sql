-- v31 (compatible with v9+): Add per-room member nicknames
CREATE TABLE portal_member (
	bridge_id       TEXT NOT NULL,
	portal_id       TEXT NOT NULL,
	portal_receiver TEXT NOT NULL,
	ghost_id        TEXT NOT NULL,
	nickname        TEXT NOT NULL,

	PRIMARY KEY (bridge_id, portal_id, portal_receiver, ghost_id),
	CONSTRAINT portal_member_portal_fkey FOREIGN KEY (bridge_id, portal_id, portal_receiver)
		REFERENCES portal (bridge_id, id, receiver)
		ON DELETE CASCADE ON UPDATE CASCADE,
	CONSTRAINT portal_member_ghost_fkey FOREIGN KEY (bridge_id, ghost_id)
		REFERENCES ghost (bridge_id, id)
		ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE INDEX portal_member_ghost_idx ON portal_member (bridge_id, ghost_id);
