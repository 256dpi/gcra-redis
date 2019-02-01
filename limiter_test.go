package gcra

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
)

var client *redis.Client

func init() {
	client = redis.NewClient(&redis.Options{Addr: "localhost:6379"})
}

func Example() {
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
}

func TestLimiter(t *testing.T) {
	assert.NoError(t, client.Del("gcra").Err())

	l := New(client)
	burst := int64(4)
	rate := int64(10)
	period := 10 * time.Second

	r, err := l.Check("gcra", burst, rate, 1, period)
	assert.NoError(t, err)
	assert.Equal(t, Result{false, 3, 0, 1 * time.Second}, r)

	r, err = l.Check("gcra", burst, rate, 1, period)
	assert.NoError(t, err)
	assert.Equal(t, Result{false, 2, 0, 2 * time.Second}, r)

	r, err = l.Check("gcra", burst, rate, 1, period)
	assert.NoError(t, err)
	assert.Equal(t, Result{false, 1, 0, 3 * time.Second}, r)

	r, err = l.Check("gcra", burst, rate, 1, period)
	assert.NoError(t, err)
	assert.Equal(t, Result{false, 0, 0, 4 * time.Second}, r)

	r, err = l.Check("gcra", burst, rate, 1, period)
	assert.NoError(t, err)
	assert.Equal(t, Result{true, 0, 1 * time.Second, 4 * time.Second}, r)

	time.Sleep(2 * time.Second)

	r, err = l.Check("gcra", burst, rate, 1, period)
	assert.NoError(t, err)
	assert.Equal(t, Result{false, 1, 0, 3 * time.Second}, r)

	time.Sleep(time.Second)

	r, err = l.Check("gcra", burst, rate, 1, period)
	assert.NoError(t, err)
	assert.Equal(t, Result{false, 1, 0, 3 * time.Second}, r)

	r, err = l.Check("gcra", burst, rate, 2, period)
	assert.NoError(t, err)
	assert.Equal(t, Result{true, 1, 1 * time.Second, 3 * time.Second}, r)
}

func TestLimiterErrors(t *testing.T) {
	l := New(client)
	burst := int64(4)
	rate := int64(10)
	period := 10 * time.Second

	r, err := l.Check("gcra", 0, rate, 1, period)
	assert.Equal(t, ErrZeroParameters, err)
	assert.Equal(t, Result{}, r)

	r, err = l.Check("gcra", burst, 0, 1, period)
	assert.Equal(t, ErrZeroParameters, err)
	assert.Equal(t, Result{}, r)

	r, err = l.Check("gcra", burst, rate, 1, 0)
	assert.Equal(t, ErrZeroParameters, err)
	assert.Equal(t, Result{}, r)

	r, err = l.Check("gcra", burst, rate, burst+1, period)
	assert.Equal(t, ErrCostHigherThanBurst, err)
	assert.Equal(t, Result{}, r)
}

func TestRedisErrors(t *testing.T) {
	l := New(redis.NewClient(&redis.Options{Addr: "localhost:1234"}))

	r, err := l.Check("gcra", 10, 1, 1, time.Second)
	assert.Error(t, err)
	assert.Equal(t, Result{}, r)
}

func BenchmarkLimiter(b *testing.B) {
	err := client.Del("gcra").Err()
	if err != nil {
		panic(err)
	}

	l := New(client)

	for i := 0; i < b.N; i++ {
		_, err := l.Check("gcra", int64(b.N), 10, 1, 10*time.Second)
		if err != nil {
			panic(err)
		}
	}
}
