package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

func dowork(d time.Duration, wg *sync.WaitGroup) {
	fmt.Println("doing work...")
	time.Sleep(d)
	fmt.Println("work is done!")
	wg.Done()
}

func doworkS(d time.Duration, resch chan string) {
	fmt.Println("doing work...")
	time.Sleep(d)
	fmt.Println("work is done!")
	resch <- fmt.Sprintf("work %d", rand.Intn(100))
}

func main() {
	//Wait group
	start := time.Now()
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go dowork(2*time.Second, wg)
	go dowork(4*time.Second, wg)
	wg.Wait()
	fmt.Printf("work took %v seconds\n", time.Since(start))

	// Channels
	start = time.Now()
	wg = &sync.WaitGroup{}
	resultch := make(chan string)
	wg.Add(3)
	go doworkS(2*time.Second, resultch)
	go doworkS(4*time.Second, resultch)
	go doworkS(6*time.Second, resultch)

	go func() {
		for res := range resultch {
			fmt.Println(res)
			wg.Done()
		}
	}()

	wg.Wait()
	close(resultch)
	fmt.Printf("work took %v seconds\n", time.Since(start))
}
