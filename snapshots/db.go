package snapshots

import (
	"log"
	"time"

	"github.com/fox-one/mixin-sdk-go"
)

func (sw *SnapshotsWorker) WriteInputSnapshot(s *mixin.Snapshot) {
	result := sw.db.Create(&InputSnapshot{
		SnapshotID: s.SnapshotID,
		TraceID:    s.TraceID,
		OpponentID: s.OpponentID,
		CreatedAt:  s.CreatedAt.Format(time.RFC3339),
		Memo:       s.Memo,
	})
	if result.Error != nil {
		log.Println("db.Create(InputSnapshots) => ", result.Error)
	}
}

func (sw *SnapshotsWorker) WriteSwap(order *SwapOrder) {
	result := sw.db.Create(order)
	if result.Error != nil {
		log.Println("db.WriteSwap() => ", result.Error)
	}
}

func (sw *SnapshotsWorker) UpdateSwap(newOrder *SwapOrder, followID string) {
	result := sw.db.Model(&SwapOrder{}).Where("follow_id = ?", followID).Updates(newOrder)
	if result.Error != nil {
		log.Println("db.UpdateSwap() => ", result.Error)
	}
}

func (sw *SnapshotsWorker) WriteOutputSnapshot(s *mixin.Snapshot, inputSnID, toAddress string) {
	result := sw.db.Create(&OutputSnapshot{
		InputSnID:  inputSnID,
		SnapshotID: s.SnapshotID,
		TraceID:    s.TraceID,
		ToAddress:  toAddress,
		CreatedAt:  s.CreatedAt.Format(time.RFC3339),
		AssetID:    s.AssetID,
		Amount:     s.Amount.String(),
		Memo:       s.Memo,
	})
	if result.Error != nil {
		log.Println("db.Create(OutputSnapshots) => ", result.Error)
	}
}
