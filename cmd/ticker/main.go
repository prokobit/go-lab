package main

import (
	"fmt"
	"time"
)

func main() {
	userch := make(chan string, 1)
	go func() {
		arr := []int{1, 2, 3, 4, 5}
		for n := range arr {
			time.Sleep(3 * time.Second)
			userch <- fmt.Sprintf("User-%d", n)
		}
	}()

	ticker := time.NewTicker(2 * time.Second)
	for {
		select {
		case <-ticker.C:
			fmt.Println("tick")
		case user := <-userch:
			fmt.Printf("%s\n", user)
		}
	}
}
