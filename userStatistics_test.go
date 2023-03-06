// Copyright 2020 rateLimit Author(https://github.com/yudeguang/ratelimit). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/yudeguang/ratelimit.

package ratelimit

import (
	"testing"
	"time"
)

func Test_userStatistics(t *testing.T) {
	r := NewRule()
	r.AddRule(time.Hour*1, 100)
	r.AddRule(time.Second*10, 2)
	r.AllowVisit("ydg")
	r.AllowVisit("chery")
	r.AllowVisit("ydg")
	r.AllowVisit("vivian")
	curOnlineUsersVisitsDetail := r.GetCurOnlineUsersVisitsDetail()
	//log.Println(curOnlineUsersVisitsDetail)
	user := curOnlineUsersVisitsDetail[2][0]
	remainingVisit := curOnlineUsersVisitsDetail[2][1]
	if user != "ydg" {
		t.Fatalf("unexpected value obtained; got %q want %q", user, "ydg")
	}
	if remainingVisit != "0" {
		t.Fatalf("unexpected value obtained; got %q want %q", remainingVisit, "0")
	}
}
