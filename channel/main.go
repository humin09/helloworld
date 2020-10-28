package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*8)
	ch := make(chan int, 1)
	go produce(ch, cancel)
	go consume(ch, ctx)
	time.Sleep(time.Second * 10)
}
func consume(ch <-chan int, ctx context.Context) {
	for {
		select {
		case s, more := <-ch:
			if more {
				fmt.Println(s)
			} else {
				fmt.Println("close")
				return
			}

		case _ = <-ctx.Done():
			fmt.Println("end")
			return
		}
	}
}
func produce(ch chan<- int, cancelFunc context.CancelFunc) {
	for i := 0; i < 5; i++ {
		ch <- i
		time.Sleep(time.Second)
	}
	close(ch)
}
