package wttj

import "time"

func ResetLimiter() { limiter = &gate{now: time.Now} }
