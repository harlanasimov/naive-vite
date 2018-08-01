package syncer

import (
	"sync"
	"time"
)

type sender interface {
	sendA(tasks []hashTask)
	sendB(task hashTask, prevCnt int)
}

type retryPolicy interface {
	retry(hash string) bool
	done(hash string)
}

type RetryStatus struct {
	cnt   int
	done  bool
	ftime time.Time // first time
	dtime time.Time // done time
}

func (self *RetryStatus) reset() {
	self.cnt = 1
	self.done = false
	self.ftime = time.Now()
}

func (self *RetryStatus) finish() {
	self.done = true
	self.cnt = 0
	self.dtime = time.Now()
}
func (self *RetryStatus) inc() {
	self.cnt = self.cnt + 1
}

type defaultRetryPolicy struct {
	fetchedHashs map[string]*RetryStatus
	mu           sync.Mutex
}

func (self *defaultRetryPolicy) done(hash string) {
	self.mu.Lock()
	defer self.mu.Unlock()
	status, ok := self.fetchedHashs[hash]
	if ok {
		status.finish()
	} else {
		tmp := self.newRetryStatus()
		tmp.finish()
		self.fetchedHashs[hash] = tmp
	}
}

func (self *defaultRetryPolicy) retry(hash string) bool {
	self.mu.Lock()
	defer self.mu.Unlock()
	status, ok := self.fetchedHashs[hash]
	now := time.Now()
	if ok {
		status.inc()
		if status.done {
			// cnt>5 && now - dtime > 10s
			if status.cnt > 5 && now.After(status.dtime.Add(time.Second*10)) {
				status.reset()
				return true
			}
		} else {
			// cnt>5 && now - ftime > 5s
			if status.cnt > 10 && now.After(status.ftime.Add(time.Second*5)) {
				status.reset()
				return true
			}
		}
	} else {
		self.fetchedHashs[hash] = self.newRetryStatus()
		return true
	}
	return false
}

func (self *defaultRetryPolicy) newRetryStatus() *RetryStatus {
	return &RetryStatus{done: false, cnt: 1, ftime: time.Now()}
}

type fetcher struct {
	sender sender

	retryPolicy retryPolicy
}

type hashTask struct {
	height int
	hash   string
}

func (self *fetcher) fetchBlockByHash(tasks []hashTask) {
	var target []hashTask
	for _, task := range tasks {
		if self.retryPolicy.retry(task.hash) {
			target = append(target, task)
		}
	}
	if len(target) > 0 {
		self.sender.sendA(target)
	}
}

func (self *fetcher) fetchHash(hashTask hashTask, prevCnt int) {
	self.sender.sendB(hashTask, prevCnt)
}
func (self *fetcher) done(block string, height int) {

	self.retryPolicy.done(block)
}
