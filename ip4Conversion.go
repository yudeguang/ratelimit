// Copyright 2020 ratelimit Author(https://github.com/yudeguang/ratelimit). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/yudeguang/ratelimit.
package ratelimit

import (
	"fmt"
	"math/big"
	"net"
)

/*
IP4与int64之间的转换
*/
//把Int64转换成IP4的的字符串形式
func int64ToIp4String(ip int64) string {
	return fmt.Sprintf("%d.%d.%d.%d", byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip))
}

//IP4地址转换为Int64，方便存储时，减少内存占用，提升性能
func ip4StringToInt64(ip string) int64 {
	ret := big.NewInt(0)
	ret.SetBytes(net.ParseIP(ip).To4())
	return ret.Int64()
}

//判断是否是IP地址，同时支持IP4,IP6
func IsIP(ip string) bool {
	address := net.ParseIP(ip)
	if address == nil {
		return false
	}
	return true
}
