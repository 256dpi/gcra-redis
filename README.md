# gcra

**A library for [go-redis](github.com/go-redis/redis) that implements the GCRA rate limit algorithm.**

This code is based on the Node.js implementation by [Losant](https://github.com/Losant/redis-gcra).

## Example

```go
// create redis client
client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

// create limiter
l := New(client)

// check limit
r, err := l.Check("user-1234", 100, 10, 1, time.Second)
if err != nil {
    panic(err)
}

fmt.Printf("%+v\n", r)

// check limit
r, err = l.Check("user-1234", 100, 10, 100, time.Second)
if err != nil {
    panic(err)
}

fmt.Printf("%+v\n", r)

// Output:
// {Limited:false Remaining:99 RetryIn:0s ResetIn:1s}
// {Limited:true Remaining:99 RetryIn:1s ResetIn:1s}
```
