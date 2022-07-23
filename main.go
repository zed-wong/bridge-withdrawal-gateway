package main

import (
	"context"
	"fmt"

	"main/message"
	"main/snapshots"

	"github.com/fox-one/mixin-sdk-go"
	"github.com/spf13/viper"
)

func main() {
	ctx := context.Background()
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	store := &mixin.Keystore{
		ClientID:   viper.GetString("bot.clientID"),
		SessionID:  viper.GetString("bot.sessionID"),
		PrivateKey: viper.GetString("bot.privateKey"),
		PinToken:   viper.GetString("bot.pinToken"),
	}
	dsn := viper.GetString("db.dsn")

	messsageWorker := message.NewMessageWorker(ctx, store, dsn)
	go messsageWorker.Loop(ctx)

	snapshotWorker := snapshots.NewSnapshotsWorker(ctx, store, dsn, viper.GetString("bot.pin"))
	go snapshotWorker.Loop(ctx)
	snapshotWorker.LoopSwap(ctx)
}
