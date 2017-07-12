package redis

import (
	"fmt"
	"strconv"
)

var redisHost string

// Dialer ...
type Dialer struct {
	Resp resp
}

// GetDialer ...
func GetDialer(url string) (*Dialer, error) {
	resp, err := newRESP(url)
	if err != nil {
		return nil, err
	}
	return &Dialer{Resp: resp}, nil
}

// Close ...
func (d *Dialer) Close() {
	d.Resp.Conn.Close()
}

// HMSET key object
func (d *Dialer) HMSET(key string, obj map[string]interface{}) (string, error) {
	cmd := []string{"HMSET", key}
	for k, v := range obj {
		cmd = append(cmd, k, fmt.Sprint(v))
	}
	result, err := d.Resp.cmd(cmd)
	if err != nil {
		return "", err
	}
	return result.(string), nil
}

// HGET key field
func (d *Dialer) HGET(key string, field string) (interface{}, error) {
	result, err := d.Resp.cmd(append([]string{"HGET"}, key, field))
	if err != nil {
		return nil, err
	}
	return result, nil
}

// EXPIRE key seconds
func (d *Dialer) EXPIRE(key string, second int) (int64, error) {
	result, err := d.Resp.cmd([]string{"EXPIRE", key, strconv.Itoa(second)})
	if err != nil {
		return 0, err
	}
	return result.(int64), nil
}

//HGETALL key
func (d *Dialer) HGETALL(key string) (map[string]interface{}, error) {
	result, err := d.Resp.cmd([]string{"HGETALL", key})
	data := map[string]interface{}{}
	for i := 0; i < len(result.([]interface{})); i += 2 {
		key := result.([]interface{})[i].(string)
		val := result.([]interface{})[i+1]
		data[key] = val
	}
	return data, err
}

// EXISTS key [key ...]
func (d *Dialer) EXISTS(keys ...string) (int64, error) {
	result, err := d.Resp.cmd(append([]string{"EXISTS"}, keys...))
	if err != nil {
		return 0, err
	}
	return result.(int64), nil
}

// DEL key [key ...]
func (d *Dialer) DEL(keys ...string) (int64, error) {
	result, err := d.Resp.cmd(append([]string{"DEL"}, keys...))
	if err != nil {
		return 0, err
	}
	return result.(int64), nil
}

// SADD key member [member ...]
func (d *Dialer) SADD(key string, member ...string) (int64, error) {
	result, err := d.Resp.cmd(append([]string{"SADD", key}, member...))
	if err != nil {
		return 0, err
	}
	return result.(int64), nil
}

// SISMEMBER key member
func (d *Dialer) SISMEMBER(key string, member string) (int64, error) {
	result, err := d.Resp.cmd([]string{"SISMEMBER", key, member})
	if err != nil {
		return 0, err
	}
	return result.(int64), nil
}

// SMEMBERS key
func (d *Dialer) SMEMBERS(key string) ([]interface{}, error) {
	result, err := d.Resp.cmd([]string{"SMEMBERS", key})
	return result.([]interface{}), err
}

// ZADD key score member
func (d *Dialer) ZADD(key string, score int, member string) (int64, error) {
	result, err := d.Resp.cmd([]string{"ZADD", key, strconv.Itoa(score), member})
	if err != nil {
		return 0, err
	}
	return result.(int64), nil
}

// ZRANGE key start stop [WITHSCORES]
func (d *Dialer) ZRANGE(key string, start int, stop int, WITHSCORES bool) ([]interface{}, error) {
	cmd := []string{"ZRANGE", key, strconv.Itoa(start), strconv.Itoa(stop)}
	if WITHSCORES {
		cmd = append(cmd, "WITHSCORES")
	}
	result, err := d.Resp.cmd(cmd)

	return result.([]interface{}), err
}

// ZSCORE key member
func (d *Dialer) ZSCORE(key string, member string) (interface{}, error) {
	cmd := []string{"ZSCORE", key, member}
	result, err := d.Resp.cmd(cmd)
	return result, err
}

// DBSIZE ...
func (d *Dialer) DBSIZE() (int64, error) {
	result, err := d.Resp.cmd([]string{"DBSIZE"})
	if err != nil {
		return 0, err
	}
	return result.(int64), nil
}

// LPUSH key value [value ...]
func (d *Dialer) LPUSH(key string, values ...string) (int64, error) {
	result, err := d.Resp.cmd(append([]string{"LPUSH", key}, values...))
	if err != nil {
		return 0, err
	}
	return result.(int64), nil
}

// LTRIM key start stop
func (d *Dialer) LTRIM(key string, start int, stop int) (string, error) {
	cmd := []string{"LTRIM", key, strconv.Itoa(start), strconv.Itoa(stop)}
	result, err := d.Resp.cmd(cmd)
	if err != nil {
		return "", err
	}
	return result.(string), nil
}

// LRANGE key start stop
func (d *Dialer) LRANGE(key string, start int, stop int) ([]interface{}, error) {
	cmd := []string{"LRANGE", key, strconv.Itoa(start), strconv.Itoa(stop)}
	result, err := d.Resp.cmd(cmd)
	if err != nil {
		return nil, err
	}
	return result.([]interface{}), nil
}
