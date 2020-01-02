package ratelimit

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

//多策略访问规则
type Rule struct {
	Rule []*singleRule
}

/*
初始化一个多重规则的频率控制策略
例：
	r := NewRule()
	r.AddRule(time.Minute*5, 20) 或 r.AddRule(time.Minute*5, 20, 100)
	r.AddRule(time.Minute*30, 50) 或 r.AddRule(time.Minute*30, 50, 1000)
	r.AddRule(time.Hour*24, 200) 或  r.AddRule(time.Hour*24, 200, 10000)
它表示:
在5分钟内每个用户最多允许访问20次,并且我们预计在这5分钟内大致有100个用户会访问我们的网站
在30分钟内每个用户最多允许访问50次,并且我们预计在这30分钟内大致有1000个用户会访问我们的网站
在24小时内每个用户最多允许访问200次,并且我们预计在这24小时内大致有10000个用户会访问我们的网站
以上任何一条规则的访问次数超出，都不允许访问,并且我们在匹配规则时，是按时间段从小到大匹配
*/
func NewRule() *Rule {
	return new(Rule)
}

//增加一条访问规则
func (this *Rule) AddRule(defaultExpiration time.Duration, numberOfAllowedAccesses int, estimatedNumberOfOnlineUserNum ...int) {
	this.Rule = append(this.Rule, newsingleRule(defaultExpiration, numberOfAllowedAccesses, estimatedNumberOfOnlineUserNum...))
	//把时间控制调整为从小到大排列，防止用户在实例化的时候，未按照预期的时间顺序添加，导致某些规则失效
	sort.Slice(this.Rule, func(i int, j int) bool {
		return this.Rule[i].defaultExpiration < this.Rule[j].defaultExpiration
	})
}

//是否允许访问,允许访问则往访问记录中加入一条访问记录
//例: AllowVisit("usernameexample")
func (this *Rule) AllowVisit(key interface{}) bool {
	if len(this.Rule) == 0 {
		panic("访问规则暂时为空，请调用AddRule为其增加访问规则")
	}
	//这个地方需要注意，如果前面的某些策略通过，但是后面的策略不通过。这时候，在前面允许访问的策略中，
	//允许访问次数同样是会减少的,我们这里并没有严格的做回滚操作。原因在于，一方面是性能，另外一方面是随着
	//时间流逝，前面的策略中允许访问的次数很快就会自动增长。
	for i := range this.Rule {
		if err := this.Rule[i].add(key); err != nil {
			return false
		}
	}
	return true
}

//是否允许某IP的用户访问
//例: AllowVisitIP("127.0.0.1")
func (this *Rule) AllowVisitIP(ip string) bool {
	ipInt64 := Ip4StringToInt64(ip)
	if ipInt64 == 0 {
		return false
	}
	return this.AllowVisit(ipInt64)
}

//当前在线用户总数,每个时间段的在线用户总数
func (this *Rule) CurOnlineUserNum() []int {
	arr := make([]int, 0, len(this.Rule))
	for i := range this.Rule {
		arr = append(arr, this.Rule[i].CurOnlineUserNum())
	}
	return arr
}

//剩余访问次数
//例: RemainingVisits("usernameexample")
func (this *Rule) RemainingVisits(key interface{}) []int {
	arr := make([]int, 0, len(this.Rule))
	for i := range this.Rule {
		arr = append(arr, this.Rule[i].RemainingVisits(key))
	}
	return arr
}

//某IP剩余访问次数
//例: RemainingVisitsIP("127.0.0.1")
func (this *Rule) RemainingVisitsIP(ip string) []int {
	ipInt64 := Ip4StringToInt64(ip)
	if ipInt64 == 0 {
		return []int{}
	}
	return this.RemainingVisits(ipInt64)
}

//把在线用户数据转化成JSON输出,数据较大时，最好不要使用
func (this *Rule) OnlineUserInfoToJson() string {
	var pp []printHelper
	for i := range this.Rule {
		var p printHelper
		p.DefaultExpiration = this.Rule[i].defaultExpiration.String()
		p.NumberOfAllowedAccesses = this.Rule[i].numberOfAllowedAccesses
		p.EstimatedNumberOfOnlineUsers = this.Rule[i].estimatedNumberOfOnlineUsers
		var CurOnlineUserInfo []userInfo
		this.Rule[i].indexes.Range(func(k, v interface{}) bool {
			this.Rule[i].lock.Lock() //range里面不能用defer
			index := v.(int)
			//防止越界出错，理论上不存在这种情况
			if index < len(this.Rule[i].visitorRecords) && index >= 0 {
				this.Rule[i].visitorRecords[index].DeleteExpired()
				//删除完过期数据之后，如果该用户的所有访问记录均过期了，那么就删除该用户
				//并把该空间返还给notUsedVisitorRecordsIndex以便下次重复使用
				if this.Rule[i].visitorRecords[index].UsedSize() == 0 {
					this.Rule[i].indexes.Delete(k)
					this.Rule[i].notUsedVisitorRecordsIndex.Add(index)
				} else {
					//加入统计数据表
					var u userInfo
					u.UserName = fmt.Sprint(k)
					u.RemainingVisits = this.Rule[i].visitorRecords[index].UnUsedSize()
					u.Used = this.Rule[i].visitorRecords[index].UsedSize()
					CurOnlineUserInfo = append(CurOnlineUserInfo, u)
				}
			} else {
				this.Rule[i].indexes.Delete(k)
			}
			this.Rule[i].lock.Unlock()
			return true
		})
		sort.Slice(CurOnlineUserInfo, func(i int, j int) bool { return CurOnlineUserInfo[i].UserName < CurOnlineUserInfo[j].UserName })
		p.CurOnlineUserInfo = CurOnlineUserInfo
		p.CurOnlineUserNum = len(p.CurOnlineUserInfo)
		pp = append(pp, p)
	}
	b, _ := json.Marshal(pp)
	return string(b)
}
