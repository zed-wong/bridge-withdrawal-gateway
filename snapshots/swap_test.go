package snapshots_test

import (
	"context"
	"testing"
	"time"

	"main/snapshots"

	fswap "github.com/fox-one/4swap-sdk-go"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
)

func TestSwap(t *testing.T) {
	ctx := context.Background()
	store := &mixin.Keystore{}
	dsn := ""
	pin := ""
	client, err := mixin.NewFromKeystore(store)
	if err != nil {
		panic(err)
	}
	// ka, err := mixin.AuthFromKeystore(store)
	// if err != nil {
	// 	panic(err)
	// }
	// token := ka.SignToken(mixin.SignRaw("GET", "/me", nil), uuid.Must(uuid.NewV4()).String(), 60*time.Minute)
	group := snapshots.GetMtgGroup(ctx)
	sw := snapshots.NewSnapshotsWorker(ctx, store, dsn, pin)

	sender := "3bb60b8a-e7a6-3402-8d63-ed74c259e961"
	fromAssetID := "4d8c508b-91c5-375b-92b0-ee702ed2dac5"
	toAssetID := "43d61dcd-e413-450d-80b8-101d5e903357"
	payAmount, _ := decimal.NewFromString("0.124")
	minAmount, _ := decimal.NewFromString("0.000075")
	followID := mixin.RandomTraceID()

	preOrder, err := snapshots.PreOrder(ctx, fromAssetID, toAssetID, payAmount)
	if err != nil {
		t.Log("preOrder() => ", err)
	}
	t.Logf("PreOrder:%+v ", preOrder)
	t.Log("Time:", time.Now().Format(time.RFC3339))
	err = sw.Swap(group, ctx, client.ClientID, fromAssetID, toAssetID, followID, "", payAmount, minAmount)
	if err != nil {
		t.Log("Swap Error:", err)
	}
	t.Log("FollowID:", followID)
	sw.WriteSwap(sender, followID, time.Now().Format(time.RFC3339), "")
}

func TestReadSwap(t *testing.T) {
	ctx := context.Background()
	fswap.UseEndpoint(fswap.MtgEndpoint)
	store := &mixin.Keystore{}
	ka, err := mixin.AuthFromKeystore(store)
	if err != nil {
		panic(err)
	}
	token := ka.SignToken(mixin.SignRaw("GET", "/me", nil), uuid.Must(uuid.NewV4()).String(), 60*time.Minute)
	ctx = fswap.WithToken(ctx, token)
	order, err := fswap.ReadOrder(ctx, "97e4d103-dd22-44f3-b6b7-d47d71be0446")
	if err != nil {
		t.Log(err)
	}
	t.Logf("Order: %+v", order)
}
