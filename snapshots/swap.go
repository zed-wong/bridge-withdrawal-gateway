package snapshots

import (
	"context"
	"errors"
	"log"
	"time"

	fswap "github.com/fox-one/4swap-sdk-go"
	mtg "github.com/fox-one/4swap-sdk-go/mtg"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
)

func (sw *SnapshotsWorker) Swap(group *fswap.Group, ctx context.Context, receiverID, fromAssetID, toAssetID, followID, routes string, swapAmount, min decimal.Decimal) error {
	action := mtg.SwapAction(
		receiverID,
		followID,
		toAssetID,
		routes,
		min,
	)
	memo, err := action.Encode(group.PublicKey)
	if err != nil {
		log.Println("Swap.Encode() =>", err)
		return err
	}
	tx, err := sw.client.Transaction(ctx, &mixin.TransferInput{
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
		return err
	}
	log.Printf("Swap tx: %+v \n", tx)
	return nil
}

func (sw *SnapshotsWorker) LoopSwap(ctx context.Context) {
	period := 5 * time.Second
	for {
		var order []SwapOrder
		sw.db.Where(&SwapOrder{Withdrawn: false}).Find(&order)
		if len(order) == 0 {
			time.Sleep(period)
			continue
		}
		token := sw.ka.SignToken(mixin.SignRaw("GET", "/me", nil), uuid.Must(uuid.NewV4()).String(), 60*time.Minute)
		for _, o := range order {
			if !o.Withdrawn {
				new, err := ReadOrder(ctx, token, o.FollowID)
				log.Printf("new: %+v", new)
				//log.Printf("o: %+v", o)
				if err != nil {
					log.Println("ReadOrder() => ", err)
					continue
				}
				sw.UpdateSwap(&SwapOrder{OrderState: new.State}, o.FollowID)

				if new.State == "Done" {
					time.Sleep(5 * time.Second)
					err := sw.withdrawal(ctx, o.AddressID, o.InputSnID, o.ToAddress, o.ToMemo, o.Amount)
					if err != nil {
						log.Println("LoopSwap.withdrawal() => ", err)
						return
					}
					sw.UpdateSwap(&SwapOrder{Withdrawn: true}, o.FollowID)
				}
			}
		}

		time.Sleep(period)
	}
}

func PreOrder(ctx context.Context, payAssetID, fillAssetID string, payAmount decimal.Decimal) (*fswap.Order, error) {
	return fswap.PreOrder(ctx, &fswap.PreOrderReq{
		PayAssetID:  payAssetID,
		FillAssetID: fillAssetID,
		PayAmount:   payAmount,
	})
}

func ReadOrder(ctx context.Context, token, followID string) (*fswap.Order, error) {
	fswap.UseEndpoint(fswap.MtgEndpoint)
	if len(followID) == 0 {
		return nil, errors.New("Swap Failed")
	}
	ctx = fswap.WithToken(ctx, token)
	order, err := fswap.ReadOrder(ctx, followID)
	if err != nil {
		return nil, err
	}
	return order, err
}

func GetMtgGroup(ctx context.Context) *fswap.Group {
	fswap.UseEndpoint(fswap.MtgEndpoint)
	group, _ := fswap.ReadGroup(ctx)
	return group
}
