package snapshots

import (
	"log"

	"github.com/fox-one/mixin-sdk-go"
)

func (sw *SnapshotsWorker) WriteInputSnapshot(s *mixin.Snapshot) {
	result := sw.db.Create(&InputSnapshot{
		SnapshotID: s.SnapshotID,
		TraceID:    s.TraceID,
		OpponentID: s.OpponentID,
		CreatedAt:  s.CreatedAt.String(),
		Memo:       s.Memo,
	})
	if result.Error != nil {
		log.Println("db.Create(InputSnapshots) => ", result.Error)
	}
}

func (sw *SnapshotsWorker) WriteSwap(receiverID, followID, time string) {
	result := sw.db.Create(&SwapOrder{
		ReceiverID: receiverID,
		FollowID:   followID,
		CreatedAt:  time,
		OrderState: "Init",
	})
	if result.Error != nil {
		log.Println("db.Create(WriteSnapshots) => ", result.Error)
	}
}

func (sw *SnapshotsWorker) WriteOutputSnapshot(s *mixin.Snapshot, inputSnID, toAddress string) {
	result := sw.db.Create(&OutputSnapshot{
		InputsnID:  inputSnID,
		SnapshotID: s.SnapshotID,
		TraceID:    s.TraceID,
		ToAddress:  toAddress,
		CreatedAt:  s.CreatedAt.String(),
		AssetID:    s.AssetID,
		Amount:     s.Amount.String(),
		Memo:       s.Memo,
	})
	if result.Error != nil {
		log.Println("db.Create(OutputSnapshots) => ", result.Error)
	}
}
