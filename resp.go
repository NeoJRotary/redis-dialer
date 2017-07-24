package redis

import (
	"bytes"
	"errors"
	"net"
	"strconv"
	"sync"
	"time"
)

// RESP ...
type RESP struct {
	Conn     net.Conn
	wBuf     chan uint32
	rBuf     []byte
	ioBuf    [100000]*ioBuf
	i        uint32
	mutex    *sync.Mutex
	TimeoutR time.Duration
	TimeoutW time.Duration
}

type ioBuf struct {
	cmd  []byte
	data []interface{}
	err  (chan error)
	len  int
}

const crlf = "\r\n"

func newRESP(url string) (*RESP, error) {
	conn, err := net.DialTimeout("tcp", url, 5*time.Second)
	if err != nil {
		return nil, err
	}

	r := RESP{
		Conn:     conn,
		wBuf:     make(chan uint32),
		rBuf:     make([]byte, 65536),
		i:        0,
		mutex:    new(sync.Mutex),
		TimeoutW: 1 * time.Second,
		TimeoutR: 2 * time.Second,
	}

	go r.listen()
	return &r, nil
}

func parseCmd(args []string) []byte {
	cmd := []byte("*" + strconv.Itoa(len(args)) + crlf)
	for _, v := range args {
		cmd = append(cmd, []byte("$"+strconv.Itoa(len([]byte(v)))+crlf)...)
		cmd = append(cmd, []byte(v+crlf)...)
	}
	return cmd
}

func (r *RESP) pipe(cmds [][]string) ([]interface{}, error) {
	buf := parseCmd(cmds[0])
	for _, s := range cmds[1:] {
		buf = append(buf, []byte(crlf)...)
		buf = append(buf, parseCmd(s)...)
	}
	return r.write(buf, len(cmds))
}

func (r *RESP) cmd(args []string) (interface{}, error) {
	result, err := r.write(parseCmd(args), 1)
	if err != nil {
		return nil, err
	}
	return result[0], nil
}

func (r *RESP) write(cmd []byte, len int) ([]interface{}, error) {
	r.mutex.Lock()
	i := r.i
	r.i++
	if r.i == 100000 {
		r.i = 0
	}
	r.mutex.Unlock()

	r.ioBuf[i] = &ioBuf{
		data: []interface{}{},
		err:  make(chan error),
		cmd:  cmd,
		len:  len,
	}

	r.wBuf <- i
	err := <-r.ioBuf[i].err
	if err != nil {
		return nil, err
	}
	data := r.ioBuf[i].data

	go func() {
		r.ioBuf[i].data = nil
		r.ioBuf[i].err = nil
		r.ioBuf[i].cmd = nil
		r.ioBuf[i].len = 0
	}()

	return data, err
}

func (r *RESP) listen() {
	for {
		i := <-r.wBuf

		timer := time.Now().Add(r.TimeoutW)
		r.Conn.SetWriteDeadline(timer)
		_, err := r.Conn.Write(r.ioBuf[i].cmd)
		if err != nil {
			r.ioBuf[i].err <- err
			return
		}

		timer = time.Now().Add(r.TimeoutR)
		r.Conn.SetReadDeadline(timer)

		for {
			n, err := r.Conn.Read(r.rBuf)
			if err != nil {
				r.ioBuf[i].err <- err
			}
			data, err := decoder(r.rBuf[:n])
			if err != nil {
				r.ioBuf[i].err <- err
			}
			r.ioBuf[i].data = append(r.ioBuf[i].data, data...)
			if len(r.ioBuf[i].data) >= r.ioBuf[i].len {
				r.ioBuf[i].err <- nil
				break
			}
		}
	}
}

func decoder(buf []byte) ([]interface{}, error) {
	newBuf := bytes.Split(buf, []byte{'\r', '\n'})
	result := []interface{}{}
	i := 0
	for i != -1 {
		data, j, err := analyzer(newBuf, i)
		if err != nil {
			return nil, err
		}
		if j == -1 {
			i = -1
		} else {
			i = j + 1
			result = append(result, data)
		}
	}
	return result, nil
}

/*
For Simple Strings the first byte of the reply is "+"
For Errors the first byte of the reply is "-"
For Integers the first byte of the reply is ":"
For Bulk Strings the first byte of the reply is "$"
For Arrays the first byte of the reply is "*"
*/

func analyzer(buf [][]byte, index int) (interface{}, int, error) {
	data := buf[index]
	if len(data) == 0 {
		return nil, -1, nil
	}

	switch data[0] {
	case '-':
		return nil, index, errors.New(string(data[1:]))
	case '+':
		return string(data[1:]), index, nil
	case ':':
		result, err := strconv.ParseInt(string(data[1:]), 10, 64)
		if err != nil {
			return nil, index, err
		}
		return result, index, nil
	case '$':
		len := string(data[1:])
		if len == "-1" {
			return nil, index, nil
		}
		index++
		return string(buf[index]), index, nil
	case '*':
		len := string(data[1:])
		if len == "-1" {
			return nil, index, nil
		}
		num, err := strconv.ParseInt(len, 10, 32)
		if err != nil {
			return nil, index, err
		}
		result := make([]interface{}, num)
		for i := range result {
			index++
			result[i], index, err = analyzer(buf, index)
			if err != nil {
				return nil, index, err
			}
		}
		return result, index, nil
	}

	return nil, index, errors.New("RESP cant resolve incorrect response")
}
