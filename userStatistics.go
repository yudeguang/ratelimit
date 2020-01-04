package ratelimit

import (
	"fmt"
	"sort"
)

/*
某用户剩余访问次数，例:
RemainingVisits("username")
*/
func (this *Rule) RemainingVisits(key interface{}) []int {
	arr := make([]int, 0, len(this.rules))
	for i := range this.rules {
		arr = append(arr, this.rules[i].RemainingVisits(key))
	}
	return arr
}

/*
以IP作为用户名，此用户剩余访问次数,例:
RemainingVisitsByIP("127.0.0.1")
*/
func (this *Rule) RemainingVisitsByIP(ip string) []int {
	ipInt64 := ip4StringToInt64(ip)
	if ipInt64 == 0 {
		return []int{}
	}
	return this.RemainingVisits(ipInt64)
}

//获得当前所有的在线用户
func (this *Rule) CurOnlineUsers() [][]string {
	var users [][]string
	for i := range this.rules {
		var user []string
		f := func(k, v interface{}) bool {
			user = append(user, fmt.Sprint(k))
			return true
		}
		sort.Strings(user)
		this.rules[i].indexes.Range(f)
		users = append(users, user)
	}
	return users
}

//获取当前所有访问过的IP,前提是我们必须是以IP作为用户名存储，否则程序会崩溃(k.(int64))
func (this *Rule) CurOnlineIPs() [][]string {
	var users [][]string
	for i := range this.rules {
		var user []string
		f := func(k, v interface{}) bool {
			user = append(user, int64ToIp4String(k.(int64)))
			return true
		}
		sort.Strings(user)
		this.rules[i].indexes.Range(f)
		users = append(users, user)
	}
	return users
}
