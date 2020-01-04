package ratelimit

import (
	"github.com/yudeguang/hashset"
	"log"
	"sync"
	"time"
)

//单组用户访问控制策略
type singleRule struct {
	defaultExpiration            time.Duration       //表示在某个时间段内,也是每条访问记录需要保存的时长，也就是过期时间
	cleanupInterval              time.Duration       //默认多长时间需要执行一次清除操作
	numberOfAllowedAccesses      int                 //每个用户在相应时间段内最多允许访问的次数
	estimatedNumberOfOnlineUsers int                 //单位时间内预计有多少个用户会访问网站，建议选用一个稍大于实际值的值，以减少内存分配次数
	indexes                      sync.Map            //索引：key代表用户名或IP；value代表visitorRecords中的索引位置
	visitorRecords               []*circleQueueInt64 //存储用户访问记录
	notUsedVisitorRecordsIndex   *hashset.SetInt     //对应visitorRecords中未使用的数据的索引位置
	lock                         *sync.Mutex         //锁
}

/*
初始化一个条单组用户访问控制控制策略,例：
vc := newsingleRule(time.Minute*30, 50)
或者 vc := newsingleRule(time.Minute*30, 50, 1000)
它表示:
在30分钟内每个用户最多允许访问50次,并且我们预计在这30分钟内大致有1000个用户会访问我们的网站
1000为可选字段，此参数可默认不填写，主要是用于提升性能，类似于声明切片时的cap,绝大部分情况下无需关注此参数。
*/
func newsingleRule(defaultExpiration time.Duration, numberOfAllowedAccesses int, estimatedNumberOfOnlineUserNum ...int) *singleRule {
	/*
		对于默认过期时间defaultExpiration,如果小于1秒，从效率的角度讲，
		整个算法实际上可以衰退为令牌桶算法golang.org/x/time/rate,以应
		对超高并发的情况，在此并不实现。
	*/
	estimatedNumberOfOnlineUsers := 0
	if len(estimatedNumberOfOnlineUserNum) > 0 {
		estimatedNumberOfOnlineUsers = estimatedNumberOfOnlineUserNum[0]
	}
	//设立默认清除过期数据的间隔,用于清除过期数据,并定期清理内存
	cleanupInterval := defaultExpiration / 100
	if cleanupInterval < time.Nanosecond*1 {
		cleanupInterval = time.Nanosecond * 1
	}
	vc := createsingleRule(defaultExpiration, cleanupInterval, numberOfAllowedAccesses, estimatedNumberOfOnlineUsers)
	go vc.deleteExpired()
	return vc
}

func createsingleRule(defaultExpiration, cleanupInterval time.Duration, numberOfAllowedAccesses, estimatedNumberOfOnlineUsers int) *singleRule {
	//规范化输入参数
	if numberOfAllowedAccesses <= 0 {
		log.Println("请检察参数numberOfAllowedAccesses设置是否合理，在此被强行修改为1")
		numberOfAllowedAccesses = 1
	}
	if estimatedNumberOfOnlineUsers <= 0 {
		log.Println("请检察参数estimatedNumberOfOnlineUsers设置是否合理，在此被强行修改为1")
		estimatedNumberOfOnlineUsers = 1
	}
	var vc singleRule
	var lock sync.Mutex
	vc.defaultExpiration = defaultExpiration
	vc.cleanupInterval = cleanupInterval
	vc.numberOfAllowedAccesses = numberOfAllowedAccesses
	vc.estimatedNumberOfOnlineUsers = estimatedNumberOfOnlineUsers
	vc.notUsedVisitorRecordsIndex = hashset.NewInt()
	vc.lock = &lock
	//根据在线用户数量初始化用户访问记录数据
	vc.visitorRecords = make([]*circleQueueInt64, vc.estimatedNumberOfOnlineUsers)
	for i := range vc.visitorRecords {
		vc.visitorRecords[i] = newCircleQueueInt64(vc.numberOfAllowedAccesses)
		vc.notUsedVisitorRecordsIndex.Add(i)
	}
	return &vc

}

//是否允许访问,允许访问则往访问记录中加入一条访问记录
func (this *singleRule) AllowVisit(key interface{}) bool {
	return this.add(key) == nil
}

//是否允许某IP的用户访问
func (this *singleRule) AllowVisitByIP(ip string) bool {
	ipInt64 := ip4StringToInt64(ip)
	if ipInt64 == 0 {
		return false
	}
	return this.AllowVisit(ipInt64)
}

//剩余访问次数
func (this *singleRule) RemainingVisits(key interface{}) int {
	//先前曾经有访问记录，则取剩余空间长度。
	if index, exist := this.indexes.Load(key); exist {
		this.visitorRecords[index.(int)].DeleteExpired()
		return this.visitorRecords[index.(int)].UnUsedSize()
	}
	//若不存在，就取numberOfAllowedAccesses
	return this.numberOfAllowedAccesses
}

//某IP剩余访问次数
func (this *singleRule) RemainingVisitsIP(ip string) int {
	ipInt64 := ip4StringToInt64(ip)
	if ipInt64 == 0 {
		return 0
	}
	return this.RemainingVisits(ipInt64)
}

