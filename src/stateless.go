package main

import (
	"fmt"
	"io"
	"math"
	"net"
	"strconv"
	"time"
)

func readStatelessConnectionParameters(conn *net.Conn) (a int64, n int64, m int64, err error) {
	// This is an extremely rudimentary connection data parser which just
	// takes in two fixed-size data points (a and n)
	//
	// For a more sophisticated app, use JSON / XML / etc.
	// (actually please don't use XML)
	buf := make([]byte, 256)
	tmp := make([]byte, 8)

	numChars := 0

	for {
		char, err := (*conn).Read(tmp)
		numChars += char
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Read error: %v\n", err)
				panic(err)
			}
			break
		}

		buf = append(buf, tmp[:char]...)

		// A in 1..0xff, N in 1..0xffff, M in 1..0xffff where M is "N so far"
		// We read these numbers in from the connection data
		// They are passed as decimals with leading zeroes, i.e.
		// A in 1..255 (3 chars), N, M in 1..65535 (5 chars)
		if numChars >= 3+5+5 {
			// This is necessary to handle UTF-8 input
			runes := []rune(string(buf))

			offset := 32 * 8

			a, err = strconv.ParseInt(string(runes[offset:offset+3]), 10, 16)
			n, err = strconv.ParseInt(string(runes[offset+3:offset+3+5]), 10, 32)
			m, err = strconv.ParseInt(string(runes[offset+3+5:offset+3+5+5]), 10, 32)

			fmt.Printf("[stateless] Found: a=%v, n=%v, m=%v, runes=%v\n", a, n, m, string(runes))

			break
		}
	}

	if a < 0 || a > 0xff {
		err = fmt.Errorf("a out of range: %v", a)
	}
	if n < 1 || n > 0xffff {
		err = fmt.Errorf("n out of range: %v", n)
	}

	return
}

func handleStatelessConnection(conn *net.Conn) {
	(*conn).SetDeadline(time.Now().Add(CONNECTION_DEADLINE))

	a, n, m, err := readStatelessConnectionParameters(conn)
	if err != nil {
		// TODO: client error
		panic(err)
	}

	if a == 0 {
		a = int64(generateInitialNumber())
		fmt.Printf("Handled initial connection; a=%v, n=%v\n", a, n)
	}

	a *= int64(math.Pow(2, float64(m)))

	i := int64(0)

	for {
		fmt.Printf("Sending %d\n", a)
		_, err = (*conn).Write([]byte(fmt.Sprintf("%d\n", a)))
		if err == nil {
			(*conn).SetDeadline(time.Now().Add(CONNECTION_DEADLINE))
		} else {
			(*conn).Close()
		}

		i++

		if i < n {
			a *= 2
			time.Sleep(1 * time.Second)
		} else {
			_, err = (*conn).Write([]byte("EOF\n"))
			if err != nil {
				fmt.Printf("Error writing EOF: %v\n", err)
			}
			break
		}
	}

	fmt.Printf("Finished connection; a=%v\n", a)
}

func listenStateless() {
	fmt.Printf("Listening (stateless) on port %d\n", PORT_STATELESS)

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", PORT_STATELESS))

	if err != nil {
		panic(err)
	}

	for {
		conn, err := ln.Accept()
		if err == nil {
			go handleStatelessConnection(&conn)
		} else {
			fmt.Printf("Error handling stateless connection: %v\n", err)
		}
	}
}
