package tests

import (
	"fmt"
	"time"
)

func SpeedTest(name string, fn func()) {
	start := time.Now()
	fn()
	fmt.Printf("%s выполнена за %v\n", name, time.Since(start))
}
