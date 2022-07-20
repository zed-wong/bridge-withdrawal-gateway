package message

import (
	"fmt"
	"log"
	"time"
	"context"
	"strings"
	"encoding/json"
	"encoding/base64"

	"gorm.io/gorm"
	"gorm.io/driver/postgres"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/fox-one/mixin-sdk-go"
)

const (
	USDT="4d8c508b-91c5-375b-92b0-ee702ed2dac5"
	pUSD="31d2ea9c-95eb-3355-b65b-ba096853bc18"
	CNB="965e5c6e-434c-3fa9-b780-c50f43cd955c"
	TIMEFORMAT="2006-01-02T15:04:05Z"
)

type MessageWorker struct{
	client	*mixin.Client
	db	*gorm.DB
}

func NewMessageWorker(ctx context.Context, store *mixin.Keystore, dsn string) *MessageWorker{
        client, err := mixin.NewFromKeystore(store)
        if err != nil {
                panic(err)
        }
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}
        rw := &MessageWorker{
                client: client,
		db: db,
        }
        return rw
}

func (rw *MessageWorker) handlePlainText(ctx context.Context, msg *mixin.MessageView, data []byte) {
	udata := strings.ToUpper(string(data))
	switch udata {
	case "HI", "HELLO", "HOLA", "HALLO",  "안녕하세요", "こんにちは":
		rw.respond(ctx, msg, mixin.MessageCategoryPlainText, []byte(""))
	case "你好":
		rw.respond(ctx, msg, mixin.MessageCategoryPlainText, []byte(""))
	default:
		log.Println(fmt.Sprintf("%s:%s",msg.UserID,string(data)))
	}
}

func (rw *MessageWorker) handleSnapshot(ctx context.Context, msg *mixin.MessageView, data []byte) {
	var t mixin.TransferInput
	json.Unmarshal(data, &t)
	return
}

func (rw *MessageWorker) handleData(ctx context.Context, msg *mixin.MessageView, data []byte) {
	var m mixin.DataMessage
	json.Unmarshal(data, &m)
	log.Printf("%+v", m)
}

func (rw *MessageWorker) handleAppCard(ctx context.Context, msg *mixin.MessageView, data []byte) {
	return
}

func (rw *MessageWorker) handleAudio(ctx context.Context, msg *mixin.MessageView, data []byte) {
	log.Println(string(data))
	return
}

func (rw *MessageWorker) respond(ctx context.Context, msg *mixin.MessageView, category string, data []byte) error{
        payload := base64.StdEncoding.EncodeToString(data)
        reply := &mixin.MessageRequest{
                ConversationID: msg.ConversationID,
                RecipientID:    msg.UserID,
                MessageID:      uuid.Must(uuid.NewV4()).String(),
                Category:       category,
                Data:           payload,
        }
        return rw.client.SendMessage(ctx, reply)
}
func (rw *MessageWorker) sendmsg(ctx context.Context, userID, conversationID, category string, data []byte) error{
        payload := base64.StdEncoding.EncodeToString(data)
        reply := &mixin.MessageRequest{
                ConversationID: conversationID,
                RecipientID:    userID,
                MessageID:      uuid.Must(uuid.NewV4()).String(),
                Category:       category,
                Data:           payload,
        }
        return rw.client.SendMessage(ctx, reply)
}
func (rw *MessageWorker) refund(ctx context.Context, msg *mixin.MessageView, view *mixin.TransferView, pin string) error {
	amount, err := decimal.NewFromString(view.Amount)
	if err != nil {
		return err
	}

	id, _ := uuid.FromString(msg.MessageID)

	input := &mixin.TransferInput{
		AssetID:    view.AssetID,
		OpponentID: msg.UserID,
		Amount:     amount,
		TraceID:    uuid.NewV5(id, "refund").String(),
		Memo:       "refund",
	}

	if _, err := rw.client.Transfer(ctx, input, pin); err != nil {
		return err
	}
	return nil
}

func (rw *MessageWorker) OnMessage(ctx context.Context) mixin.BlazeListenFunc{
        talk := func(ctx context.Context, msg *mixin.MessageView, userID string) error {
                if userID, _ := uuid.FromString(msg.UserID); userID == uuid.Nil {
                        return nil
                }

                data, err := base64.StdEncoding.DecodeString(msg.Data)
                if err != nil {
                        return err
                }

		/*
		user, err := rw.client.ReadUser(ctx,msg.UserID)
		if err != nil{
			return err
		}

		rw.db.Where(Users{UserID: msg.UserID}).FirstOrCreate(&Users {
			UserID: msg.UserID,
			ConversationID: msg.ConversationID,
			UserName: user.FullName,
			FirstActive: time.Now().Format(TIMEFORMAT),
		})
		*/

                switch msg.Category{
                case mixin.MessageCategoryAppCard:
                        rw.handleAppCard(ctx, msg, data)

                case mixin.MessageCategoryPlainText:
                        rw.handlePlainText(ctx, msg, data)

                case mixin.MessageCategorySystemAccountSnapshot:
                        rw.handleSnapshot(ctx, msg, data)

		case mixin.MessageCategoryPlainAudio:
			rw.handleAudio(ctx, msg, data)

		case mixin.MessageCategoryPlainData:
			rw.handleData(ctx, msg, data)

                default:
			log.Println(string(data))
                }
                return err
        }
        return mixin.BlazeListenFunc(talk)
}

func (rw *MessageWorker) Loop(ctx context.Context) {
        for {
                err := rw.client.LoopBlaze(ctx, rw.OnMessage(ctx))
                log.Printf("LoopBlaze() => %v\n", err)
                if ctx.Err() != nil {
                        break
                }
                time.Sleep(1 * time.Second)
        }
}

