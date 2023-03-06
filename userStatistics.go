// Copyright 2020 ratelimit Author(https://github.com/yudeguang/ratelimit). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/yudeguang/ratelimit.
package ratelimit

import (
	"fmt"
	"sort"
	"strconv"
)

const (
	Chinese = iota
	English
)

/*
某用户剩余访问次数，只返回所有规则中时间范围跨度最大的一条规则的剩余访问次数。
例如，假如规定了每天能访问多少次，每小时能返回多少次，每分钟能返回多少次
那么此处只返回每天剩余访问次数。
例:
RemainingVisit("username")
*/
func (r *Rule) RemainingVisit(key interface{}) int {
	return rightInt(r.RemainingVisits(key), 1)[0]
}

// 返回右往左最多n个元素
func rightInt(s []int, n int) []int {
	if len(s) < n {
		return s
	}
	if n <= 0 {
		return nil
	}
	return s[len(s)-n:]
}

/*
某用户剩余访问次数，例:
RemainingVisits("username")
*/
func (r *Rule) RemainingVisits(key interface{}) []int {
	arr := make([]int, 0, len(r.rules))
	for i := range r.rules {
		arr = append(arr, r.rules[i].remainingVisits(key))
	}
	return arr
}

/*
打印各细分规则下的剩余访问次数
*/
func (r *Rule) PrintRemainingVisits(key interface{}, language ...int) {
	//先确定语言，默认为中文，目前只支持中文，英文两种语言
	lan := 0
	if len(language) == 1 && language[0] == 1 {
		lan = 1
	}
	for i := range r.rules {
		if lan == 0 {
			fmt.Println(key, "在", r.rules[i].defaultExpiration, "内共允许访问", r.rules[i].numberOfAllowedAccesses, "次,剩余", r.rules[i].remainingVisits(key))
		} else {
			fmt.Println(key, "allowed", r.rules[i].numberOfAllowedAccesses, "visits within", r.rules[i].defaultExpiration, ",with", r.rules[i].remainingVisits(key), "remaining")
		}
	}
}

/*
以IP作为用户名，此用户剩余访问次数,例:
RemainingVisitsByIP4("127.0.0.1")
*/
func (r *Rule) RemainingVisitsByIP4(ip string) []int {
	ipInt64 := ip4StringToInt64(ip)
	if ipInt64 == 0 {
		return []int{}
	}
	return r.RemainingVisits(ipInt64)
}

// 获得当前所有的在线用户,注意所有用int64存储的用户会被默认认为是IP地址，会被自动转换为IP的字符串形式输出以方便查看
// 如果不是本身就是以int64形式存储，而不是IP4，那么可以用ip4StringToInt64自己再转换回去
func (r *Rule) GetCurOnlineUsers() []string {
	//向切片Sli中插入没出现过的元素V，如果切片中有V，则不插入
	var insertIgnoreString = func(s []string, v string) []string {
		for _, val := range s {
			if val == v {
				return s
			}
		}
		s = append(s, v)
		return s
	}
	var users []string
	for i := range r.rules {
		r.rules[i].usedVisitorRecordsIndex.Range(func(k, v interface{}) bool {
			var user string
			switch k.(type) {
			case int64:
				user = int64ToIp4String(k.(int64))
			default:
				user = fmt.Sprint(k)
			}
			users = insertIgnoreString(users, user)
			return true
		})
	}
	sort.Strings(users)
	return users
}

// 返回所有用户的剩余返回次数详情,注意，为简单起见，返回值被转化为string类型 默认只返回1000
func (r *Rule) GetCurOnlineUsersVisitsDetail(num ...int) (CurOnlineUsersVisitsDetail [][]string) {
	if len(num) > 0 && num[0] < 1 {
		panic("num must be>0")
	}
	CurOnlineUsers := r.GetCurOnlineUsers()
	sort.Strings(CurOnlineUsers)
	for _, user := range CurOnlineUsers {
		visits := r.RemainingVisits(user)
		var visitsString []string
		visitsString = append(visitsString, user)
		for i := range visits {
			visitsString = append(visitsString, strconv.Itoa(visits[i]))
		}
		if len(num) == 0 {
			if len(CurOnlineUsersVisitsDetail) >= 1000 {
				break
			}
		} else {
			if len(CurOnlineUsersVisitsDetail) >= num[0] {
				break
			}
		}
		CurOnlineUsersVisitsDetail = append(CurOnlineUsersVisitsDetail, visitsString)
	}
	return
}
