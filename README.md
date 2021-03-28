**github.com/yudeguang/ratelimit** 
    在网站的运营中，经常会遇到需要对用户访问次数做限制的情况，比如非常典型的是对于某些付费访问服务，需要对访问频率做比较精确的限制，比如单个用户每天只允许访问多少次，然后每小时只允许访问多少次等等，ratelimit就是针对这种情况而设计。
    ratelimit底层用环形队列来存储用户访问数据，拥有较高的性能。作为对比，与用redis的相关数据结构来实现用户访问控制相比，其用法相对简单。而诸如诸如 golang.org/x/time/rate 一般只用于超高频率的以秒为单位的用户访问频率控制，适用范围不尽相同。

##使用案例如下
```go
package main

import (
	"fmt"
	"github.com/yudeguang/ratelimit"
	"log"
	"time"
)

func main() {
	log.SetFlags(log.Lshortfile | log.Ltime)
	//初始化若干条访问控制规则
	//每10秒只允许访问5次
	//每30分钟只允许访问50次
	//每天只允许访问500次
	r := ratelimit.NewRule()
	r.AddRule(time.Second*10, 5)
	r.AddRule(time.Minute*30, 50)
	r.AddRule(time.Hour*24, 500)
	//默认10秒备份一次，备份到程序当前文件夹下，文件名为test.ratelimit
	r.LoadingAndAutoSaveToDisc("test", time.Second*10)
	//构建若干个用户，模拟用户访问
	users := []string{"andy", "小余", "130x"}
	for _, user := range users {
		fmt.Println("\r\n开始模拟以下用户访问:", user)
		for {
			if r.AllowVisit(user) {
				log.Println(user, "访问1次,剩余:", r.RemainingVisits(user))
			} else {
				log.Println(user, "访问过多,稍后再试")
				break
			}
			time.Sleep(time.Second * 1)
		}
	}

	//打印所有用户访问数据情况
	fmt.Println("\r\n开始打印所有用户在相关时间段内详细的剩余访问次数情况:\r\n")
	for _, user := range users {
		fmt.Println(user)
		fmt.Println("     概述:", r.RemainingVisits(user))
		fmt.Println("     具体:")
		r.PrintRemainingVisits(user)
		fmt.Println("")
	}

}
```
结果如下：
```

开始模拟以下用户访问: andy
22:10:51 rate.go:26: andy 访问1次,剩余: [4 49 499]
22:10:52 rate.go:26: andy 访问1次,剩余: [3 48 498]
22:10:53 rate.go:26: andy 访问1次,剩余: [2 47 497]
22:10:54 rate.go:26: andy 访问1次,剩余: [1 46 496]
22:10:55 rate.go:26: andy 访问1次,剩余: [0 45 495]
22:10:56 rate.go:28: andy 访问过多,稍后再试

开始模拟以下用户访问: 小余
22:10:56 rate.go:26: 小余 访问1次,剩余: [4 49 499]
22:10:57 rate.go:26: 小余 访问1次,剩余: [3 48 498]
22:10:58 rate.go:26: 小余 访问1次,剩余: [2 47 497]
22:10:59 rate.go:26: 小余 访问1次,剩余: [1 46 496]
22:11:00 rate.go:26: 小余 访问1次,剩余: [0 45 495]
22:11:01 rate.go:28: 小余 访问过多,稍后再试

开始模拟以下用户访问: 130x
22:11:01 rate.go:26: 130x 访问1次,剩余: [4 49 499]
22:11:02 rate.go:26: 130x 访问1次,剩余: [3 48 498]
22:11:03 rate.go:26: 130x 访问1次,剩余: [2 47 497]
22:11:04 rate.go:26: 130x 访问1次,剩余: [1 46 496]
22:11:05 rate.go:26: 130x 访问1次,剩余: [0 45 495]
22:11:06 rate.go:28: 130x 访问过多,稍后再试

开始打印所有用户在相关时间段内详细的剩余访问次数情况:

andy
     概述: [5 45 495]
     具体:
andy 在 10s 内共允许访问 5 次,剩余 5
andy 在 30m0s 内共允许访问 50 次,剩余 45
andy 在 24h0m0s 内共允许访问 500 次,剩余 495

小余
     概述: [1 45 495]
     具体:
小余 在 10s 内共允许访问 5 次,剩余 1
小余 在 30m0s 内共允许访问 50 次,剩余 45
小余 在 24h0m0s 内共允许访问 500 次,剩余 495

130x
     概述: [0 45 495]
     具体:
130x 在 10s 内共允许访问 5 次,剩余 0
130x 在 30m0s 内共允许访问 50 次,剩余 45
130x 在 24h0m0s 内共允许访问 500 次,剩余 495
```


------------
    可以看到，在经过一段时间后，andy，小余在10S内的剩余访问次数逐步恢复
