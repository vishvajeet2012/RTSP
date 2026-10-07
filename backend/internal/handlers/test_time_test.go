package handlers

import "time"

func future() time.Time { return time.Now().Add(time.Minute) }
