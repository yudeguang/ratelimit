**github.com/yudeguang/ratelimit** 
    在网站的运营中，经常会遇到需要对用户访问次数做限制的情况，比如非常典型的是对于某些付费访问服务，需要对访问频率做比较精确的限制，比如单个用户(或者每个IP)地址每天只允许访问多少次，然后每小时只允许访问多少次等等，ratelimit就是针对这种情况而设计。
    不同于go.uber.org/ratelimit 漏桶限流以及 github.com/juju/ratelimit 令牌桶限流,本限流方案适用于平台运营中,精细化的按单个用户,按IP等限流,为业内rdeis滑动窗口限流方案的纯GO替代方案,并且支持持久化,可定期把历史数据备份到本地磁盘,程序重启也可保留之前的访问记录。
    ratelimit底层用环形队列来存储用户访问数据，拥有较高的性能,每秒约能处理250W次。作为对比，与用redis的相关数据结构来实现用户访问控制相比，其用法相对简单。 

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

var num int //因并发问题num比实际数量小

func main() {
	log.SetFlags(log.Lshortfile | log.Ltime)
	test1()
	test2()
}

func test1() {
	fmt.Println("\r\n测试1,加载备份数据并自动定期备份以及性能测试:")
	//初始化访问规则，支持用多个规则形成的复杂规则，规则必须至少包含一条规则
	//支持从本地磁盘加载备份好的历史记录，并支持定期自动备份
	//注意下面的单条规则中，不宜设定监控时间段过大的规则，比如设定监控某个用户一个月允许访问200W次(r.AddRule(time.Hour*24*30, 2000000))，那么在这种规则下，内存会超出，启动会失败
	r := ratelimit.NewRule()
	r.AddRule(time.Second*10, 5)                               //每10秒只允许访问5次
	r.AddRule(time.Minute*30, 50)                              //每30分钟只允许访问50次
	r.AddRule(time.Hour*24, 500)                               //每天只允许访问500次
	err := r.LoadingAndAutoSaveToDisc("test1", time.Second*10) //设置10秒备份一次(不填写则默认60秒备份一次)，备份到程序当前文件夹下，文件名为test1.ratelimit
	if err == nil {
		log.Println("加载历史访问记录成功")
	} else {
		log.Println(err)
	}
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
			for ii := 0; ii < 50; ii++ {
				for user := range users {
					for {
						if r.AllowVisit(user) {
							num++
						} else {
							num++
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

	log.Println("性能测试完成，每秒约完成:", num/int(time.Now().Sub(begin).Seconds()), "次操作")
	err = r.SaveToDiscOnce("test2") //在自动备份的同时，还支持手动备份，一般在程序要退出时调用此函数
	if err == nil {
		log.Println("完成手动数据备份")
	} else {
		log.Println(err)
	}
}
func test2() {
	fmt.Println("\r\n测试2，模拟用户访问并打印:")
	r := ratelimit.NewRule()
	r.AddRule(time.Second*10, 5)                        //每10秒只允许访问5次
	r.AddRule(time.Minute*30, 50)                       //每30分钟只允许访问50次
	r.AddRule(time.Hour*24, 500)                        //每天只允许访问500次
	r.LoadingAndAutoSaveToDisc("test2", time.Second*10) //设置10秒备份一次(不填写则默认60秒备份一次)，备份到程序当前文件夹下，文件名为test2.ratelimit
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

测试1,加载备份数据并自动定期备份以及性能测试:
22:45:27 t.go:31: 加载历史访问记录成功
22:45:31 t.go:65: 性能测试完成，每秒约完成: 2495806 次操作
22:45:31 t.go:68: 完成手动数据备份

测试2，模拟用户访问并打印:

开始模拟以下用户访问: andy
22:45:31 t.go:86: andy 访问1次,剩余: [4 49 499]
22:45:32 t.go:86: andy 访问1次,剩余: [3 48 498]
22:45:33 t.go:86: andy 访问1次,剩余: [2 47 497]
22:45:34 t.go:86: andy 访问1次,剩余: [1 46 496]
22:45:35 t.go:86: andy 访问1次,剩余: [0 45 495]
22:45:36 t.go:88: andy 访问过多,稍后再试

开始模拟以下用户访问: 小余
22:45:36 t.go:86: 小余 访问1次,剩余: [4 49 499]
22:45:37 t.go:86: 小余 访问1次,剩余: [4 48 498]
22:45:38 t.go:86: 小余 访问1次,剩余: [3 47 497]
22:45:39 t.go:86: 小余 访问1次,剩余: [2 46 496]
22:45:40 t.go:86: 小余 访问1次,剩余: [1 45 495]
22:45:41 t.go:86: 小余 访问1次,剩余: [0 44 494]
22:45:42 t.go:88: 小余 访问过多,稍后再试

开始模拟以下用户访问: 130x
22:45:42 t.go:86: 130x 访问1次,剩余: [4 49 499]
22:45:43 t.go:86: 130x 访问1次,剩余: [3 48 498]
22:45:44 t.go:86: 130x 访问1次,剩余: [2 47 497]
22:45:45 t.go:86: 130x 访问1次,剩余: [1 46 496]
22:45:46 t.go:86: 130x 访问1次,剩余: [0 45 495]
22:45:47 t.go:88: 130x 访问过多,稍后再试

开始打印所有用户在相关时间段内详细的剩余访问次数情况:

andy
     概述: [5 45 495]
     具体:
andy 在 10s 内共允许访问 5 次,剩余 5
andy 在 30m0s 内共允许访问 50 次,剩余 45
andy 在 24h0m0s 内共允许访问 500 次,剩余 495

小余
     概述: [1 44 494]
     具体:
小余 在 10s 内共允许访问 5 次,剩余 1
小余 在 30m0s 内共允许访问 50 次,剩余 44
小余 在 24h0m0s 内共允许访问 500 次,剩余 494

130x
     概述: [0 45 495]
     具体:
130x 在 10s 内共允许访问 5 次,剩余 0
130x 在 30m0s 内共允许访问 50 次,剩余 45
130x 在 24h0m0s 内共允许访问 500 次,剩余 495
```


------------
    可以看到，在经过一段时间后，andy，小余在10S内的剩余访问次数逐步恢复
