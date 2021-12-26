// Copyright 2020 rateLimit Author(https://github.com/yudeguang/ratelimit). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/yudeguang/ratelimit.
package ratelimit

import (
	"errors"
	"sync"
	"time"
)

/*
用环形队列做为底层数据结构来存储用户访问数据,暂时废弃,实际使用自动扩容环形队列代替
*/

//使用切片实现的队列
type circleQueueInt64 struct {
	key interface{}
	//注意，maxSize及visitorRecord长度都比实际存储长度大1
	maxSize       int
	visitorRecord []int64
	head          int
	tail          int
	//存盘时临时用到的虚拟队列的头和尾
	headForCopy int
	tailForCopy int
	locker      *sync.Mutex
}

//初始化环形队列
func newCircleQueueInt64(size int) *circleQueueInt64 {
	var c circleQueueInt64
	c.maxSize = size + 1
	c.visitorRecord = make([]int64, c.maxSize)
	c.locker = new(sync.Mutex)
	return &c
}

//入对列
func (q *circleQueueInt64) push(val int64) (err error) {
	q.locker.Lock()
	defer q.locker.Unlock()
	if q.isFull() {
		return errors.New("queue is full")
	}
	q.visitorRecord[q.tail] = val
	q.tail = (q.tail + 1) % q.maxSize
	return
}

//出对列
func (q *circleQueueInt64) pop() (val int64, err error) {
	q.locker.Lock()
	defer q.locker.Unlock()
	if q.isEmpty() {
		return 0, errors.New("queue is empty")
	}
	val = q.visitorRecord[q.head]
	q.head = (q.head + 1) % q.maxSize
	return
}

//出对列
func (q *circleQueueInt64) popForCopy() (val int64, err error) {
	q.locker.Lock()
	defer q.locker.Unlock()
	if q.isEmptyForCopy() {
		return 0, errors.New("queue is empty")
	}
	val = q.visitorRecord[q.headForCopy]
	q.headForCopy = (q.headForCopy + 1) % q.maxSize
	return
}

//判断队列是否已满
func (q *circleQueueInt64) isFull() bool {
	return (q.tail+1)%q.maxSize == q.head
}

//判断队列是否为空
func (q *circleQueueInt64) isEmpty() bool {
	return q.tail == q.head
}

//判断队列是否为空
func (q *circleQueueInt64) isEmptyForCopy() bool {
	return q.tailForCopy == q.headForCopy
}

//判断已使用多少个元素
func (q *circleQueueInt64) usedSizeForCopy() int {
	return (q.tailForCopy + q.maxSize - q.headForCopy) % q.maxSize
}

//判断已使用多少个元素
func (q *circleQueueInt64) usedSize() int {
	return (q.tail + q.maxSize - q.head) % q.maxSize
}

//判断队列中还有多少空间未使用
func (q *circleQueueInt64) unUsedSize() int {
	return q.maxSize - 1 - q.usedSize()
}

//队列总的可用空间长度
func (q *circleQueueInt64) Len() int {
	return q.maxSize - 1
}

//删除过期数据
func (q *circleQueueInt64) ueleteExpired(key interface{}) {
	if q.key != key {
		return
	}
	now := time.Now().UnixNano()
	size := q.usedSize()
	if size == 0 {
		return
	}
	//依次删除过期数据
	for i := 0; i < size; i++ {
		if now > q.visitorRecord[q.head] {
			q.head = (q.head + 1) % q.maxSize
		} else {
			return
		}
	}
}
