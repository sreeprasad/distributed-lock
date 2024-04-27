## Redlock implementation

## Main concept

### Background

There are 2 go-lang servers running and there are 4 redis servers
running. Each go-lang server is trying to accquire a lock and peform some work.

### How to accquire lock

Each server has a unique lock value given by `consumer_id`
Each server tries to acquire the lock by setting the lock-key as "sp-lock" and
lock value as its environment "consumer_id". If the server is able to acquire
locks on majority of redis servers (here greater than or equal to 3 redis
servers) and the time to acquire the lock since start is less than `ttl` then
server has accquired the lock and perform work

```shell

start_time = current_time()
ttl_ms = 100ms
total_locks_accquired = accquire_locks_with_ttl(ttl_ms)
has_majority = total_locks_accquired > len(redis_hosts)/2
total_time = current_time - start_time
if has_majority and total_time < ttl_ms:

    perform_work
    release_all_locks()

else:
    release_all_locks()

```
