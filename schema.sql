-- Store all snapshots
CREATE TABLE inputSnapshots(
	snapshot_id VARCHAR(36) NOT NULL UNIQUE,
	trace_id VARCHAR(36) NOT NULL UNIQUE,
	opponent_id VARCHAR(36),
	identity_id VARCHAR(36),
	created_at VARCHAR(60),
	memo VARCHAR(256)
);

-- CREATE TABLE withdrawal(
-- 	address_id VARCHAR(128) NOT NULL UNIQUE,
-- 	amount VARCHAR(16),
-- 	asset_id VARCHAR(36) NOT NULL,
-- 	fee_asset_id VARCHAR(36),
-- 	fee_amount VARCHAR(36),
-- 	pre_order_id VARCHAR(36),
-- 	trace_id VARCHAR(36) NOT NULL UNIQUE,
-- );

-- Store withdrawal snapshot id
CREATE TABLE withdrawalSnapshots (
	snapshot_id VARCHAR(36) NOT NULL
)

-- Store all swaps
CREATE TABLE swaps(
	snapshot_id VARCHAR(36) NOT NULL
)