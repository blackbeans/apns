package log4moa

import (
	"fmt"
	log "github.com/blackbeans/log4go"
	"github.com/go-errors/errors"
	"sync"
	"sync/atomic"
	"time"
)

const (
	MAX_ROTATE_SIZE = 10
	MOA_STAT_LOG    = "moa-stat"
)

type MoaInfo struct {
	Recv  int64 `json:"recv"`
	Proc  int64 `json:"proc"`
	Error int64 `json:"error"`
}

//
type MoaStat struct {
	lasterMoaInfo *MoaInfo
	currMoaInfo   *MoaInfo
	RotateSize    int32
	network       func() string
	MoaTicker     *time.Ticker
	lock          sync.RWMutex
}

type MoaLog interface {
	StartLog()
	Destory()
}

func NewMoaStat(network func() string) *MoaStat {
	moaStat := &MoaStat{
		currMoaInfo: &MoaInfo{},
		RotateSize:  0,
		network:     network}
	return moaStat
}

func (self *MoaStat) StartLog() {
	ticker := time.NewTicker(time.Second * 1)
	self.MoaTicker = ticker
	go func() {
		defer func() {
			if err := recover(); nil != err {
				var e error
				er, ok := err.(*errors.Error)
				if ok {
					stack := er.ErrorStack()
					e = errors.New(stack)
				} else {
					e = errors.New(fmt.Sprintf("time.ticker Call Err %s", err))
				}

				log.ErrorLog("stderr", "time.ticker|Invoke|FAIL|%s", e)
				// 销毁定时器
				self.Destory()
			}

		}()
		log.InfoLog(MOA_STAT_LOG, "RECV\tPROC\tERROR\tNetWork")
		for {
			<-ticker.C
			if self.RotateSize == MAX_ROTATE_SIZE {
				log.InfoLog(MOA_STAT_LOG, "RECV\tPROC\tERROR\tNetWork")
				log.InfoLog(MOA_STAT_LOG, "%d\t%d\t%d\t%s",
					self.currMoaInfo.Recv, self.currMoaInfo.Proc, self.currMoaInfo.Error, self.network())
				// self.RotateSize = 0
				atomic.StoreInt32(&self.RotateSize, 0)
			} else {
				log.InfoLog(MOA_STAT_LOG, "%d\t%d\t%d\t%s",
					self.currMoaInfo.Recv, self.currMoaInfo.Proc, self.currMoaInfo.Error, self.network())
				// self.RotateSize++
				atomic.AddInt32(&self.RotateSize, 1)
			}
			self.reset()
		}
	}()
}

func (self *MoaStat) IncreaseRecv() {
	atomic.AddInt64(&self.currMoaInfo.Recv, 1)
}

func (self *MoaStat) IncreaseProc() {
	atomic.AddInt64(&self.currMoaInfo.Proc, 1)
}

func (self *MoaStat) IncreaseError() {
	atomic.AddInt64(&self.currMoaInfo.Error, 1)
}

func (self *MoaStat) GetMoaInfo() *MoaInfo {
	return self.currMoaInfo
}

func (self *MoaStat) reset() {
	self.lasterMoaInfo = &MoaInfo{
		Recv:  self.currMoaInfo.Recv,
		Proc:  self.currMoaInfo.Proc,
		Error: self.currMoaInfo.Error,
	}
	atomic.StoreInt64(&self.currMoaInfo.Recv, 0)
	atomic.StoreInt64(&self.currMoaInfo.Proc, 0)
	atomic.StoreInt64(&self.currMoaInfo.Error, 0)
}

func (self *MoaStat) Destory() {
	self.MoaTicker.Stop()
}
