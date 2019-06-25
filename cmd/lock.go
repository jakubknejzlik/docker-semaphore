package cmd

import (
	"errors"
	"os"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/urfave/cli"
)

var lockCmd = cli.Command{
	Name:      "lock",
	Usage:     "lock semaphore for given key",
	UsageText: "lock [key]",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "timeout",
			Usage: "amount of time till the lock expires",
			Value: "5m",
		},
		cli.IntFlag{
			Name:  "size",
			Usage: "semaphore size",
		},
	},
	Action: func(ctx *cli.Context) error {
		key := ctx.Args().First()
		secret := ctx.Args().Get(1)
		// size := ctx.Int("size")
		timeout := ctx.String("timeout")

		t, err := time.ParseDuration(timeout)
		if err != nil {
			return cli.NewExitError(err, 1)
		}

		retryCount := int(t.Seconds() / 5)
		for i := 0; i < retryCount; i++ {
			ok, _err := writeLock(key, secret, 1000*600)
			err = _err

			if !ok {
				time.Sleep(time.Second * 5)
				continue
			}
			// if err != nil {
			// 	return cli.NewExitError(err, 1)
			// }
		}
		return err
	},
}
var unlockCmd = cli.Command{
	Name:      "unlock",
	Usage:     "unlock semaphore for given key",
	UsageText: "unlock [key]",
	Action: func(ctx *cli.Context) error {
		key := ctx.Args().First()
		secret := ctx.Args().Get(1)

		if _, err := releaseLock(key, secret); err != nil {
			return cli.NewExitError(err, 1)
		}
		return nil
	},
}

var ErrLockMismatch = errors.New("key is locked with a different secret")

const lockScript = `
local v = redis.call("GET", KEYS[1])
if v == false or v == ARGV[1]
then
	return redis.call("SET", KEYS[1], ARGV[1], "EX", ARGV[2]) and 1
else
	return 0
end
`

const unlockScript = `
local v = redis.call("GET",KEYS[1])
if v == false then
	return 1
elseif v == ARGV[1] then
	return redis.call("DEL",KEYS[1])
else
	return 0
end
`

var redisPool = &redis.Pool{
	MaxIdle:     3,
	IdleTimeout: 240 * time.Second,
	// Dial or DialContext must be set. When both are set, DialContext takes precedence over Dial.
	Dial: func() (redis.Conn, error) { return redis.Dial("tcp", os.Getenv("REDIS_ADDR")) },
}

// writeLock attempts to grab a redis lock. The error returned is safe to ignore
// if all you care about is whether or not the lock was acquired successfully.
func writeLock(name, secret string, ttl uint64) (bool, error) {
	rc := redisPool.Get()
	defer rc.Close()

	script := redis.NewScript(1, lockScript)
	resp, err := redis.Int(script.Do(rc, name, secret, int64(ttl*1000)))
	if err != nil {
		return false, err
	}
	if resp == 0 {
		return false, ErrLockMismatch
	}
	return true, nil
}

// writeLock releases the redis lock
func releaseLock(name, secret string) (bool, error) {
	rc := redisPool.Get()
	defer rc.Close()

	script := redis.NewScript(1, unlockScript)
	resp, err := redis.Int(script.Do(rc, name, secret))
	if err != nil {
		return false, err
	}
	if resp == 0 {
		return false, ErrLockMismatch
	}
	return true, nil
}