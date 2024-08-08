package cleisthenes

import (
	"fmt"
	"time"
)

type EpochResult struct {
	Epoch Epoch
	Count int
}

type TPSInfo struct {
	Epoch       Epoch
	Total       int64
	SingleCount int64
	OutputTime  time.Time
}

type TPS struct {
	EnterTime time.Time
	StartTime time.Time

	LastOutputTime time.Time
	Epoch          Epoch

	Total               int64
	SingleCount         int64
	CompleteCount       chan EpochResult
	BranchCompleteCount chan EpochResult

	History []TPSInfo

	Rbc     time.Duration
	Ce      time.Duration
	Irbc    time.Duration
	Bba     time.Duration
	Acs     time.Duration
	TpkeEnc time.Duration
	TpkeDec time.Duration
}

var (
	BranchTps TPS
	MainTps   TPS
)

func (t *TPS) ListenResult(prefix string) {

	t.CompleteCount = make(chan EpochResult, 1000)

	go func() {
		for {
			select {
			case epochResult := <-t.CompleteCount:
				t.Total += int64(epochResult.Count)
				t.SingleCount = int64(epochResult.Count)
				curTime := time.Now()
				t.LastOutputTime = curTime
				t.Epoch = epochResult.Epoch

				t.History = append(t.History, TPSInfo{
					Epoch:       t.Epoch,
					Total:       t.Total,
					SingleCount: t.SingleCount,
					OutputTime:  t.LastOutputTime,
				})

				timeConsume := curTime.UnixMilli() - t.StartTime.UnixMilli()
				timeConsume2 := curTime.UnixMilli() - t.EnterTime.UnixMilli()
				tps := float64(t.Total) / float64(timeConsume) * 1000.0
				tps2 := float64(t.SingleCount) / float64(timeConsume2) * 1000.0

				fmt.Printf("%s Epoch:%d 开始时间：%s - %s total:%d 耗时：%dms tps:%f\n", prefix,
					epochResult.Epoch, t.StartTime.Format(time.StampMicro), curTime.Format(time.StampMicro), t.Total, timeConsume, tps)
				fmt.Printf("%s Epoch:%d 单次时间：%s - %s total:%d 耗时: %dms tps:%f\n", prefix,
					epochResult.Epoch, t.EnterTime.Format(time.StampMicro), curTime.Format(time.StampMicro), t.SingleCount, timeConsume2, tps2)
			}
		}
	}()

}
