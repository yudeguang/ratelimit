// Copyright 2020 ratelimit Author(https://github.com/yudeguang/ratelimit). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/yudeguang/ratelimit.

package ratelimit

import (
	"fmt"
	"io/ioutil"

	"github.com/yudeguang/iox"
)

func (this *Rule) loading() (err error) {
	var errFileDifferent = fmt.Errorf("backup rules is inconsistent with current rules")
	b, err := ioutil.ReadFile(this.backupFileName + ".ratelimit")
	if err != nil {
		return err
	}

	r := iox.NewFromBytes(b)
	rulesNum, err := r.ReadUint64()
	if err != nil {
		return err
	}
	//判断规则数量是否一致
	if int(rulesNum) != len(this.rules) {
		return errFileDifferent
	}

	for i := range this.rules {
		//判断单条规则的下标一致
		cur_index, err := r.ReadUint64()
		if err != nil {
			return err
		}
		if i != int(cur_index) {
			return errFileDifferent
		}

		curRuleKeyNum, err := r.ReadUint64()
		if err != nil {
			return err
		}

		//有可能某条规则下面暂时没有历史记录
		if int(curRuleKeyNum) != 0 {
			for ii := 0; ii < int(curRuleKeyNum); ii++ {
				//获取键类型
				KeyType, err := r.ReadUint8()
				if err == nil {
					var key interface{}
					var tempKey uint64
					switch KeyType {
					case 0: // string
						key, err = r.ReadStringUint64()
					case 1:
						tempKey, err = r.ReadUint64()
						key = int(tempKey)
					case 2:
						tempKey, err = r.ReadUint64()
						key = int8(tempKey)

					case 3:
						tempKey, err = r.ReadUint64()
						key = int16(tempKey)

					case 4: //int32
						tempKey, err = r.ReadUint64()
						key = int32(tempKey)

					case 5:
						tempKey, err = r.ReadUint64()
						key = int64(tempKey)

					case 6:
						tempKey, err = r.ReadUint64()
						key = uint(tempKey)
					case 7:
						tempKey, err = r.ReadUint64()
						key = uint8(tempKey)

					case 8:
						tempKey, err = r.ReadUint64()
						key = uint16(tempKey)

					case 9:
						tempKey, err = r.ReadUint64()
						key = uint32(tempKey)
					case 10:
						tempKey, err = r.ReadUint64()
						key = uint64(tempKey)
					default:
						return errFileDifferent
					}
					if err != nil {
						return err
					}
					curKeyRecordsNum, err := r.ReadUint64()
					if err != nil {
						return err
					}
					for iii := 0; iii < int(curKeyRecordsNum); iii++ {
						record, err := r.ReadUint64()
						if err != nil {
							return err
						}
						if err == nil {
							this.rules[i].add(key, int64(record))
						}
					}
				}
			}
		}
	}
	return nil
}
