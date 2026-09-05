-- v31 (compatible with v9+): Re-send DM room names left stale by unsaved portal updates
UPDATE portal SET name_set=false WHERE room_type='dm' AND NOT name_is_custom AND name_set;
