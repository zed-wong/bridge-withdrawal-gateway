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
	group := GetMtgGroup(ctx)

	for _, s := range snaps {
		state, txmemo := getMemo(s.Memo)
		if !state {
			// log.Printf("[%d] Not withdrawal memo", i)
			continue
		}
		if !checkTxMemo(txmemo) {
			continue
		}
		if sw.checkSnapshotExist(s.SnapshotID) {
			continue
		}
		sw.WriteInputSnapshot(s)

		log.Println("Valid")
		fmt.Printf("%+v\n\n", s)

		Asset, err := sw.client.ReadAsset(ctx, s.AssetID)
		if err != nil {
			log.Println("ReadAsset(s.AssetID) => ", err)
			continue
		}
		feeAsset, err := sw.client.ReadAsset(ctx, Asset.ChainID)
		if err != nil {
			log.Println("ReadAsset(Asset.ChainID) => ", err)
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

		if s.AssetID == feeAsset.AssetID {
			leftAmount := s.Amount.Sub(memoAmount)
			if leftAmount.LessThan(feeAmount) {
				err = sw.refund(ctx, s.AssetID, s.OpponentID, s.Amount, sw.pin)
				if err != nil {
					log.Println("leftAmount.LessThan(feeAmount), refund() =>", err)
				}
				continue
			}
		}
		log.Println("Basic fee check passed")
		if s.AssetID != feeAsset.AssetID {
			swapAmount := s.Amount.Sub(memoAmount)
			order, err := PreOrder(ctx, s.AssetID, feeAsset.AssetID, swapAmount)
			if err != nil {
				err = sw.refund(ctx, s.AssetID, s.OpponentID, s.Amount, sw.pin)
				if err != nil {
					log.Println("PreOrder error, refund() =>", err)
				}
				continue
			}
			if order.FillAmount.Sub(feeAmount).IsNegative() {
				err = sw.refund(ctx, s.AssetID, s.OpponentID, s.Amount, sw.pin)
				if err != nil {
					log.Println("PreOrder feeAmount not enough, refund() =>", err)
				}
				continue
			}
			log.Println("Swap fee check passed")

			followID := mixin.RandomTraceID()
			log.Println("FollowID:", followID)

			if err = sw.Swap(group, ctx, sw.client.ClientID, s.AssetID, feeAsset.AssetID, followID, "", swapAmount, feeAmount); err != nil {
				log.Println("sw.Swap() => ", err)
				continue
			}

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
			sw.WriteSwap(&SwapOrder{
				FollowID:   followID,
				CreatedAt:  time.Now().Format(time.RFC3339),
				OrderState: "Init",
				OpponentID: s.OpponentID,
				InputSnID:  s.SnapshotID,
				AddressID:  Address.AddressID,
				ToAddress:  txmemo.ToAddress,
				ToMemo:     txmemo.Memo,
				Amount:     txmemo.Amount,
				Withdrawn:  false,
			})
			return
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

		err = sw.withdrwal(ctx, Address.AddressID, s.SnapshotID, txmemo.ToAddress, txmemo.Memo, txmemo.Amount)
		if err != nil {
			log.Println("sw.withdrawal() => ", err)
		}
	}
}

func (sw *SnapshotsWorker) withdrwal(ctx context.Context, addressID, inputSnapshotID, toAddress, toMemo, amount string) error {
	Amount, err := decimal.NewFromString(amount)
	if err != nil {
		return err
	}
	input := &mixin.WithdrawInput{
		AddressID: addressID,
		Amount:    Amount,
		TraceID:   uuid.Must(uuid.NewV4()).String(),
		Memo:      toMemo,
	}
	tx, err := sw.client.Withdraw(ctx, *input, sw.pin)
	if err != nil {
		return err
	}
	sw.WriteOutputSnapshot(tx, inputSnapshotID, toAddress)
	log.Printf("Withdrawal success: %+v", tx)
	return nil
}

func (sw *SnapshotsWorker) checkSnapshotExist(snapshotID string) bool {
	var exist bool
	err := sw.db.Model(&InputSnapshot{}).Select("count(*) > 0").Where("snapshot_id = ?", snapshotID).Find(&exist).Error
	if err != nil {
		log.Println("checkSnapshotExist() => ", err)
	}
	return exist
}

func (sw *SnapshotsWorker) refund(ctx context.Context, assetID, receiverID string, amount decimal.Decimal, pin string) error {
	payload := base64.StdEncoding.EncodeToString([]byte("Refund from Withdrawal Gateway"))
	input := &mixin.TransferInput{
		AssetID:    assetID,
		OpponentID: receiverID,
		Amount:     amount,
		TraceID:    uuid.Must(uuid.NewV4()).String(),
		Memo:       payload,
	}
	if _, err := sw.client.Transfer(ctx, input, pin); err != nil {
		return err
	}
	return nil
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
