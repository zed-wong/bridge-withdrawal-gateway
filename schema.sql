-- Store all snapshots
CREATE TABLE input_snapshots(
	snapshot_id VARCHAR(36) NOT NULL UNIQUE,
	trace_id VARCHAR(36) NOT NULL UNIQUE,
	opponent_id VARCHAR(36),
	created_at VARCHAR(48),
	memo VARCHAR(256)
);

CREATE TABLE swap_orders(
	order_state VARCHAR(36),
	follow_id VARCHAR(36),
	created_at VARCHAR(48),
	receiver_id VARCHAR(36)
);

CREATE TABLE output_snapshots(
	input_sn_id VARCHAR(36) NOT NULL UNIQUE,
	snapshot_id VARCHAR(36) NOT NULL UNIQUE,
	trace_id VARCHAR(36) NOT NULL UNIQUE,
	to_address VARCHAR(256),
	memo VARCHAR(256),
	asset_id VARCHAR(36),
	amount VARCHAR(36),
	created_at VARCHAR(48),
);