package snapshots

type InputSnapshot struct {
	SnapshotID string `json:"snapshot_id"`
	TraceID    string `json:"trace_id"`
	OpponentID string `json:"opponent_id"`
	CreatedAt  string `json:"created_at"`
	Memo       string `json:"memo"`
}

type TxMemo struct {
	ToAddress string `json:"t"`
	Memo      string `json:"m"`
	Amount    string `json:"a"`
}

type SwapOrder struct {
	FollowID   string `json:"follow_id"`
	CreatedAt  string `json:"created_at"`
	OrderState string `json:"order_state"`
	ReceiverID string `json:"receiver_id"`
}

type OutputSnapshot struct {
	InputSnID  string `json:"input_sn_id"`
	SnapshotID string `json:"snapshot_id"`
	TraceID    string `json:"trace_id"`
	ToAddress  string `json:"to_address"`
	CreatedAt  string `json:"created_at"`
	AssetID    string `json:"asset_id"`
	Amount     string `json:"amount"`
	Memo       string `json:"memo"`
}
