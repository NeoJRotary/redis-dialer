# Redis-Dialer 0.2.0
> Redis CMD funcs library and RESP client

### Methods
**func GetDialer(url string) (\*Dialer, error)**  
Return `*Dialer` with RESP client and CMD funcs.  
\> url : redis server host string
  
**func (\*Dialer) Close()**  
Close RESP client.  
  
**func (\*Dialer) CMD(args ...string)**  
Send Redis Command.   
\> arge : spread string   
  
**func (\*Dialer) PIPELINE(cmds [][]string)**  
Close RESP client.  
\> cmds : string slice's slice  
   
### Commands
See detail or try searching in `redis.go`   
Now there is only few Commands support. If you create more commands, I'm glad to merge it : )

### Simple Example : 
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

    myhash, err := dialer.CMD("HGETALL", "myhash")
    if err != nil {
    fmt.Println("redis fail: " + err.Error())
  }
    fmt.Println(myhash["apple"])
    fmt.Println(myhash["orange"])

    result, err := dialer.PIPELINE([][]string{
        []string{"PING"},
        []string{"DBSIZE"},
        []string{"PING"},
    })
    if err != nil {
    fmt.Println("redis fail: " + err.Error())
  }
    fmt.Println(result)
}

```