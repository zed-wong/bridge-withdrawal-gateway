package snapshots

import (
	"context"
	"errors"
	"log"
	"time"

	fswap "github.com/fox-one/4swap-sdk-go"
	mtg "github.com/fox-one/4swap-sdk-go/mtg"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/shopspring/decimal"
)

var (
	group, _ = fswap.ReadGroup(context.Background())
)

func (sw *SnapshotsWorker) Swap(client *mixin.Client, ctx context.Context, receiverID, fromAssetID, toAssetID, followID string, swapAmount, min decimal.Decimal) string {
	action := mtg.SwapAction(
		receiverID,
		followID,
		toAssetID,
		"",
		min,
	)
	memo, err := action.Encode(group.PublicKey)
	if err != nil {
		log.Println("Swap.Encode() =>", err)
	}
	tx, err := client.Transaction(ctx, &mixin.TransferInput{
		AssetID: fromAssetID,
		Amount:  swapAmount,
		TraceID: mixin.RandomTraceID(),
		Memo:    memo,
		OpponentMultisig: struct {
			Receivers []string `json:"receivers,omitempty"`
			Threshold uint8    `json:"threshold,omitempty"`
		}{
			Receivers: group.Members,
			Threshold: uint8(group.Threshold),
		},
	}, sw.pin)
	if err != nil {
		log.Println("Swap.Transaction() => ", err)
		return ""
	}
	log.Println("Swap tx:", tx)
	sw.WriteSwap(tx.SnapshotID, followID, time.Now().String())
	return followID
}

func PreOrder(ctx context.Context, payAssetID, fillAssetID string, payAmount decimal.Decimal) (*fswap.Order, error) {
	return fswap.PreOrder(ctx, &fswap.PreOrderReq{
		PayAssetID:  payAssetID,
		FillAssetID: fillAssetID,
		PayAmount:   payAmount,
	})
}

func readOrder(ctx context.Context, token, followID string) (int, error) {
	if len(followID) == 0 {
		return 0, errors.New("Swap Failed")
	}
	ctx = fswap.WithToken(ctx, token)
	order, err := fswap.ReadOrder(ctx, followID)
	if err != nil {
		return 0, err
	}
	log.Println("Swap state:", order.State)
	switch order.State {
	case "Trading":
		return 1, nil
	case "Rejected":
		return 0, errors.New("Swap Rejected")
	case "Done":
		return 2, nil
	}
	return 3, errors.New("unknown case")
}

func WaitForSwap(ctx context.Context, token, followID string) bool {
	for {
		code, err := readOrder(ctx, token, followID)
		if err != nil {
			return false
		}

		switch code {
		case 1:
			continue
		case 2:
			return true
		case 0, 3:
			return false
		}
		time.Sleep(1 * time.Second)
	}
}
