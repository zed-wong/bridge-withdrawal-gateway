package snapshots

type InputSnapshots struct {
	SnapshotID string `json:"snapshot_id"`
	TraceID    string `json:"trace_id"`
	OpponentID string `json:"opponent_id"`
	IdentityID string `json:"identity_id"`
	CreatedAt  string `json:"created_at"`
	Memo       string `json:"memo"`
}

type TxMemo struct {
	ToAddress  string `json:"to_address"`
	Amount     string `json:"amount"`
	AssetID    string `json:"asset_id"`
	SwapAmount string `json:"swap_amount"`
	FeeAssetID string `json:"fee_asset_id"`
	FeeAmount  string `json:"fee_amount"`
	TraceID    string `json:"trace_id"`
}
