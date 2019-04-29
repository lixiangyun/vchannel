package main

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"
)

type Item struct {
	size  uint64
	count uint64
	nums  int64
}

type Stat struct {
	prefix   string
	now      Item
	old      Item
	interval uint64
}

func (now *Item) Sub(old Item) {
	now.size -= old.size
	now.count -= old.count
}

func (now *Item) Add(add Item) {
	now.size += add.size
	now.count += add.count
}

func (now *Item) Div(interval uint64) {
	now.size = now.size / interval
	now.count = now.count / interval
}

func (now *Item) Avg(nums uint64) {
	now.size = now.size / nums
	now.count = now.count / nums
}

func calcUnit(cnt uint64) string {
	if cnt < 1024 {
		return fmt.Sprintf("%d", cnt)
	} else if cnt < 1024*1024 {
		return fmt.Sprintf("%.2f k/s", float32(cnt)/1024)
	} else if cnt < 1024*1024*1024 {
		return fmt.Sprintf("%.2f M/s", float32(cnt)/(1024*1024))
	} else {
		return fmt.Sprintf("%.2f G/s", float32(cnt)/(1024*1024*1024))
	}
}

func calcTime(tm uint64) string {
	if tm < uint64(time.Microsecond) {
		return fmt.Sprintf("%d ns", tm)
	} else if tm < uint64(time.Millisecond) {
		return fmt.Sprintf("%.2f us", float64(tm)/float64(time.Microsecond))
	} else if tm < uint64(time.Second) {
		return fmt.Sprintf("%.2f ms", float64(tm)/float64(time.Millisecond))
	} else {
		return fmt.Sprintf("%.2f s", float64(tm)/float64(time.Second))
	}
}

func (now *Item) Format() string {
	str := fmt.Sprintf("[ nums %d , times %s , size %s ]", now.nums, calcUnit(now.count), calcUnit(now.size))
	return str
}

func (s *Stat) display() {

	timer := time.NewTimer(time.Duration(s.interval) * time.Second)
	for {
		<-timer.C

		now := s.now
		old := s.old

		now.Sub(old)
		now.Div(s.interval)
		str := now.Format()

		log.Printf("[%s] stat: %s\r\n", s.prefix, str)

		s.old = s.now
		timer.Reset(time.Duration(s.interval) * time.Second)
	}
}

var globalstat Stat

func init() {
	globalstat.interval = 10
	go globalstat.display()
}

func StatPrefix(str string) {
	globalstat.prefix = str
}

func StatAdd(size int) {
	atomic.AddUint64(&globalstat.now.count, 1)
	atomic.AddUint64(&globalstat.now.size, uint64(size))
}

func StatNumsAdd() {
	atomic.AddInt64(&globalstat.now.nums, 1)
}

func StatNumsSub() {
	atomic.AddInt64(&globalstat.now.nums, -1)
}
