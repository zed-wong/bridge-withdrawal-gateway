package snapshots

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"time"

	"github.com/fox-one/mixin-sdk-go"
	"github.com/shopspring/decimal"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type SnapshotsWorker struct {
	client *mixin.Client
	db     *gorm.DB
	pin    string
}

func NewSnapshotsWorker(ctx context.Context, store *mixin.Keystore, dsn, pin string) *SnapshotsWorker {
	client, err := mixin.NewFromKeystore(store)
	if err != nil {
		panic(err)
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	sw := &SnapshotsWorker{
		client: client,
		db:     db,
		pin:    pin,
	}
	return sw
}

func (sw *SnapshotsWorker) Loop(ctx context.Context) {
	for {
		sw.MonitorSnapshots(ctx)
		time.Sleep(3 * time.Second)
	}
}

func (sw *SnapshotsWorker) MonitorSnapshots(ctx context.Context) {
	snaps, err := sw.client.ReadSnapshotsWithOptions(ctx, time.Now(), 10, mixin.ReadSnapshotsOptions{
		Order:         "",
		AssetID:       "",
		OpponentID:    "",
		DestinationID: "",
		Tag:           "",
	})
	if err != nil {
		log.Println("sw.client.ReadSnapshotsWithOptions() => ", err)
	}

	for i, s := range snaps {
		state, txmemo := getMemo(s.Memo)
		if !state {
			log.Printf("[%d] Not withdrawal memo", i)
			continue
		}
		if sw.checkSnapshotExist(s.SnapshotID) {
			continue
		}
		if !checkTxMemo(txmemo) {
			log.Println("[Error] TxMemo invalid")
			continue
		}

		//   Initialize a swap base on payload (Set min receive)
		Swap(sw.client, ctx, sw.client.ClientID, txmemo.AssetID, txmemo.FeeAssetID, txmemo.SwapAmount, decimal.RequireFromString(txmemo.FeeAmount), sw.pin)

		//   Initialize a withdrawal
		Amount, err := decimal.NewFromString(txmemo.Amount)
		if err != nil {
			log.Println(err)
			continue
		}
		input := &mixin.WithdrawInput{
			AddressID: txmemo.AssetID,
			Amount:    Amount,
			TraceID:   txmemo.TraceID,
			Memo:      "Withdraw",
		}
		tx, err := sw.client.Withdraw(ctx, *input, sw.pin)
		if err != nil {
			log.Println(err)
			continue
		}
		log.Println("Withdraw tx:", tx)
		// write db
	}
}

func (sw *SnapshotsWorker) checkSnapshotExist(snapshotID string) bool {
	var exist bool
	err := sw.db.Model(&InputSnapshots{}).Select("count(*) > 0").Where("snapshot_id = ?", snapshotID).Find(&exist).Error
	if err != nil {
		log.Println("checkSnapshotExist() => ", err)
	}
	return exist
}

func getMemo(UTXOmemo string) (bool, *TxMemo) {
	if len(UTXOmemo) == 0 {
		return false, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(UTXOmemo)
	if err != nil {
		return false, nil
	}
	var txmemo *TxMemo
	err = json.Unmarshal(decoded, txmemo)
	if err != nil {
		return false, nil
	}
	return true, txmemo
}

func checkTxMemo(memo *TxMemo) bool {
	if len(memo.ToAddress) == 0 || len(memo.ToAddress) > 100 {
		return false
	}
	_, err := decimal.NewFromString(memo.Amount)
	if err != nil {
		return false
	}
	_, err = decimal.NewFromString(memo.SwapAmount)
	if err != nil {
		return false
	}
	_, err = decimal.NewFromString(memo.FeeAmount)
	if err != nil {
		return false
	}
	if len(memo.AssetID) != 36 {
		return false
	}
	if len(memo.TraceID) != 36 {
		return false
	}
	return true
}
