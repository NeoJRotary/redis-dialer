package redis

import (
	"bufio"
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
	cmd []byte
	buf (chan []byte)
	err (chan error)
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

func (r *RESP) cmd(args []string) (data interface{}, err error) {
	r.mutex.Lock()
	i := r.i
	r.i++
	if r.i == 100000 {
		r.i = 0
	}
	r.mutex.Unlock()

	cmd := []byte("*" + strconv.Itoa(len(args)) + crlf)
	for _, v := range args {
		cmd = append(cmd, []byte("$"+strconv.Itoa(len([]byte(v)))+crlf)...)
		cmd = append(cmd, []byte(v+crlf)...)
	}

	r.ioBuf[i] = &ioBuf{
		buf: make(chan []byte),
		err: make(chan error),
		cmd: cmd,
	}

	r.wBuf <- i

	select {
	case err = <-r.ioBuf[i].err:
	case buf := <-r.ioBuf[i].buf:
		data, err = decoder(buf)
	}
	r.ioBuf[i].cmd = nil
	r.ioBuf[i].buf = nil
	r.ioBuf[i].err = nil

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
		n, err := r.Conn.Read(r.rBuf)
		if err != nil {
			r.ioBuf[i].err <- err
		}
		r.ioBuf[i].buf <- r.rBuf[:n]
	}
}

func decoder(buf []byte) (result interface{}, err error) {
	scanner := bufio.NewScanner(bytes.NewBuffer(buf))
	scanner.Split(splitFunc)
	result, err = analyzer(scanner)
	err = scanner.Err()
	return result, err
}

func analyzer(scanner *bufio.Scanner) (result interface{}, err error) {
	scanner.Scan()
	data := scanner.Bytes()
	if len(data) == 0 {
		return result, err
	}
	switch data[0] {
	case '-':
		result = nil
		err = errors.New(scanner.Text())
	case '+':
		result = string(data[1:])
	case ':':
		result, err = strconv.ParseInt(string(data[1:]), 10, 64)
	case '$':
		len := string(data[1:])
		if len == "0" {
			scanner.Scan()
			result = ""
		} else if len == "-1" {
			result = nil
		} else {
			scanner.Scan()
			result = string(scanner.Bytes()[0:])
		}
	case '*':
		len := string(data[1:])
		if len == "0" {
			result = make([]interface{}, 0)
		}
		if len == "-1" {
			result = nil
		}
		num, err := strconv.ParseInt(len, 10, 32)
		if err != nil {
			return nil, err
		}
		result = make([]interface{}, num)
		for i := range result.([]interface{}) {
			result.([]interface{})[i], _ = analyzer(scanner)
		}
	}
	return result, err
}

/*
For Simple Strings the first byte of the reply is "+"
For Errors the first byte of the reply is "-"
For Integers the first byte of the reply is ":"
For Bulk Strings the first byte of the reply is "$"
For Arrays the first byte of the reply is "*"
*/

func splitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	i := bytes.Index(data, []byte{'\r', '\n'})
	if i != -1 {
		return i + 2, data[:i], nil
	}
	if atEOF {
		return 0, data, bufio.ErrFinalToken
	}
	return 0, nil, nil
}
