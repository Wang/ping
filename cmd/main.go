package main

import (
	"fmt"
	"sync"
	"time"

	"ping"
)

func main() {
	//ping.Debug = true
	ping.SetDevice("en0")
	ipList := []string{"120.24.94.152", "123.57.41.221", "103.28.10.236"}
	//ipList := []string{"123.57.41.221", "103.28.10.236"}
	var wg sync.WaitGroup
	for k := 0; k < 5; k++ {
		st := time.Now()
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(ip string) {
				err := ping.Run(ip, 100)
				if err != nil {
					fmt.Println(err)
				}
				wg.Done()
			}(ipList[i%2])
		}
		wg.Wait()
		fmt.Println("extime:", time.Now().Sub(st))
		time.Sleep(time.Second * 2)
	}

}
