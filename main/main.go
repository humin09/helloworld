package main

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

func main()  {
	fmt.Println(check())
}
func check() string {
	apis := []int{1, 2, 3, 4, 5, 6, 7}
	wg := sync.WaitGroup{}
	errs := make([]error, 0, 2)
	for _, i := range apis {
		if i%13 == 0 {
			errs = append(errs, errors.New("3 is invalid"))
			continue
		}
		j := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(time.Second)
			//收集错误
			if j%17 == 0 {
				errs = append(errs, errors.New("2 is invalid"))
			}
		}()
	}
	wg.Wait()
	if len(errs) > 0 {
		errString := make([]string, len(errs))
		for i, e := range errs {
			errString[i] = e.Error()
		}
		return strings.Join(errString, ",")
	} else {
		return ""
	}
}
