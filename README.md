 
在网站的运营中，经常会遇到需要对用户访问次数做限制的情况，比如非常典型的是对于某些付费访问服务，需要对访问频率做比较精确的限制，比如单个用户(或者每个IP)地址每天只允许访问多少次，然后每小时只允许访问多少次等等，ratelimit就是针对这种情况而设计。
    
不同于网关级限流(包括go.uber.org/ratelimit漏桶限流以及github.com/juju/ratelimit令牌桶限流),本限流方案为业务级限流，适用于平台运营中,精细化的按单个用户,按IP等限流,为业内rdeis滑动窗口限流方案的纯GO替代方案,并且支持持久化,可定期把历史数据备份到本地磁盘,程序重启也可保留之前的访问记录。
      
github.com/yudeguang/ratelimit底层用一个大小能自动伸缩的环形队列来存储用户访问数据，并发安全，拥有较高性能的同时还非常省内存,同时拥有高达1000W次/秒的处理能力(redis约10W次/秒)。作为对比，与用redis的相关数据结构来实现用户访问控制相比，其用法相对简单。 
    
    

##使用案例如下
```go
package main

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/yudeguang/ratelimit"
)

func main() {
	log.SetFlags(log.Lshortfile | log.Ltime)
	test1()
	test2()
}

//模拟1000个用户，累计进行总共约10亿次访问测试
func test1() {
	var Visits int //因并发问题num比实际数量稍小
	fmt.Println("\r\n测试1,性能测试，预计耗时1分钟，请耐心等待:")
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
	r.LoadingAndAutoSaveToDisc("test1", time.Second*10) //设置10秒备份一次(不填写则默认60秒备份一次)，备份到程序当前文件夹下，文件名为test1.ratelimit
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

//模拟用户访问并打印
func test2() {
	fmt.Println("\r\n测试2，模拟用户访问并打印:")
	//步骤一：初始化
	r := ratelimit.NewRule()
	//步骤二：增加一条或者多条规则组成复合规则，规则必须至少包含一条规则
	r.AddRule(time.Second*10, 5)  //每10秒只允许访问5次
	r.AddRule(time.Minute*30, 50) //每30分钟只允许访问50次
	r.AddRule(time.Hour*24, 500)  //每天只允许访问500次
	//步骤三：调用函数判断某用户是否允许访问
	/*
	   allow:= r.AllowVisit(user)
	*/
	//构建若干个用户，模拟用户访问
	users := []string{"andyyu", "tony", "chery"}
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

测试1,性能测试，预计耗时1分钟，请耐心等待:
14:16:00 t.go:35: 性能测试正式开始
14:17:14 t.go:68: 性能测试完成:共计访问 936794614 次, 耗时 73 秒,即每秒约完成 12832802 次操作
14:17:14 t.go:72: 完成手动数据备份

测试2，模拟用户访问并打印:

开始模拟以下用户访问: andyyu
14:17:14 t.go:97: andyyu 访问1次,剩余: [4 49 499]
14:17:15 t.go:97: andyyu 访问1次,剩余: [3 48 498]
14:17:16 t.go:97: andyyu 访问1次,剩余: [2 47 497]
14:17:17 t.go:97: andyyu 访问1次,剩余: [1 46 496]
14:17:18 t.go:97: andyyu 访问1次,剩余: [0 45 495]
14:17:19 t.go:99: andyyu 访问过多,稍后再试

开始模拟以下用户访问: tony
14:17:19 t.go:97: tony 访问1次,剩余: [4 49 499]
14:17:20 t.go:97: tony 访问1次,剩余: [3 48 498]
14:17:21 t.go:97: tony 访问1次,剩余: [2 47 497]
14:17:22 t.go:97: tony 访问1次,剩余: [1 46 496]
14:17:23 t.go:97: tony 访问1次,剩余: [0 45 495]
14:17:24 t.go:99: tony 访问过多,稍后再试

开始模拟以下用户访问: chery
14:17:24 t.go:97: chery 访问1次,剩余: [4 49 499]
14:17:25 t.go:97: chery 访问1次,剩余: [3 48 498]
14:17:26 t.go:97: chery 访问1次,剩余: [2 47 497]
14:17:27 t.go:97: chery 访问1次,剩余: [1 46 496]
14:17:28 t.go:97: chery 访问1次,剩余: [0 45 495]
14:17:29 t.go:99: chery 访问过多,稍后再试

开始打印所有用户在相关时间段内详细的剩余访问次数情况:

andyyu
     概述: [5 45 495]
     具体:
andyyu 在 10s 内共允许访问 5 次,剩余 5
andyyu 在 30m0s 内共允许访问 50 次,剩余 45
andyyu 在 24h0m0s 内共允许访问 500 次,剩余 495

tony
     概述: [0 45 495]
     具体:
tony 在 10s 内共允许访问 5 次,剩余 0
tony 在 30m0s 内共允许访问 50 次,剩余 45
tony 在 24h0m0s 内共允许访问 500 次,剩余 495

chery
     概述: [0 45 495]
     具体:
chery 在 10s 内共允许访问 5 次,剩余 0
chery 在 30m0s 内共允许访问 50 次,剩余 45
chery 在 24h0m0s 内共允许访问 500 次,剩余 495
```


------------
    可以看到，在经过一段时间后，andyyu在10S内的剩余访问次数逐步恢复
