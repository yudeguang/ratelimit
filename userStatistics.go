package ratelimit

import (
	"fmt"
	"github.com/yudeguang/slice"
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
RemainingVisitsByIP4("127.0.0.1")
*/
func (this *Rule) RemainingVisitsByIP4(ip string) []int {
	ipInt64 := ip4StringToInt64(ip)
	if ipInt64 == 0 {
		return []int{}
	}
	return this.RemainingVisits(ipInt64)
}

//获得当前所有的在线用户,注意所有用int64存储的用户会被默认认为是IP地址，会被自动转换为IP的字符串形式输出以方便查看
func (this *Rule) GetCurOnlineUsers() []string {
	var users []string
	for i := range this.rules {
		f := func(k, v interface{}) bool {
			var user string
			switch k.(type) {
			case int64:
				user = int64ToIp4String(k.(int64))
			default:
				user = fmt.Sprint(k)
			}
			users = slice.InsertIgnoreString(users, user)
			return true
		}
		this.rules[i].indexes.Range(f)
	}
	sort.Strings(users)
	return users
}
