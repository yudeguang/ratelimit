// Copyright 2020 ratelimit Author(https://github.com/yudeguang/ratelimit). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/yudeguang/ratelimit.
package ratelimit

import (
	"errors"
	"time"
)

/*
用环形队列做为底层数据结构来存储用户访问数据
*/

//使用切片实现的队列
type circleQueueInt64 struct {
	maxSize int     //比实际队列长度大1
	slice   []int64 //切片会被实际队列长度大1
	head    int     //头
	tail    int     //尾
}

//初始化环形队列
func newCircleQueueInt64(size int) *circleQueueInt64 {
	var c circleQueueInt64
	c.maxSize = size + 1
	c.slice = make([]int64, c.maxSize)
	return &c
}

//入对列
func (this *circleQueueInt64) Push(val int64) (err error) {
	if this.IsFull() {
		return errors.New("queue is full")
	}
	this.slice[this.tail] = val
	this.tail = (this.tail + 1) % this.maxSize
	return
}

//出对列
func (this *circleQueueInt64) Pop() (val int64, err error) {
	if this.IsEmpty() {
		return 0, errors.New("queue is empty")
	}
	val = this.slice[this.head]
	this.head = (this.head + 1) % this.maxSize
	return
}

//判断队列是否已满
func (this *circleQueueInt64) IsFull() bool {
	return (this.tail+1)%this.maxSize == this.head
}

//判断队列是否为空
func (this *circleQueueInt64) IsEmpty() bool {
	return this.tail == this.head
}

//判断已使用多少个元素
func (this *circleQueueInt64) UsedSize() int {
	return (this.tail + this.maxSize - this.head) % this.maxSize
}

//判断队列中还有多少空间未使用
func (this *circleQueueInt64) UnUsedSize() int {
	return this.maxSize - 1 - this.UsedSize()
}

//队列总的可用空间长度
func (this *circleQueueInt64) Len() int {
	return this.maxSize - 1
}

//删除过期数据
func (this *circleQueueInt64) DeleteExpired() {
	now := time.Now().UnixNano()
	size := this.UsedSize()
	if size == 0 {
		return
	}
	//依次删除过期数据
	for i := 0; i < size; i++ {
		if now > this.slice[this.head] {
			this.Pop()
		} else {
			return
		}
	}
}
