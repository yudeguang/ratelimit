package ratelimit

import (
	"sort"
	"time"
)

//用户访问控制策略,可由一个或多个访问控制规则组成
type Rule struct {
	rules []*singleRule
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
以上任何一条用户访问控制策略没通过,都不允许访问
*/
func (this *Rule) AddRule(defaultExpiration time.Duration, numberOfAllowedAccesses int, estimatedNumberOfOnlineUserNum ...int) {
	this.rules = append(this.rules, newsingleRule(defaultExpiration, numberOfAllowedAccesses, estimatedNumberOfOnlineUserNum...))
	//把时间控制调整为从小到大排列，防止用户在实例化的时候，未按照预期的时间顺序添加，导致某些规则失效
	sort.Slice(this.rules, func(i int, j int) bool {
		return this.rules[i].defaultExpiration < this.rules[j].defaultExpiration
	})
}

/*
某用户是否允许访问,若允许访问,则会分别往该规则的各细分访问记录中各自动增加一条访问记录，例:
AllowVisit("username")
*/
func (this *Rule) AllowVisit(user interface{}) bool {
	if len(this.rules) == 0 {
		panic("访问规则暂时为空，请调用AddRule为其增加访问规则")
	}
	//这个地方需要注意，如果前面的某些策略通过，但是后面的策略不通过。这时候，在前面允许访问的策略中，
	//允许访问次数同样是会减少的,我们这里并没有严格的做回滚操作。原因在于，一方面是性能，另外一方面是随着
	//时间流逝，前面的策略中允许访问的次数很快就会自动增长。
	for i := range this.rules {
		if err := this.rules[i].add(user); err != nil {
			return false
		}
	}
	return true
}

/*
以IP作为用户名，判断该用户是否允许访问,例:
AllowVisitByIP("127.0.0.1")
在实际的网站运营中，往往需要以IP作为判断用户的标准
*/
func (this *Rule) AllowVisitByIP(ip string) bool {
	ipInt64 := ip4StringToInt64(ip)
	if ipInt64 == 0 {
		return false
	}
	return this.AllowVisit(ipInt64)
}
