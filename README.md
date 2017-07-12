# Redis-Dialer 0.1.0
> Redis CMD funcs library and RESP client

### Methods
**func GetDialer(url string) (\*Dialer, error)**  
Return `*Dialer` with RESP client and CMD funcs.  
\> url : redis server host string
  
**func (\*Dialer) Close()**  
Close RESP client.  
   
Now there is only few CMD funcs support, you can see detail in `redis.go`. If you create more CMD, I'm glad to merge it : )

Simple Example : 
```
package main

import (
    "log"

	redis "../redis-dialer"
)

func main() {
    dialer, err := redis.GetDialer("localhost:6379")
	if err != nil {
		log.Fatalln("redis fail: " + err.Error())
	}

    num, err := dialer.DBSIZE()
	if err != nil {
		fmt.Println("redis fail: " + err.Error())
	}
    fmt.Println(num)

    result, err := dialer.HMSET("myhash", map[string]interface{}{
        "apple": "good",
        "orange": 4,
    })
    if err != nil {
		fmt.Println("redis fail: " + err.Error())
	}
    fmt.Println(result)

    myhash, err := dialer.HGETALL("myhash")
    if err != nil {
		fmt.Println("redis fail: " + err.Error())
	}
    fmt.Println(myhash["apple"])
    fmt.Println(myhash["orange"])
}

```