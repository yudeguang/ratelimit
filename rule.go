// Copyright 2020 rateLimit Author(https://github.com/yudeguang/ratelimit). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/yudeguang/ratelimit.
package ratelimit

import (
	"math"
	"sort"
	"strconv"
	"sync"
	"time"
)

//用户访问控制策略,可由一个或多个访问控制规则组成
type Rule struct {
	rules []*singleRule
	//以下用于备份数据，在需要备份时才在作用
	needBackup         bool          //是否需要把数据备份到硬盘，开启备份之后，不允许再临时增加规则singleRule
	backupFileName     string        //缓存存到硬盘上的文件名
	backUpInterval     time.Duration //默认多长时间需要执行一次数据备份操作
	lockerForBackup    *sync.Mutex   //用于数据备份
	loadBackupFileOnce sync.Once
}

/*
初始化一个多重规则的频率控制策略，例：
r := NewRule()
初始化之后，紧跟着需要调用AddRule方法增加一条或若干条用户访问控制策略，增加用户访问控制策略后，才可以正式使用
*/
func NewRule() *Rule {
	return new(Rule)
}

/*
增加用户访问控制策略，例:
r.AddRule(time.Minute*5, 20)
r.AddRule(time.Minute*30, 50)
r.AddRule(time.Hour*24, 200)
它表示:
在5分钟内每个用户最多允许访问20次
在30分钟内每个用户最多允许访问50次
在24小时内每个用户最多允许访问200次
其中:
defaultExpiration              表示在某个时间段内
numberOfAllowedAccesses        表示允许访问的次数
estimatedNumberOfOnlineUserNum 表示预计可能有多少人访问,此参数为可变参数,可不填写
以上任何一条用户访问控制策略没通过,都不允许访问，注意单条规则中，不宜设定监控时间段过大的规则，比如设定监控某个用户一个月甚至是1年的访问规则，它会占用大多的内存
*/
func (r *Rule) AddRule(defaultExpiration time.Duration, numberOfAllowedAccesses int, estimatedNumberOfOnlineUserNum ...int) {
	//开启备份之后，不下允许添加规则
	if r.needBackup {
		panic("cant't use AddRule after LoadingAndAutoSaveToDisc")
	}
	r.rules = append(r.rules, newsingleRule(defaultExpiration, numberOfAllowedAccesses, estimatedNumberOfOnlineUserNum...))
	//把时间控制调整为从小到大排列，防止用户在实例化的时候，未按照预期的时间顺序添加，导致某些规则失效
	sort.Slice(r.rules, func(i int, j int) bool {
		return r.rules[i].defaultExpiration < r.rules[j].defaultExpiration
	})
	//如果有多条规则，单位时间内所承载的访问量需要有递进关系，否则则非法
	if len(r.rules) > 1 {
		var pre = math.MaxFloat64
		for i, v := range r.rules {
			cur := float64(v.numberOfAllowedAccesses) / float64(v.defaultExpiration.Nanoseconds())
			if cur > pre {
				panic(`This rule is illegal,please modify the relevant rules:"allow ` + strconv.Itoa(v.numberOfAllowedAccesses) + ` visits within ` + v.defaultExpiration.String() +
					`" can't be bigger than "allow ` + strconv.Itoa(r.rules[i-1].numberOfAllowedAccesses) + ` visits within ` + r.rules[i-1].defaultExpiration.String() + `"`)
			}
			pre = cur
		}
	}
}

/*
是否还允许某用户访问，如果访问量过多，超出各细分规则中任何一条规则规定的访问量，则不允许访问
无论是否允许访问都会尝试在各细分访问规则记录中增加一条访问日志记录，函数AllowVisit也可以认为
是AddRecords
例:
AllowVisit("username")
*/
func (r *Rule) AllowVisit(key interface{}) bool {
	if len(r.rules) == 0 {
		panic("rule is empty，please add rule by AddRule")
	}
	//这个地方需要注意，如果前面的某些策略通过，但是后面的策略不通过。这时候，在前面允许访问的策略中，
	//允许访问次数是会减少的,我们这里并没有严格的做回滚操作。
	//原因在于一方面是性能，另外一方面是随着
	//时间流逝，前面的策略中允许访问的次数很快就会自动增长。
	for i := range r.rules {
		if !r.rules[i].allowVisit(key) {
			return false
		}
	}
	return true
}

/*
以IP作为用户名，判断该用户是否允许访问,例:
AllowVisitByIP4("127.0.0.1")
在实际的网站运营中，往往需要以IP作为判断用户的标准，IP转换为int64存储可略微减少内存占用
*/
func (r *Rule) AllowVisitByIP4(ip string) bool {
	ipInt64 := ip4StringToInt64(ip)
	if ipInt64 == 0 {
		return false
	}
	return r.AllowVisit(ipInt64)
}

/*
人工清空某用户的访问数据，主要针对某些特定客户的个性化需求，比如某个客户要求临时允许其访问更多的页面，
此时，调用出函数，清空其历史访问数据，间接实现这个目的,例:
ManualEmptyVisitorRecordsOf("andyyu")
*/
func (r *Rule) ManualEmptyVisitorRecordsOf(key interface{}) {
	if len(r.rules) == 0 {
		panic("rule is empty，please add rule by AddRule")
	}
	for i := range r.rules {
		r.rules[i].manualEmptyVisitorRecordsOf(key)
	}
}

// 人工清空所有用户的访问数据
func (r *Rule) ManualEmptyVisitorRecordsOfAll() {
	for i := range r.rules {
		r.rules[i].usedVisitorRecordsIndex.Range(func(k, v interface{}) bool {
			r.rules[i].manualEmptyVisitorRecordsOf(k)
			return true
		})
	}
}
