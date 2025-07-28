package main

import (
	"fmt"
)

func printPrimes(max int) {
	for i := range max {
		j := 2
		for ; j <= i; j++ {
			if i%j == 0 {
				break
			}
		}
		if j == i {
			println(i)
		}
	}
}

// don't edit below this line

func test(max int) {
	fmt.Printf("Primes up to %v:\n", max)
	printPrimes(max)
	fmt.Println("===============================================================")
}

func main() {
	test(10)
	test(20)
	test(30)
}
