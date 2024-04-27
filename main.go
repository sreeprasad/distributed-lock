package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
)

func tryLock(conn redis.Conn, lockKey, lockValue string, ttl int) bool {
	reply, err := redis.String(conn.Do("SET", lockKey, lockValue, "NX", "PX", ttl))
	if err != nil {
		return false
	}
	return reply == "OK"
}

func unlock(conn redis.Conn, lockKey, lockValue string) {
	luaScript := `
	if redis.call("get", KEYS[1]) == ARGV[1] then
		return redis.call("del", KEYS[1])
	else
		return 0
	end`
	script := redis.NewScript(1, luaScript)
	script.Do(conn, lockKey, lockValue)
}

func main() {

	redisHostsEnv := os.Getenv("REDIS_HOSTS")
	if redisHostsEnv == "" {
		log.Fatal("REDIS_HOSTS environment variable not set")
	}
	redisHosts := strings.Split(redisHostsEnv, ",")

	var pool *redis.Pool
	var conn redis.Conn
	for _, host := range redisHosts {
		pool = &redis.Pool{
			MaxIdle:   3,
			MaxActive: 3,
			Dial: func() (redis.Conn, error) {
				return redis.Dial("tcp", host)
			},
		}
		conn = pool.Get()
	}

	defer conn.Close()
	lockValue := os.Getenv("CONSUMER_ID")
	if lockValue == "" {
		log.Fatal("CONSUMER_ID environment variable not set")
	}

	lockKey := "sp-lock"
	ttlInMs := 10000
	startTime := time.Now()
	locksAcquired := 0

	var conns []redis.Conn
	for _, host := range redisHosts {
		conn, err := redis.Dial("tcp", host)
		if err != nil {
			log.Println("Failed to connect to", host)
			continue
		}
		defer conn.Close()
		if tryLock(conn, lockKey, lockValue, ttlInMs) {
			locksAcquired++
		}
		conns = append(conns, conn)
	}

	if isMajoriry(locksAcquired, redisHosts) && isTotalTimeLessThanTTL(startTime, ttlInMs) {

		fmt.Println("Lock acquired on majority of Redis instances")

		performSomeWork()

		for _, conn := range conns {
			unlock(conn, lockKey, lockValue)
		}
		fmt.Println("Lock released on all instances")

	} else {

		fmt.Println("Failed to acquire the lock on a majority of instances")
		for _, conn := range conns {
			unlock(conn, lockKey, lockValue)
		}

	}

}

func performSomeWork() {
	time.Sleep(5 * time.Second)
}

func isTotalTimeLessThanTTL(startTime time.Time, ttlInMs int) bool {
	return time.Since(startTime).Milliseconds() < int64(ttlInMs)
}

func isMajoriry(locksAcquired int, redisHosts []string) bool {
	return locksAcquired > len(redisHosts)/2
}
