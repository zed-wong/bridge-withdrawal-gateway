package snapshots

import (
	"context"
	"log"

	fswap "github.com/fox-one/4swap-sdk-go"
	mtg "github.com/fox-one/4swap-sdk-go/mtg"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
)

var (
	group, _ = fswap.ReadGroup(context.Background())
)

func Swap(client *mixin.Client, ctx context.Context, receiverID, fromAssetID, toAssetID, swapAmount string, min decimal.Decimal, pin string) {
	followID, _ := uuid.NewV4()
	action := mtg.SwapAction(
		receiverID,
		followID.String(),
		toAssetID,
		"",
		min,
	)
	memo, err := action.Encode(group.PublicKey)
	if err != nil {
		log.Println("Swap() =>", err)
	}
	tx, err := client.Transaction(ctx, &mixin.TransferInput{
		AssetID: fromAssetID,
		Amount:  decimal.RequireFromString(swapAmount),
		TraceID: mixin.RandomTraceID(),
		Memo:    memo,
		OpponentMultisig: struct {
			Receivers []string `json:"receivers,omitempty"`
			Threshold uint8    `json:"threshold,omitempty"`
		}{
			Receivers: group.Members,
			Threshold: uint8(group.Threshold),
		},
	}, pin)
	if err != nil {
		log.Println("Swap.Transaction() => ", err)
		return
	}
	log.Println("Swap tx:", tx)
	// write db
}
