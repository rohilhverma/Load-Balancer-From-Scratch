# 1. `net/http`: how Go web servers work

The most important idea is that Go models an HTTP server around **handlers**.

A handler is simply something that receives:

- an HTTP request
- a place to write the HTTP response

The core interface is effectively:

```go
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}
```

You don't normally implement all the low-level networking yourself.

A tiny HTTP server looks like this:

```go
package main

import (
	"fmt"
	"net/http"
)

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello from the server")
}

func main() {
	http.HandleFunc("/", helloHandler)

	http.ListenAndServe(":8080", nil)
}
```

Then:

```go
curl localhost:8080
```

returns:

```go
Hello from the server
```

---
# 2. `http.Request`

Every incoming request is represented by:

```go
*http.Request
```

It contains things like:

```go
r.Method
r.URL
r.Header
r.Body
r.Host
```

So your handler method is generally going to be in the format of 
```go
func handler(w http.ResponseWriter, r *http.Request) {
	...
}
```

---
# 3. `http.ResponseWriter`

This:

```go
w http.ResponseWriter
```

is how your handler sends information **back to the client**.

For example:

```go
func handler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("hello"))
}
```

You can also set headers:

```go
w.Header().Set("Content-Type", "application/json")
```


---
# 4. `http.Handler`

Instead of using functions directly, Go has an interface:

```go
type Handler interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}
```

That means this is valid:

```go
type MyHandler struct{}

func (h MyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "hello")
}
```

Then:

```go
func main() {
	handler := MyHandler{}

	http.ListenAndServe(":8080", handler)
}
```

So it'll be like a one-stop package for dealing with all of the HTTP requests

---
# 22. `http.Client`

You've seen HTTP from the server side:

```
http.ListenAndServe(...)
```

`http.Client` represents the **client side**.

Example:

```
client := &http.Client{}

resp, err := client.Get("http://localhost:9001/health")
```

Now your load balancer itself is acting as an HTTP client.

That's important because the load balancer is both:

```
server to the original client
```

and:

```
client to the backend
```

---
# 23. `http.Transport`

The client is the high-level API.

The **Transport** controls lower-level HTTP connection behavior.

Conceptually:

```
http.Client
    ↓
http.Transport
    ↓
TCP connections
```

Things the transport handles include:

- connection reuse
- keep-alive
- connection pooling
- proxy settings
- TLS
- idle connections

For example:

```
transport := &http.Transport{
	MaxIdleConns:        100,
	MaxIdleConnsPerHost: 20,
}

client := &http.Client{
	Transport: transport,
	Timeout:   2 * time.Second,
}
```



---
Use of a Reverse Proxy: This sits in front of servers, between the clients and servers. It can make the client believe it's talking to `localhost:8080`, but really it could be talking to `localhost:9001`. 

So `httputil.ReverseProxy` will receive a request, modify the destination, send and receive to/from the server, and send to the client. 

A minimal proxy looks like:

```go
proxy := &httputil.ReverseProxy{
    Rewrite: func(r *httputil.ProxyRequest) {
        r.SetURL(target)
    },
}
```

	Rewrite is the function that lets the proxy determine what the outgoing request to the server should look like. This ProxyRequest object comes with a `r.In` and `r.Out`. Modify `r.Out` to redirect/change direction. 

And because `ReverseProxy` implements `http.Handler`, you can do:

```go
http.ListenAndServe(":8080", proxy)
```

