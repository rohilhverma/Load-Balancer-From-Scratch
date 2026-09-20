Every concurrently executing activity is a goroutine, like running functions that are active at the same time. They have their call stack and can kind of be thought of as very cheap threads. 

Starting up a program, the only goroutine at the moment is the one calling the main function, the main goroutine. You split off new goroutines, specifically method calls with the `go` statement. 

They'll stop upon the main function returning

```run-go
package main

import (
"fmt"
"time"
)

func main() {
		go spinner(100 * time.Millisecond)
		const n = 45
		fibN := fib(n) // slow
		fmt.Printf("\rFibonacci(%d) = %d\n", n, fibN)
	}
	
	func spinner(delay time.Duration) {
		for {
			for _, r := range `-\|/` {
					fmt.Printf("\r%c", r)
					time.Sleep(delay)
					}
				}
	}
		
	func fib(x int) int {
	if x < 2 {
		return x
	}
	return fib(x-1) + fib(x-2)
}
```


It makes sense in terms of networking, where servers are handling independent clients. You can use the net package to build TCP/UDP connections. - [TCP Server]

```go
func main() {
	listener, err := net.Listen("tcp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}
	conn, err := listener.Accept()
```

You create a TCP connection request, and use the Accept to block for a connection to the port. You can create the connection by using `nc localhost 8000` to connect to the server. 

