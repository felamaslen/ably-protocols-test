package main

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"time"
)

func readStatefulConnectionParameters(conn *net.Conn, buf *[]byte, numChars int) (uuid string, n int, m int, err error) {
	// Rudimentary data parser
	// example uuid:
	// 09ff8f0b-c0fb-4587-9173-807695f5b576
	tmp := make([]byte, 8)

	for {
		// Please see the stateless.go for a very similar implementation
		// If I had time I would write a library (or use JSON) to do this kind of thing
		char, err := (*conn).Read(tmp)
		numChars += char
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Read error: %v\n", err)
				panic(err)
			}
			break
		}

		*buf = append(*buf, tmp[:char]...)

		if numChars >= 1+36+5+5 {
			runes := []rune(string(*buf))

			offset := 1 + 32*8

			uuid = string(runes[offset : offset+36])

			n64, err := strconv.ParseInt(string(runes[offset+36:offset+36+5]), 10, 32)
			if err != nil {
				panic(err)
			}
			n = int(n64)

			m64, err := strconv.ParseInt(string(runes[offset+36+5:offset+36+5+5]), 10, 32)
			if err != nil {
				panic(err)
			}
			m = int(m64)

			break
		}
	}

	fmt.Printf("[stateful] Found: uuid=%v, n=%v, m=%v\n", uuid, n, m)

	if n < 1 || n > 0xffff {
		err = fmt.Errorf("n out of range: %v", n)
	}
	if m < 0 || m > n {
		err = fmt.Errorf("m out of range: %v", m)
	}

	return
}

func handleStatefulConnection(store *Store, conn *net.Conn, buf *[]byte, numChars int) {
	(*conn).SetDeadline(time.Now().Add(CONNECTION_DEADLINE))

	// Here, m keeps track of the index of the last value confirmed as received by the client
	uuid, n, m, err := readStatefulConnectionParameters(conn, buf, numChars)

	if err != nil {
		// TODO: client error
		panic(err)
	}

	if !store.has(uuid) {
		err = store.set(uuid, n, m)
		if err != nil {
			fmt.Printf("Error setting client in store: %s\n", err)
			(*conn).Write([]byte(fmt.Sprintf("%s", err)))
			(*conn).Close()
			return
		}
		store.setSelfDestructTimer(uuid)
	}

	// connectionTime := store.setConnectionTime(uuid)

	client := store.get(uuid)

	if m < client.length {
		var channel = client.getSequenceChannelFromIndex(m)

		for {
			nextValue, more := <-channel

			if nextValue != 0 {
				fmt.Printf("[stateful] sending %v\n", nextValue)
				_, err := (*conn).Write([]byte(fmt.Sprintf("%d\n", nextValue)))

				if err == nil {
					store.keepalive(uuid)
					(*conn).SetDeadline(time.Now().Add(CONNECTION_DEADLINE))
				} else {
					break
				}
			}

			if !more {
				break
			}

			time.Sleep(1 * time.Second)
			store.keepalive(uuid)
		}
	}

	checksum := client.getChecksum()

	client.selfDestructTimer.Stop()
	store.unset(uuid)

	fmt.Printf("[stateful] sending checksum: %v\n", checksum)

	_, err = (*conn).Write([]byte(fmt.Sprintf("checksum=%s\n", checksum)))
	if err != nil {
		fmt.Printf("Error writing checksum: %v\n", err)
	}

	_, err = (*conn).Write([]byte("EOF\n"))
	if err != nil {
		fmt.Printf("Error writing EOF: %v\n", err)
	}

	// This ensures we only delete the state if there has not been a subsequent connection
	// if store.has(uuid) && store.get(uuid).lastConnectionTime ==
}
