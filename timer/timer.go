package timer

import (
	"fmt"
	"time"
)

func timer1() {
	//初始化定时器
	t := time.NewTimer(2 * time.Second)
	//当前时间
	now := time.Now()
	fmt.Printf("Now time : %v.\n", now)

	expire := <-t.C
	fmt.Printf("Expiration time: %v.\n", expire)
}

func timer2(i1, i2, i3 time.Duration) {
	ch1 := make(chan int, 1)
	ch2 := make(chan int, 1)
	go func() {
		time.Sleep(i1)
		ch1 <- 1
	}()
	go func() {
		time.Sleep(i2)
		ch2 <- 2
	}()
	select {
	case e1 := <-ch1:
		//如果ch1通道成功读取数据，则执行该case处理语句
		fmt.Printf("1th case is selected. e1=%v", e1)
	case e2 := <-ch2:
		//如果ch2通道成功读取数据，则执行该case处理语句
		fmt.Printf("2th case is selected. e2=%v", e2)
	case <-time.After(i3):
		fmt.Println("Timed out")
	}
}
