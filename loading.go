// Copyright 2020 rateLimit Author(https://github.com/yudeguang/ratelimit). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/yudeguang/ratelimit.

package ratelimit

import (
	"fmt"
	"github.com/yudeguang/iox"
	"io/ioutil"
	"strconv"
	"time"
)

//从本地磁盘加载历史数据
func (r *Rule) loading() (err error) {
	var errFileDifferent = fmt.Errorf("backup rules is inconsistent with current rules")
	b, err := ioutil.ReadFile(r.backupFileName + ".ratelimit")
	if err != nil {
		return fmt.Errorf("Open backup file fail," + err.Error())
	}
	rs := iox.NewFromBytes(b)
	rulesNum, err := rs.ReadUint64()
	if err != nil {
		return err
	}
	//1 判断规则数量是否一致
	if int(rulesNum) != len(r.rules) {
		return errFileDifferent
	}
	for i := range r.rules {
		now := time.Now().Add(r.rules[i].defaultExpiration).UnixNano()
		//2 判断单条规则的下标一致
		curIndex, err := rs.ReadUint64()
		if err != nil {
			return err
		}
		if i != int(curIndex) {
			return errFileDifferent
		}

		curRuleKeyNum, err := rs.ReadUint64()
		if err != nil {
			return err
		}
		//有可能某条规则下面暂时没有历史记录
		if int(curRuleKeyNum) > 0 {
			for ii := 0; ii < int(curRuleKeyNum); ii++ {
				//获取键类型,以下每种类型对应存盘时的相应定义
				KeyType, err := rs.ReadUint8()
				if err == nil {
					var key interface{}
					var tempKey uint64
					switch KeyType {
					// string
					case 0:
						key, err = rs.ReadStringUint64()
					case 1:
						tempKey, err = rs.ReadUint64()
						key = int(tempKey)
					case 2:
						tempKey, err = rs.ReadUint64()
						key = int8(tempKey)
					case 3:
						tempKey, err = rs.ReadUint64()
						key = int16(tempKey)
					//int32
					case 4:
						tempKey, err = rs.ReadUint64()
						key = int32(tempKey)
					case 5:
						tempKey, err = rs.ReadUint64()
						key = int64(tempKey)
					case 6:
						tempKey, err = rs.ReadUint64()
						key = uint(tempKey)
					case 7:
						tempKey, err = rs.ReadUint64()
						key = uint8(tempKey)
					case 8:
						tempKey, err = rs.ReadUint64()
						key = uint16(tempKey)
					case 9:
						tempKey, err = rs.ReadUint64()
						key = uint32(tempKey)
					case 10:
						tempKey, err = rs.ReadUint64()
						key = tempKey
					default:
						return errFileDifferent
					}
					if err != nil {
						return err
					}
					if _, exist := r.rules[i].usedVisitorRecordsIndex.Load(key); exist {
						panic("The function LoadingAndAutoSaveToDisc can only be called when the program is initialized,and can only be called once.")
					}
					curKeyRecordsNum, err := rs.ReadUint64()
					if err != nil {
						return err
					}
					var preRecord int64
					for iii := 0; iii < int(curKeyRecordsNum); iii++ {
						record, err := rs.ReadUint64()
						if err != nil {
							return err
						}
						curRecord := int64(record)
						//curRecord必须依次变大，否则不合法,另外,curRecord的值也不能太大，大过当前时间加上过期时间
						location, _ := rs.CurPos()
						if curRecord < preRecord || curRecord > now {
							return fmt.Errorf("The backup file has been illegally modified and has become invalid,the location is:" + strconv.Itoa(int(location)))
						}
						err = r.rules[i].addFromBackUpFile(key, int64(record))
						if err != nil {
							return err
						}
						preRecord = curRecord
					}
				}
			}
		}
	}
	return nil
}
