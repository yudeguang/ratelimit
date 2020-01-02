package ratelimit

import (
	"fmt"
	"math/big"
	"net"
)

//把Int64转换成IP4的的字符串形式
func Int64ToIp4String(ip int64) string {
	return fmt.Sprintf("%d.%d.%d.%d", byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip))
}

//IP4地址转换为Int64，方便存储时，减少内存占用，提升性能
func Ip4StringToInt64(ip string) int64 {
	ret := big.NewInt(0)
	ret.SetBytes(net.ParseIP(ip).To4())
	return ret.Int64()
}
