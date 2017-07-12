package redis

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"strconv"
	"time"
)

type resp struct {
	Conn     net.Conn
	WBuf     chan []byte
	RBuf     []byte
	ioBuf    *ioBuf
	i        uint32
	j        uint32
	TimeoutR time.Duration
	TimeoutW time.Duration
}

type ioBuf struct {
	buf [100000](chan []byte)
	err [100000](chan error)
}

const crlf = "\r\n"

func newRESP(url string) (r resp, err error) {
	r.Conn, err = net.DialTimeout("tcp", url, 5*time.Second)
	if err != nil {
		return r, err
	}
	r.WBuf = make(chan []byte)
	r.RBuf = make([]byte, 8192)
	r.ioBuf = &ioBuf{}
	r.i, r.j = 0, 0
	r.TimeoutW = 1 * time.Second
	r.TimeoutR = 2 * time.Second

	go r.listen()
	return r, nil
}

func (r *resp) cmd(args []string) (data interface{}, err error) {
	i := r.i
	r.i++
	if r.i == 100000 {
		r.i = 0
	}
	r.ioBuf.buf[i] = make(chan []byte)
	r.ioBuf.err[i] = make(chan error)
	go r.queueCmd(i, args)
	select {
	case err = <-r.ioBuf.err[i]:
	case buf := <-r.ioBuf.buf[i]:
		data, err = decoder(buf)
	}
	r.ioBuf.buf[i] = make(chan []byte)
	r.ioBuf.err[i] = make(chan error)
	return data, err
}

func (r *resp) queueCmd(i uint32, args []string) {
	buf := []byte("*" + strconv.Itoa(len(args)) + crlf)
	for _, v := range args {
		buf = append(buf, []byte("$"+strconv.Itoa(len([]byte(v)))+crlf)...)
		buf = append(buf, []byte(v+crlf)...)
	}
	r.WBuf <- buf
}

func (r *resp) listen() {
	for {
		cmd := <-r.WBuf
		j := r.j
		r.j++
		if r.j == 100000 {
			r.j = 0
		}
		timer := time.Now().Add(r.TimeoutW)
		r.Conn.SetWriteDeadline(timer)
		_, err := r.Conn.Write(cmd)
		if err != nil {
			r.ioBuf.err[j] <- err
			return
		}
		timer = time.Now().Add(r.TimeoutR)
		r.Conn.SetReadDeadline(timer)
		n, err := r.Conn.Read(r.RBuf)
		if err != nil {
			r.ioBuf.err[j] <- err
		}
		r.ioBuf.buf[j] <- r.RBuf[:n]
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
		num, _ := strconv.ParseInt(len, 10, 32)
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
