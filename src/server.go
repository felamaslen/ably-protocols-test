package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"time"
)

func handleConnection(conn *net.Conn, store *Store) {
	buf := make([]byte, 256)
	tmp := make([]byte, 8)

	var isStateless bool

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

		if numChars >= 1 {
			isStateless = string(buf[32*8]) == "N"
			break
		}
	}

	if isStateless {
		handleStatelessConnection(conn, &buf, numChars)
	} else {
		handleStatefulConnection(store, conn, &buf, numChars)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	if len(os.Args) < 2 {
		log.Fatalf("Must provide port as first argument")
	}
	port, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatalf("Must provide integer as port")
	}

	store := Store{
		clients:            &map[string]*StoreClient{},
		expiredClientUuids: &map[string]bool{},
	}

	fmt.Printf("Listening on port %d\n", port)

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		panic(err)
	}

	for {
		conn, err := ln.Accept()
		if err == nil {
			go handleConnection(&conn, &store)
		} else {
			fmt.Printf("Error handling stateless connection: %v\n", err)
		}
	}
}
