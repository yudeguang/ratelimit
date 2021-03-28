// Copyright 2020 ratelimit Author(https://github.com/yudeguang/ratelimit). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/yudeguang/ratelimit.

package ratelimit

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//如果有历史备份文件，则加载，并且开启自动保存，默认60秒完成一次存盘
func (this *Rule) LoadingAndAutoSaveToDisc(backupFileName string, backUpInterval ...time.Duration) (err error) {
	if len(this.rules) == 0 {
		panic("rule is empty，please add rule by AddRule")
	}
	this.needBackup = true
	this.backupFileName = strings.Split(backupFileName, ".")[0]
	if len(backUpInterval) == 0 {
		this.backUpInterval = time.Second * 60 //默认60秒存盘一次
	} else {
		this.backUpInterval = backUpInterval[0]
	}

	if this.backupFileName == "" {
		panic("backupFileName err:" + backupFileName)
	}
	err = this.loading()
	go func() {
		finished := true
		for range time.Tick(this.backUpInterval) {
			//如果数据量较大，那么在一个清除周期内不一定会把所有数据全部清除,所以要判断上一轮次的清除是否完成
			if finished {
				finished = false
				this.SaveToDiscOnce()
				finished = true
			}
		}

	}()
	return err
}

//把数据保存到硬盘上,仅支持key为string,int,int64等类型数据的缓存
func (this *Rule) SaveToDiscOnce(backupFileNames ...string) (err error) {
	this.locker.Lock()
	defer this.locker.Unlock()
	if len(this.rules) == 0 {
		panic("rule is empty，please add rule by AddRule")
	}
	backupFileName := this.backupFileName
	if len(backupFileNames) > 0 {
		backupFileName = strings.Split(backupFileNames[0], ".")[0]
	}
	if backupFileName == "" {
		return fmt.Errorf("back up file not exist")
	}
	f, err := os.Create(backupFileName + ".ratelimit_temp")
	if err != nil {
		return err
	}
	buf := bufio.NewWriterSize(f, 40960)
	//先写规则数量
	_, err = buf.Write(uint64ToByte(uint64(len(this.rules))))
	if err != nil {
		return err
	}
	//依次写入每一组数据
	for i := range this.rules {
		curRuleData := new(bytes.Buffer)
		tempBuf := bufio.NewWriterSize(curRuleData, 40960)
		num := 0
		this.rules[i].usedRecordsIndex.Range(func(key, Index interface{}) bool {
			index := Index.(int)
			//有效的才能加进去
			if len(this.rules[i].visitorRecords) > index && this.rules[i].visitorRecords[index].key == key {
				switch key.(type) {
				case string:
					//与其它类型不同，KEY长度是不定长的
					tempBuf.Write([]byte{0x00})
					tempBuf.Write(uint64ToByte(uint64(len(key.(string)))))
					tempBuf.WriteString(key.(string))
				case int:
					tempBuf.Write([]byte{0x01})
					tempBuf.Write(uint64ToByte(uint64(key.(int))))
				case int8:
					tempBuf.Write([]byte{0x02})
					tempBuf.Write(uint64ToByte(uint64(key.(int8))))
				case int16:
					tempBuf.Write([]byte{0x03})
					tempBuf.Write(uint64ToByte(uint64(key.(int16))))
				case int32:
					tempBuf.Write([]byte{0x04})
					tempBuf.Write(uint64ToByte(uint64(key.(int32))))
				case int64:
					tempBuf.Write([]byte{0x05})
					tempBuf.Write(uint64ToByte(uint64(key.(int64))))
				case uint:
					tempBuf.Write([]byte{0x06})
					tempBuf.Write(uint64ToByte(uint64(key.(uint))))
				case uint8:
					tempBuf.Write([]byte{0x07})
					tempBuf.Write(uint64ToByte(uint64(key.(uint8))))
				case uint16:
					tempBuf.Write([]byte{0x08})
					tempBuf.Write(uint64ToByte(uint64(key.(uint16))))
				case uint32:
					tempBuf.Write([]byte{0x09})
					tempBuf.Write(uint64ToByte(uint64(key.(uint32))))
				case uint64:
					tempBuf.Write([]byte{0x0A})
					tempBuf.Write(uint64ToByte(key.(uint64)))
				default:
					panic("key type can only be string,int,int8,int16,int32,int64,uint,uint8,uint16,uint32,uint64")
				}

				this.rules[i].visitorRecords[index].tailForCopy = this.rules[i].visitorRecords[index].tail
				this.rules[i].visitorRecords[index].headForCopy = this.rules[i].visitorRecords[index].head
				size := this.rules[i].visitorRecords[index].UsedSizeForCopy()

				//无数据可以不写
				if size > 0 {
					//单个KEY对应的访问记录数
					tempBuf.Write(uint64ToByte(uint64(size)))
					for ii := 0; ii < size; ii++ {
						val, _ := this.rules[i].visitorRecords[index].PopForCopy()
						tempBuf.Write(uint64ToByte(uint64(val)))
					}
				}
				num++
			}
			return true
		})
		buf.Write(uint64ToByte(uint64(i)))   //先写当前下标
		buf.Write(uint64ToByte(uint64(num))) //再写当前键的个数
		//某条规则下面，可能还无访问记录
		if num > 0 {
			tempBuf.Flush()
			b := curRuleData.Bytes()
			buf.Write(b)
		}
	}
	buf.Flush()
	err = f.Close()
	if err != nil {
		return
	}
	//成功生成临时文件后，成替换正式文件
	_, err = copy(backupFileName+".ratelimit", backupFileName+".ratelimit_temp")
	return
}
func uint64ToByte(i uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, i)
	return b
}

//复制文件，目标文件所在目录不存在，则创建目录后再复制
//Copy(`d:\test\hello.txt`,`c:\test\hello.txt`)
func copy(dstFileName, srcFileName string) (w int64, err error) {
	//打开源文件
	srcFile, err := os.Open(srcFileName)
	if err != nil {
		return 0, err
	}
	defer srcFile.Close()
	// 创建新的文件作为目标文件
	dstFile, err := os.Create(dstFileName)
	if err != nil {
		//如果出错，很可能是目标目录不存在，需要先创建目标目录
		err = os.MkdirAll(filepath.Dir(dstFileName), 0666)
		if err != nil {
			return 0, err
		}
		//再次尝试创建
		dstFile, err = os.Create(dstFileName)
		if err != nil {
			return 0, err
		}
	}
	defer dstFile.Close()
	//通过bufio实现对大文件复制的自动支持
	dst := bufio.NewWriter(dstFile)
	defer dst.Flush()
	src := bufio.NewReader(srcFile)
	w, err = io.Copy(dst, src)
	if err != nil {
		return 0, err
	}
	return w, err
}
