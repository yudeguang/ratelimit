**github.com/yudeguang/ratelimit** 
用于对用户访问频率进行控制，同时支持一条或者多条用户访问控制策略。
比如 每10秒只允许访问5次，每30分钟只允许访问50次,每天只允许访问500次。
另外，ratelimit 用环形队列做为底层数据结构来存储用户访问数据，同时还针对于该队列做了缓存机制，所以有较高的性能

##测试代码如下
```go
package main

import (
    "github.com/yudeguang/ratelimit"
    "log"
    "time"
)

func main() {
    log.SetFlags(log.Lshortfile | log.Ltime)
    /*
        初始化若干条访问控制规则 每10秒只允许访问5次，每30分钟只允许访问50次,每天只允许访问500次
    */
    r := ratelimit.NewRule()
    r.AddRule(time.Second*10, 5)
    r.AddRule(time.Minute*30, 50)
    r.AddRule(time.Hour*24, 500)
    //模拟用户访问
    for {
        user := "小余"
        if r.AllowVisit(user) {
            log.Println(user, "访问一次，目前在相关时间段内还允许访问次数分别如下:", r.RemainingVisits(user))
        } else {
            log.Println(user, "访问数量过大，已经不允许访问，请过一段时间再试")
            break
        }
        time.Sleep(time.Second * 1)
    }
    log.Println("...............")
    for {
        user := "小明"
        if r.AllowVisit(user) {
            log.Println(user, "访问一次，目前在相关时间段内还允许访问次数分别如下:", r.RemainingVisits(user))
        } else {
            log.Println(user, "访问数量过大，已经不允许访问，请过一段时间再试")
            break
        }
        time.Sleep(time.Second * 1)
    }
    log.Println("...............")
    //模拟IP作为用户名访问
    for {
        ip := "127.0.0.1"
        if r.AllowVisitByIP4(ip) {
            log.Println(ip, "访问一次，目前在相关时间段内还允许访问次数分别如下:", r.RemainingVisitsByIP4(ip))
        } else {
            log.Println(ip, "访问数量过大，已经不允许访问，请过一段时间再试")
            break
        }
        time.Sleep(time.Second * 1)
    }
    log.Println("...............")
    //用户统计
    users := r.GetCurOnlineUsers()
    log.Println("目前所有在线用户如下:", users)
    log.Println("...............")
    //查看每个用户分别剩余的访问次数，注意如果用户是用IP4转化为INT64存储的，要用RemainingVisitsByIP4方法读取
    for _, user := range users {
        if ratelimit.IsIP(user) {
            log.Println(user, "目前在相关时间段内还允许访问次数分别如下:", r.RemainingVisitsByIP4(user))
        } else {
            log.Println(user, "目前在相关时间段内还允许访问次数分别如下:", r.RemainingVisits(user))
        }
    }
}
```