health checks: used to ensure only healthy servers are kept in a load balancer rotation, checking status of every server. 

Active Health Checks: Attempt connection to a server/send it a HTTP request at a interval. If connected can't be established, health check fails. Take server out of rotation if fails a number of consectutive failed checks. 

So we are talking about waiting maybe 5 seconds, sending request to server. track number of failures, if hits a threshold, halt the server.


Passive Health Checks: This is where you are monitoring live traffic for errors, generally watching for bad HTTP responses. They'll detect these errors are any point of your proxied service, and require active traffic.



A good quick checklist is:
- ✅ The work can run independently of the caller.
- ✅ The work may block on I/O, like an HTTP request, file read, or network call.
- ✅ You want multiple tasks to happen at the same time.
- ✅ The caller does not need the result immediately before continuing.
- ✅ It’s background work, like health checks or periodic monitoring.
- ✅ Running sequentially would unnecessarily delay other work.
Reasons not to use one:
- ❌ net/http is already running your request handler concurrently.
- ❌ The work is tiny and must immediately finish before the next line.
- ❌ You would create an unbounded number of goroutines.
- ❌ You have no clear way to stop/cancel the goroutine.
- ❌ The goroutine would make shared-state synchronization much more complicated for little benefit.