//增加一条访问记录
func (this *singleRule) add(key interface{}) (err error) {
	this.lock.Lock()
	defer this.lock.Unlock()
	//存在某访客，则在该访客记录中增加一条访问记录
	if index, exist := this.indexes.Load(key); exist {
		this.visitorRecords[index.(int)].DeleteExpired()
		return this.visitorRecords[index.(int)].Push(time.Now().Add(this.defaultExpiration).UnixNano())
	}
	//该访客在这一段时间从来未出现过
	//在visitorRecords中有未使用的空间时,根据notUsedVisitorRecordsIndex随机取一条出来使用
	if this.notUsedVisitorRecordsIndex.Size() > 0 {
		for index := range this.notUsedVisitorRecordsIndex.Items {
			this.notUsedVisitorRecordsIndex.Remove(index)
			this.indexes.Store(key, index)
			return this.visitorRecords[index].Push(time.Now().Add(this.defaultExpiration).UnixNano())
		}
	}
	//visitorRecords没有空余空间时，则需要插入一条新数据到visitorRecords中
	queue := newCircleQueueInt64(this.numberOfAllowedAccesses)
	this.visitorRecords = append(this.visitorRecords, queue)
	index := len(this.visitorRecords) - 1 //最后一条的位置即为新的索引位置
	this.indexes.Store(key, index)
	return this.visitorRecords[index].Push(time.Now().Add(this.defaultExpiration).UnixNano())
}

//删除过期数据
func (this *singleRule) deleteExpired() {
	finished := true
	for range time.Tick(this.cleanupInterval) {
		//如果数据量较大，那么在一个清除周期内不一定会把所有数据全部清除,所以要判断上一轮次的清除是否完成
		if finished {
			finished = false
			this.deleteExpiredOnce()
			this.gc() //回收空间
			finished = true
		}
	}
}

//在特定时间间隔内执行一次删除过期数据操作
func (this *singleRule) deleteExpiredOnce() {
	this.indexes.Range(func(k, v interface{}) bool {
		this.lock.Lock() //range里面不能用defer
		index := v.(int)
		//防止越界出错，理论上不存在这种情况
		if index < len(this.visitorRecords) && index >= 0 {
			this.visitorRecords[index].DeleteExpired()
			//删除完过期数据之后，如果该用户的所有访问记录均过期了，那么就删除该用户
			//并把该空间返还给notUsedVisitorRecordsIndex以便下次重复使用
			if this.visitorRecords[index].UsedSize() == 0 {
				this.indexes.Delete(k)
				this.notUsedVisitorRecordsIndex.Add(index)
			}
		} else {
			this.indexes.Delete(k)
		}
		this.lock.Unlock()
		return true
	})
}

/*
回收未使用的空间
GC的目的在于，防止出现访问峰值之后，实际的访问峰值远大于我们假想的峰值estimatedNumberOfOnlineUsers
之后用户数又大幅下降，这时候，为了减少内存占用，需要进行数据回收操作，重新分配空间。
*/
func (this *singleRule) gc() {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.needGc() {
		curLen := len(this.visitorRecords)
		unUsedLen := len(this.notUsedVisitorRecordsIndex.Items)
		usedLen := curLen - unUsedLen
		//算出新的visitorRecords长度
		var newLen int
		if usedLen < this.estimatedNumberOfOnlineUsers {
			newLen = this.estimatedNumberOfOnlineUsers
		} else {
			newLen = usedLen * 2
		}
		//根据新长度，建立新的用户访问记录
		visitorRecordsNew := make([]*circleQueueInt64, newLen)
		for i := range visitorRecordsNew {
			visitorRecordsNew[i] = newCircleQueueInt64(this.numberOfAllowedAccesses)
		}
		//清空未使用索引notUsedVisitorRecordsIndex
		this.notUsedVisitorRecordsIndex.Clear()
		//重建索引indexes
		indexNew := 0
		this.indexes.Range(func(k, v interface{}) bool {
			indexOld := v.(int)
			visitorRecordsNew[indexNew] = this.visitorRecords[indexOld]
			indexNew++
			return true
		})
		this.visitorRecords = visitorRecordsNew
		//重建未使用索引notUsedVisitorRecordsIndex
		for i := range this.visitorRecords {
			if i >= indexNew {
				this.notUsedVisitorRecordsIndex.Add(i)
			}
		}
	}
}

/*
是否需要进行数据清理
如果visitorRecords数据空的太多,则需要进行清理操作
并且长度远大于默认在线用户数量，则需要进行GC操作
*/
func (this *singleRule) needGc() bool {
	curLen := len(this.visitorRecords)
	unUsedLen := len(this.notUsedVisitorRecordsIndex.Items)
	usedLen := curLen - unUsedLen
	//log.Println("总:", curLen, "已用:", usedLen, "未使用:", unUsedLen)
	//比预期的少，我们就不回收了
	if curLen < 2*this.estimatedNumberOfOnlineUsers {
		return false
	}
	//未使用的太多，则需要回收
	if usedLen*2 < unUsedLen {
		return true
	}
	return false
}
