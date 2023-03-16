 
在网站的运营中，经常会遇到需要对用户访问次数做限制的情况，比如非常典型的是对于某些付费访问服务，需要对访问频率做比较精确的限制，比如单个用户(或者每个IP)地址每天只允许访问多少次，然后每小时只允许访问多少次等等，ratelimit就是针对这种情况而设计。
    
不同于网关级限流(包括go.uber.org/ratelimit 漏桶限流以及github.com/juju/ratelimit 令牌桶限流),本限流方案为业务级限流，适用于平台运营中,精细化的按单个用户,按IP等限流,为业内rdeis滑动窗口限流方案的纯GO替代方案,并且支持持久化,可定期把历史数据备份到本地磁盘,程序重启也可保留之前的访问记录。
      
github.com/yudeguang/ratelimit 底层用一个大小能自动伸缩的环形队列来存储用户访问数据，并发安全，拥有较高性能的同时还非常省内存,同时拥有高达1000W次/秒的处理能力(redis约10W次/秒)。作为对比，与用redis的相关数据结构来实现用户访问控制相比，其用法相对简单。 
    
    

##使用案例如下
```go
package main

import (
	"fmt"
	"github.com/yudeguang/ratelimit"
	"log"
	"strconv"
	"sync"
	"time"
)

var userVisitRule visitRule

type visitRule struct {
	paidMember      *ratelimit.Rule
	freeMember      *ratelimit.Rule
	anonymousMember *ratelimit.Rule
}

func main() {
	log.SetFlags(log.Lshortfile | log.Ltime)
	example1()
	example2()
	example3()
}

// 简单规则案例
func example1() {
	//步骤一：初始化
	r := ratelimit.NewRule()
	//步骤二：增加一条或者多条规则组成复合规则，此复合规则必须至少包含一条规则
	r.AddRule(time.Second*1, 10)
	//步骤三：调用函数判断某用户是否允许访问   allow:= r.AllowVisit(user)
	for i := 0; i <= 20; i++ {
		allow := r.AllowVisit("user")
		if !allow {
			log.Println("访问量超出,其剩余访问次数情况如下:", r.RemainingVisits("user"))
		} else {
			log.Println("允许访问,其剩余访问次数情况如下:", r.RemainingVisits("user"))
		}
	}
}

// 模拟平台运营中针对不同级别用户，设定不同的访问频率控制
func example2() {
	//paidMember
	//步骤一：初始化
	userVisitRule.paidMember = ratelimit.NewRule()
	//步骤二：增加一条或者多条规则组成复合规则，此复合规则必须至少包含一条规则
	userVisitRule.paidMember.AddRule(time.Hour*24, 10000)
	userVisitRule.paidMember.AddRule(time.Hour*1, 1000)
	userVisitRule.paidMember.AddRule(time.Minute*1, 100)
	userVisitRule.paidMember.AddRule(time.Second*1, 20)
	//步骤三(可选):从本地磁盘加载历史访问数据,并指定10秒存储一次。不加载则表示不磁盘,程序中断后，访问数据将消失
	userVisitRule.paidMember.LoadingAndAutoSaveToDisc("userVisitRule_paidMember", time.Second*10)
	//freeMember
	userVisitRule.freeMember = ratelimit.NewRule()
	userVisitRule.freeMember.AddRule(time.Hour*24, 1000)
	userVisitRule.freeMember.AddRule(time.Hour*1, 100)
	userVisitRule.freeMember.AddRule(time.Minute*1, 10)
	userVisitRule.freeMember.AddRule(time.Second*1, 2)
	userVisitRule.freeMember.LoadingAndAutoSaveToDisc("userVisitRule_freemember", time.Second*10)
	//anonymousMember
	userVisitRule.anonymousMember = ratelimit.NewRule()
	userVisitRule.anonymousMember.AddRule(time.Hour*24, 100)
	userVisitRule.anonymousMember.AddRule(time.Second*1, 2)
	userVisitRule.anonymousMember.LoadingAndAutoSaveToDisc("userVisitRule_anonymousMember", time.Second*10)

	//模拟一定数量不同类型的用户，比如可用手机号或者用户名做为KEY，匿名用户，一般用IP作为KEY
	var paidMembers = []string{"17277777770", "17277777771", "17277777772", "17277777773", "17277777774", "17277777775", "17277777776"}
	var freeMembers = []string{"16277777770", "16277777771", "16277777772", "16277777773", "16277777774", "16277777775", "16277777776"}
	var anonymousMembers = []string{"192.168.0.2", "192.168.0.3", "192.168.0.4", "192.168.0.5", "192.168.0.6", "192.168.0.7", "192.168.0.8"}

	//步骤四：调用函数判断某用户是否允许访问
	/*
	   allow:= r.AllowVisit(user)
	*/
	fmt.Println("\r\n下面模拟单个用户持续访问的情况:")
	member := paidMembers[0]
	for i := 0; i <= 30; i++ {
		allow := userVisitRule.paidMember.AllowVisit(member)
		if !allow {
			log.Println(member, "访问量超出,其剩余访问次数情况如下:", userVisitRule.paidMember.RemainingVisits(member))
		} else {
			log.Println(member, "允许访问,其剩余访问次数情况如下:", userVisitRule.paidMember.RemainingVisits(member))
		}
	}
	//我们再等一段时间，看看paidMembers[0]这个用户的允许访问是否恢复
	fmt.Println("\r\n休息10秒后,该用户将恢复部分访问次数")
	time.Sleep(time.Second * 10)
	allow := userVisitRule.paidMember.AllowVisit(member)
	if !allow {
		log.Println(member, "访问量超出,其剩余访问次数情况如下:", userVisitRule.paidMember.RemainingVisits(member))
	} else {
		log.Println(member, "允许访问,其剩余访问次数情况如下:", userVisitRule.paidMember.RemainingVisits(member))
		fmt.Println("\r\n", userVisitRule.paidMember.RemainingVisits(member), "的具体含义为:")
		userVisitRule.paidMember.PrintRemainingVisits(member)
	}
	fmt.Println("\r\n手工清空", member, "后的访问记录")
	userVisitRule.paidMember.ManualEmptyVisitorRecordsOf(member)
	log.Println(member, "其剩余访问次数情况如下:", userVisitRule.paidMember.RemainingVisits(member))

	//下面模拟三种不同的用户，访问一些页面，之后再打印出来观察
	for i := 0; i <= 10; i++ {
		for _, v := range freeMembers {
			userVisitRule.paidMember.AllowVisit(v)
		}
	}
	for i := 0; i <= 10; i++ {
		for _, v := range freeMembers {
			userVisitRule.freeMember.AllowVisit(v)
		}
	}
	for i := 0; i <= 10; i++ {
		for _, v := range anonymousMembers {
			userVisitRule.anonymousMember.AllowVisit(v)
		}
	}
	//在实际的运营中，GetCurOnlineUsersVisitsDetail()函数可以自行包装以HTTP等形式输出
	fmt.Println("\r\n下面为现在付费用户剩余访问次数情况:")
	paidMemberDeatil := userVisitRule.paidMember.GetCurOnlineUsersVisitsDetail()
	for _, v := range paidMemberDeatil {
		log.Println(v)
	}
	fmt.Println("\r\n下面为现在免费用户剩余访问次数情况:")
	freeMemberDeatil := userVisitRule.freeMember.GetCurOnlineUsersVisitsDetail()
	for _, v := range freeMemberDeatil {
		log.Println(v)
	}
	fmt.Println("\r\n下面为现为匿名S用户剩余访问次数情况:")
	anonymousMemberDeatil := userVisitRule.anonymousMember.GetCurOnlineUsersVisitsDetail()
	for _, v := range anonymousMemberDeatil {
		log.Println(v)
	}

}

// 模拟1000个用户，累计进行总共约1亿次性能测试
func example3() {
	var Visits int //因并发问题num比实际数量稍小
	fmt.Println("\r\n性能测试，预计耗时1分钟，请耐心等待:")
	//步骤一：初始化
	r := ratelimit.NewRule()
	//步骤二：增加一条或者多条规则组成复合规则，规则必须至少包含一条规则
	//此处对于性能测试，为方便准确计数，只需要添加一条规则
	r.AddRule(time.Second*10, 1000) //每10秒只允许访问1000次
	/*
		r.AddRule(time.Second*10, 10)   //每10秒只允许访问10次
		r.AddRule(time.Minute*30, 1000) //每30分钟只允许访问1000次
		r.AddRule(time.Hour*24, 5000)   //每天只允许访问500次
	*/
	//步骤三(可选):从本地磁盘加载历史访问数据
	r.LoadingAndAutoSaveToDisc("example2", time.Second*10) //设置10秒备份一次(不填写则默认60秒备份一次)，备份到程序当前文件夹下，文件名为test1.ratelimit
	log.Println("性能测试正式开始")
	//步骤四：调用函数判断某用户是否允许访问
	/*
	   allow:= r.AllowVisit(user)
	*/
	//构建若干个用户，模拟用户访问
	var users = make(map[string]bool)
	for i := 1; i < 1000; i++ {
		users["user_"+strconv.Itoa(i)] = true
	}
	begin := time.Now()
	//模拟多个协程访问
	chanNum := 200
	var wg sync.WaitGroup
	wg.Add(chanNum)
	for i := 0; i < chanNum; i++ {
		go func(i int, wg *sync.WaitGroup) {
			for ii := 0; ii < 5000; ii++ {
				for user := range users {
					for {
						Visits++
						if !r.AllowVisit(user) {
							break
						}
					}
				}
			}
			wg.Done()
		}(i, &wg)
	}
	//所有线程结束，完工
	wg.Wait()
	t := int(time.Now().Sub(begin).Seconds())
	log.Println("性能测试完成:共计访问", Visits, "次,", "耗时", t, "秒,即每秒约完成", Visits/t, "次操作")
	//步骤五:程序退出前主动手动存盘
	err := r.SaveToDiscOnce() //在自动备份的同时，还支持手动备份，一般在程序要退出时调用此函数
	if err == nil {
		log.Println("完成手动数据备份")
	} else {
		log.Println(err)
	}
}

```
结果如下：
```
21:21:04 t.go:39: 允许访问,其剩余访问次数情况如下: [9]
21:21:04 t.go:39: 允许访问,其剩余访问次数情况如下: [8]
21:21:04 t.go:39: 允许访问,其剩余访问次数情况如下: [7]
21:21:04 t.go:39: 允许访问,其剩余访问次数情况如下: [6]
21:21:04 t.go:39: 允许访问,其剩余访问次数情况如下: [5]
21:21:04 t.go:39: 允许访问,其剩余访问次数情况如下: [4]
21:21:04 t.go:39: 允许访问,其剩余访问次数情况如下: [3]
21:21:04 t.go:39: 允许访问,其剩余访问次数情况如下: [2]
21:21:04 t.go:39: 允许访问,其剩余访问次数情况如下: [1]
21:21:04 t.go:39: 允许访问,其剩余访问次数情况如下: [0]
21:21:04 t.go:37: 访问量超出,其剩余访问次数情况如下: [0]
21:21:04 t.go:37: 访问量超出,其剩余访问次数情况如下: [0]
21:21:04 t.go:37: 访问量超出,其剩余访问次数情况如下: [0]
21:21:04 t.go:37: 访问量超出,其剩余访问次数情况如下: [0]
21:21:04 t.go:37: 访问量超出,其剩余访问次数情况如下: [0]
21:21:04 t.go:37: 访问量超出,其剩余访问次数情况如下: [0]
21:21:04 t.go:37: 访问量超出,其剩余访问次数情况如下: [0]
21:21:04 t.go:37: 访问量超出,其剩余访问次数情况如下: [0]
21:21:04 t.go:37: 访问量超出,其剩余访问次数情况如下: [0]
21:21:04 t.go:37: 访问量超出,其剩余访问次数情况如下: [0]
21:21:04 t.go:37: 访问量超出,其剩余访问次数情况如下: [0]

下面模拟单个用户持续访问的情况:
21:21:04 t.go:85: 17277777770 允许访问,其剩余访问次数情况如下: [19 99 999 9999]
21:21:04 t.go:85: 17277777770 允许访问,其剩余访问次数情况如下: [18 98 998 9998]
21:21:04 t.go:85: 17277777770 允许访问,其剩余访问次数情况如下: [17 97 997 9997]
21:21:04 t.go:85: 17277777770 允许访问,其剩余访问次数情况如下: [16 96 996 9996]
21:21:04 t.go:85: 17277777770 允许访问,其剩余访问次数情况如下: [15 95 995 9995]
21:21:04 t.go:85: 17277777770 允许访问,其剩余访问次数情况如下: [14 94 994 9994]
21:21:04 t.go:85: 17277777770 允许访问,其剩余访问次数情况如下: [13 93 993 9993]
21:21:04 t.go:85: 17277777770 允许访问,其剩余访问次数情况如下: [12 92 992 9992]
21:21:04 t.go:85: 17277777770 允许访问,其剩余访问次数情况如下: [11 91 991 9991]
21:21:04 t.go:85: 17277777770 允许访问,其剩余访问次数情况如下: [10 90 990 9990]
21:21:04 t.go:85: 17277777770 允许访问,其剩余访问次数情况如下: [9 89 989 9989]
21:21:04 t.go:85: 17277777770 允许访问,其剩余访问次数情况如下: [8 88 988 9988]
21:21:04 t.go:85: 17277777770 允许访问,其剩余访问次数情况如下: [7 87 987 9987]
21:21:04 t.go:85: 17277777770 允许访问,其剩余访问次数情况如下: [6 86 986 9986]
21:21:04 t.go:85: 17277777770 允许访问,其剩余访问次数情况如下: [5 85 985 9985]
21:21:04 t.go:85: 17277777770 允许访问,其剩余访问次数情况如下: [4 84 984 9984]
21:21:04 t.go:85: 17277777770 允许访问,其剩余访问次数情况如下: [3 83 983 9983]
21:21:04 t.go:85: 17277777770 允许访问,其剩余访问次数情况如下: [2 82 982 9982]
21:21:04 t.go:85: 17277777770 允许访问,其剩余访问次数情况如下: [1 81 981 9981]
21:21:04 t.go:85: 17277777770 允许访问,其剩余访问次数情况如下: [0 80 980 9980]
21:21:04 t.go:83: 17277777770 访问量超出,其剩余访问次数情况如下: [0 80 980 9980]
21:21:04 t.go:83: 17277777770 访问量超出,其剩余访问次数情况如下: [0 80 980 9980]
21:21:04 t.go:83: 17277777770 访问量超出,其剩余访问次数情况如下: [0 80 980 9980]
21:21:04 t.go:83: 17277777770 访问量超出,其剩余访问次数情况如下: [0 80 980 9980]
21:21:04 t.go:83: 17277777770 访问量超出,其剩余访问次数情况如下: [0 80 980 9980]
21:21:04 t.go:83: 17277777770 访问量超出,其剩余访问次数情况如下: [0 80 980 9980]
21:21:04 t.go:83: 17277777770 访问量超出,其剩余访问次数情况如下: [0 80 980 9980]
21:21:04 t.go:83: 17277777770 访问量超出,其剩余访问次数情况如下: [0 80 980 9980]
21:21:04 t.go:83: 17277777770 访问量超出,其剩余访问次数情况如下: [0 80 980 9980]
21:21:04 t.go:83: 17277777770 访问量超出,其剩余访问次数情况如下: [0 80 980 9980]
21:21:04 t.go:83: 17277777770 访问量超出,其剩余访问次数情况如下: [0 80 980 9980]

休息10秒后,该用户将恢复部分访问次数
21:21:14 t.go:95: 17277777770 允许访问,其剩余访问次数情况如下: [19 79 979 9979]

 [19 79 979 9979] 的具体含义为:
17277777770 在 1s 内共允许访问 20 次,剩余 19
17277777770 在 1m0s 内共允许访问 100 次,剩余 79
17277777770 在 1h0m0s 内共允许访问 1000 次,剩余 979
17277777770 在 24h0m0s 内共允许访问 10000 次,剩余 9979

手工清空 17277777770 后的访问记录
21:21:14 t.go:101: 17277777770 其剩余访问次数情况如下: [20 100 1000 10000]

下面为现在付费用户剩余访问次数情况:
21:21:14 t.go:123: [16277777770 9 89 989 9989]
21:21:14 t.go:123: [16277777771 9 89 989 9989]
21:21:14 t.go:123: [16277777772 9 89 989 9989]
21:21:14 t.go:123: [16277777773 9 89 989 9989]
21:21:14 t.go:123: [16277777774 9 89 989 9989]
21:21:14 t.go:123: [16277777775 9 89 989 9989]
21:21:14 t.go:123: [16277777776 9 89 989 9989]
21:21:14 t.go:123: [17277777770 20 100 1000 10000]

下面为现在免费用户剩余访问次数情况:
21:21:14 t.go:128: [16277777770 0 8 98 998]
21:21:14 t.go:128: [16277777771 0 8 98 998]
21:21:14 t.go:128: [16277777772 0 8 98 998]
21:21:14 t.go:128: [16277777773 0 8 98 998]
21:21:14 t.go:128: [16277777774 0 8 98 998]
21:21:14 t.go:128: [16277777775 0 8 98 998]
21:21:14 t.go:128: [16277777776 0 8 98 998]

下面为现为匿名S用户剩余访问次数情况:
21:21:14 t.go:133: [192.168.0.2 0 98]
21:21:14 t.go:133: [192.168.0.3 0 98]
21:21:14 t.go:133: [192.168.0.4 0 98]
21:21:14 t.go:133: [192.168.0.5 0 98]
21:21:14 t.go:133: [192.168.0.6 0 98]
21:21:14 t.go:133: [192.168.0.7 0 98]
21:21:14 t.go:133: [192.168.0.8 0 98]

性能测试，预计耗时1分钟，请耐心等待:
21:21:14 t.go:154: 性能测试正式开始
21:22:33 t.go:187: 性能测试完成:共计访问 954102785 次, 耗时 78 秒,即每秒约完成 12232086 次操作
21:22:33 t.go:191: 完成手动数据备份
```
