package main

import (
	"fmt"
	"io"
	"math/rand"
	"net"
	"strconv"
	"time"
)

func readStatefulConnectionParameters(conn *net.Conn) (n int64, uuid string, err error) {
	// Rudimentary data parser
	// example uuid:
	// 09ff8f0b-c0fb-4587-9173-807695f5b576
	buf := make([]byte, 256)
	tmp := make([]byte, 8)

	numChars := 0

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

		buf = append(buf, tmp[:char]...)

		if numChars >= 5+36 {
			runes := []rune(string(buf))

			offset := 32 * 8

			n, err = strconv.ParseInt(string(runes[offset:offset+5]), 10, 32)

			uuid = string(runes[offset+5 : offset+5+36])

			break
		}
	}

	fmt.Printf("[stateful] Found: n=%v, uuid=%v\n", n, uuid)

	if n < 1 || n > 0xffff {
		err = fmt.Errorf("n out of range: %v", n)
	}

	return
}

func getRandomSequence(n int64) (seq []uint32) {
	seq = []uint32{}
	for i := int64(0); i < n; i++ {
		seq = append(seq, rand.Uint32())
	}

	return
}

func handleStatefulConnection(conn *net.Conn, store *Store) {
	(*conn).SetDeadline(time.Now().Add(CONNECTION_DEADLINE))

	n, uuid, err := readStatefulConnectionParameters(conn)

	if err != nil {
		// TODO: client error
		panic(err)
	}

	if !store.has(uuid) {
		fmt.Printf("creating client\n")
		seq := getRandomSequence(n)
		err = store.set(uuid, n, seq)
		if err != nil {
			fmt.Printf("Error setting client in store: %s\n", err)
			(*conn).Write([]byte(fmt.Sprintf("%s", err)))
			(*conn).Close()
			return
		}
		store.setSelfDestructTimer(uuid)
	}

	for {
		client := store.get(uuid)

		if client.progress >= len(client.seq) {
			break
		}

		// Using redis, I would run LREM to retrieve the first item and remove it in one op
		// This should avoid race conditions when multiple servers are running
		nextValue := client.seq[client.progress]

		store.progressClient(uuid)

		fmt.Printf("[stateful] sending %v\n", nextValue)
		_, err := (*conn).Write([]byte(fmt.Sprintf("%d\n", nextValue)))

		if err == nil {
			store.keepalive(uuid)
			(*conn).SetDeadline(time.Now().Add(CONNECTION_DEADLINE))
		}

		time.Sleep(1 * time.Second)
		store.keepalive(uuid)
	}

	client := store.get(uuid)

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
}

func listenStateful() {
	fmt.Printf("Listening (stateful) on port %d\n", PORT_STATEFUL)

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", PORT_STATEFUL))
	store := Store{
		clients:            &map[string]*StoreClient{},
		expiredClientUuids: &map[string]bool{},
	}

	if err != nil {
		panic(err)
	}

	for {
		conn, err := ln.Accept()
		if err == nil {
			go handleStatefulConnection(&conn, &store)
		} else {
			fmt.Printf("Error handling stateful connection: %v\n", err)
		}
	}
}
