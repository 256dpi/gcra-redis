package gcra

import (
	"errors"
	"time"

	"github.com/go-redis/redis"
)

// copied from https://github.com/Losant/redis-gcra
var gcraScript = redis.NewScript(`
local rate_limit_key = KEYS[1]
local now            = ARGV[1]
local burst          = ARGV[2]
local rate           = ARGV[3]
local period         = ARGV[4]
local cost           = ARGV[5]

local emission_interval = period / rate
local increment         = emission_interval * cost
local burst_offset      = emission_interval * burst

local tat = redis.call("GET", rate_limit_key)

if not tat then
  tat = now
else
  tat = tonumber(tat)
end
tat = math.max(tat, now)

local new_tat = tat + increment
local allow_at = new_tat - burst_offset
local diff = now - allow_at

local limited
local retry_in
local reset_in

local remaining = math.floor(diff / emission_interval + 0.5) -- poor man's round

if remaining < 0 then
  limited = 1
  -- calculate how many tokens there actually are, since
  -- remaining is how many there would have been if we had been able to limit
  -- and we did not limit
  remaining = math.floor((now - (tat - burst_offset)) / emission_interval + 0.5)
  reset_in = math.ceil(tat - now)
  retry_in = math.ceil(diff * -1)
elseif remaining == 0 and increment <= 0 then
  -- request with cost of 0
  -- cost of 0 with remaining 0 is still limited
  limited = 1
  remaining = 0
  reset_in = math.ceil(tat - now)
  retry_in = 0 -- retry in is meaningless when cost is 0
else
  limited = 0
  reset_in = math.ceil(new_tat - now)
  retry_in = 0
  if increment > 0 then
    redis.call("SET", rate_limit_key, new_tat, "EX", reset_in)
  end
end

return {limited, remaining, retry_in, reset_in}`)

// ErrZeroParameters is returned if a parameter is zero.
var ErrZeroParameters = errors.New("zero rate, burst or period provided")

// ErrCostHigherThanBurst is returned if the provided cost is higher than the
// provided burst.
var ErrCostHigherThanBurst = errors.New("cost higher than burst")

// Result contains the rate limiting result.
type Result struct {
	Limited   bool
	Remaining int64
	RetryIn   time.Duration
	ResetIn   time.Duration
}

// Limiter is a GCRA based limiter.
type Limiter struct {
	redis *redis.Client
}

// New creates a new GCRA based limiter.
func New(redis *redis.Client) *Limiter {
	return &Limiter{
		redis: redis,
	}
}

// Check will perform the rate limit check. Specify burst as the maximum tokens
// available and rate as the regeneration of tokens per period.
func (l *Limiter) Check(key string, burst, rate, cost int64, period time.Duration) (Result, error) {
	// prepare result
	var res Result

	// check arguments
	if burst == 0 || rate == 0 || uint64(period.Seconds()) == 0 {
		return res, ErrZeroParameters
	}

	// check if cost is higher than burst
	if cost > burst {
		return res, ErrCostHigherThanBurst
	}

	// run script
	data, err := gcraScript.Run(l.redis, []string{key}, time.Now().Unix(), burst, rate, int64(period.Seconds()), cost).Result()
	if err != nil {
		return res, err
	}

	// get slice
	slice := data.([]interface{})

	// populate result
	res.Limited = slice[0].(int64) == 1
	res.Remaining = slice[1].(int64)
	res.RetryIn = time.Duration(slice[2].(int64)) * time.Second
	res.ResetIn = time.Duration(slice[3].(int64)) * time.Second

	return res, nil
}
