Not parallelism. Parallelism has you executing multiple seperate tasks at the same time. If they were to run, they'd all complete at the same time. Concurrency is managing multiple tasks at once. If they were to run, some would complete ahead of others, and they'd differ on specific runs. 

Channels:
```go
var c chan int // define variable C of type channel integer
c = make(chant int)

//Send a value down the channel
c <- 1

//Recieve from a channel
value = <-c
```

```run-go
package main
import (
	"fmt" 
	"math/rand" 
	"time"
)
func main() {
	c := make(chan string)
	go boring("boring!", c)
	for i := 0; i < 5; i++ {
		fmt.Printf("You say: %q\n", <-c) // Receive expression is just a value.
	}
	fmt.Println("You're boring; I'm leaving.")
}

func boring(msg string, c chan string) {
	for i := 0; ; i++ {
		c <- fmt.Sprintf("%s %d", msg, i) // Expression to be sent can be any suitable value.
		time.Sleep(time.Duration(rand.Intn(1e3)) * time.Millisecond)
	}
}
```
The main function will wait for the execution of `<-c` and same with the boring function. They communicate and synchronize by this waiting mechanic. 

Buffer Channels: They remove the synchronization aspect, allowing you to proactively store some values in your channels. 

---
These channels can be passed as first-class values. You can load them up, return them, and then start ingesting the values of the channel. 

```run-go
package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {
	c := boring("boring!") // Function returning a channel.
	for i := 0; i < 5; i++ {
		fmt.Printf("You say: %q\n", <-c)
	}
	fmt.Println("You're boring, I'm leaving.")
}

func boring(msg string) <-chan string { // Returns a receive-only channel.
	c := make(chan string)
	go func() { // We launch the goroutine from inside the function.
		for i := 0; ; i++ {
			c <- fmt.Sprintf("%s %d", msg, i)
			time.Sleep(time.Duration(rand.Intn(1e3)) * time.Millisecond)
		}
	}()
	return c // Return the channel to the caller.
}
```
---
Select Statements are ways to work with multiple channels, where you'll evaluate every channel, and if one passes, you'll proceed, otherwise you default. 

```run-go
package main

import (
	"fmt"
	"math/rand"
	"time"
)

func boring(msg string) <-chan string {
	c := make(chan string)
	go func() {
		for i := 0; ; i++ {
			c <- fmt.Sprintf("%s %d", msg, i)
			time.Sleep(time.Duration(rand.Intn(1500)) * time.Millisecond)
		}
	}()
	return c
}

func main() {
	c := boring("Joe")
	for {
		select {
		case s := <-c:
			fmt.Println(s)
		case <-time.After(800 * time.Millisecond):
			fmt.Println("You're too slow! Exiting.")
			return
		}
	}
}
```
