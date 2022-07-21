package snapshots

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/fox-one/mixin-sdk-go"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type SnapshotsWorker struct {
	client *mixin.Client
	ka     *mixin.KeystoreAuth
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
	ka, err := mixin.AuthFromKeystore(store)
	if err != nil {
		panic(err)
	}
	sw := &SnapshotsWorker{
		client: client,
		ka:     ka,
		db:     db,
		pin:    pin,
	}
	return sw
}

func (sw *SnapshotsWorker) Loop(ctx context.Context) {
	for {
		sw.MonitorSnapshots(ctx)
		time.Sleep(5 * time.Second)
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
	token := sw.ka.SignToken(mixin.SignRaw("GET", "/me", nil), uuid.Must(uuid.NewV4()).String(), 60*time.Minute)

	for i, s := range snaps {
		state, txmemo := getMemo(s.Memo)
		if !state {
			log.Printf("[%d] Not withdrawal memo", i)
			continue
		}
		log.Println("Is withdrawal")
		if !checkTxMemo(txmemo) {
			log.Println("[Error] TxMemo invalid")
			continue
		}
		log.Println("Memo valid")
		if sw.checkSnapshotExist(s.SnapshotID) {
			log.Println("Snapshot exist")
			continue
		}
		//   Write Snapshot DB
		log.Println("Snapshot doesn't exist")
		fmt.Printf("%d: %+v\n\n", i, s)
		sw.WriteInputSnapshot(s)

		feeAsset, err := sw.client.ReadAsset(ctx, s.AssetID)
		if err != nil {
			log.Println("ReadAsset() => ", err)
			continue
		}
		feeAmount, err := sw.client.ReadAssetFee(ctx, s.AssetID)
		if err != nil {
			log.Println("ReadAssetFee() => ", err)
			continue
		}
		memoAmount, err := decimal.NewFromString(txmemo.Amount)
		if err != nil {
			log.Println("NewFromString() => ", err)
			continue
		}

		swapAmount := s.Amount.Sub(memoAmount)
		if swapAmount.IsNegative() {
			log.Println("Withdrawal amount is negative.")
			continue
		}
		if s.AssetID == feeAsset.AssetID {
			if swapAmount.LessThan(feeAmount) {
				//refund
				log.Println("Refund")
				continue
			}
		} else {
			order, err := PreOrder(ctx, s.AssetID, feeAsset.AssetID, swapAmount)
			if err != nil {
				//refund
				log.Println("PreOrder() =>", err)
				continue
			}
			if order.FillAmount.Sub(feeAmount).IsNegative() {
				//refund
				log.Println("Fee amount is not enought")
				continue
			}
		}
		//   s.OpponentID
		if s.AssetID != feeAsset.AssetID {
			followID := sw.Swap(sw.client, ctx, sw.client.ClientID, s.AssetID, feeAsset.AssetID, mixin.RandomTraceID(), swapAmount, feeAmount)
			sw.WriteSwap(s.OpponentID, followID, time.Now().String())
			if !WaitForSwap(ctx, token, followID) {
				log.Println("Swap failed, should retry")
			}
		}

		//   Withdrawal
		Address, err := sw.client.CreateAddress(ctx, mixin.CreateAddressInput{
			AssetID:     s.AssetID,
			Destination: txmemo.ToAddress,
			Tag:         txmemo.Memo,
			Label:       "1",
		}, sw.pin)
		if err != nil {
			log.Println("sw.client.CreateAddress() => ", err)
			continue
		}
		Amount, err := decimal.NewFromString(txmemo.Amount)
		if err != nil {
			log.Println(err)
			continue
		}
		input := &mixin.WithdrawInput{
			AddressID: Address.AddressID,
			Amount:    Amount,
			TraceID:   uuid.Must(uuid.NewV4()).String(),
			Memo:      txmemo.Memo,
		}
		tx, err := sw.client.Withdraw(ctx, *input, sw.pin)
		if err != nil {
			log.Println(err)
			continue
		}
		sw.WriteOutputSnapshot(tx, s.SnapshotID, txmemo.ToAddress)
	}
	fmt.Printf("\n\n\n")
}

func (sw *SnapshotsWorker) checkSnapshotExist(snapshotID string) bool {
	var exist bool
	err := sw.db.Model(&InputSnapshot{}).Select("count(*) > 0").Where("snapshot_id = ?", snapshotID).Find(&exist).Error
	if err != nil {
		log.Println("checkSnapshotExist() => ", err)
	}
	return exist
}

func getMemo(UTXOmemo string) (bool, *TxMemo) {
	if len(UTXOmemo) == 0 {
		// log.Println("UTXOmemo == 0")
		return false, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(UTXOmemo)
	if err != nil {
		// log.Println("base64.StdEncoding.DecodeString(UTXOmemo) => ", err)
		return false, nil
	}
	var txmemo TxMemo
	err = json.Unmarshal(decoded, &txmemo)
	if err != nil {
		// log.Println("json.Unmarshal(decoded, txmemo) => ", err)
		return false, nil
	}
	return true, &txmemo
}

func checkTxMemo(memo *TxMemo) bool {
	if len(memo.ToAddress) == 0 || len(memo.ToAddress) > 100 {
		return false
	}
	_, err := decimal.NewFromString(memo.Amount)
	return err == nil
}
