package main

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"
)

type StoreClient struct {
	uuid   string
	length int

	selfDestructTimer *time.Timer
}

func (c *StoreClient) getPRNG() *rand.Rand {
	seed := binary.BigEndian.Uint64([]byte(c.uuid))
	return rand.New(rand.NewSource(int64(seed)))
}

func (c *StoreClient) getSequenceChannelFromIndex(i int) chan uint32 {
	var channel = make(chan uint32)

	go func() {
		r := c.getPRNG()

		for j := 0; j < c.length; j++ {
			nextValue := r.Uint32()
			if j >= i {
				channel <- nextValue
			}
		}

		close(channel)
	}()

	return channel
}

func (c *StoreClient) getChecksum() string {
	var seq []uint32
	var channel = c.getSequenceChannelFromIndex(0)

	for {
		value, more := <-channel

		if value != 0 {
			seq = append(seq, value)
		}

		if !more {
			break
		}
	}

	marshaled, _ := json.Marshal(seq)

	result := fmt.Sprintf("%x", md5.Sum(marshaled))

	fmt.Printf("getChecksum() result=%v, time=%v\n", result, c.uuid)

	return result
}

type Store struct {
	clients            *map[string]*StoreClient
	expiredClientUuids *map[string]bool
}

func (s *Store) has(uuid string) bool {
	return (*s.clients)[uuid] != nil
}

func (s *Store) set(uuid string, n int, m int) error {
	if (*s.expiredClientUuids)[uuid] {
		return fmt.Errorf("Client UUID %s went away and is reserved\n", uuid)
	}

	client := StoreClient{
		uuid:   uuid,
		length: n,
	}

	(*s.clients)[uuid] = &client

	return nil
}

func (s *Store) setConnectionTime(uuid string) int64 {
	if !s.has(uuid) {
		return 0
	}
	now := time.Now().UnixNano()
	return now
}

func (s *Store) get(uuid string) (client *StoreClient) {
	client = (*s.clients)[uuid]

	return
}

func (s *Store) unset(uuid string) {
	log.Printf("Removing client state: %v\n", uuid)
	delete(*s.clients, uuid)
	delete(*s.expiredClientUuids, uuid)
}

func (s *Store) setSelfDestructTimer(uuid string) {
	// If using redis, you would just set EX 30 on the key and it would
	// delete it automatically. Then just SET the item on a timer whilst
	// connected
	if (*s.clients)[uuid] == nil {
		return
	}
	client := (*s.clients)[uuid]
	if (*client).selfDestructTimer != nil {
		(*client).selfDestructTimer.Stop()
	}
	(*client).selfDestructTimer = time.NewTimer(CONNECTION_DEADLINE)
	go func() {
		<-(*client.selfDestructTimer).C
		s.unset(uuid)
		log.Printf("Marking client as went away: %v\n", uuid)
		(*s.expiredClientUuids)[uuid] = true
	}()
}

func (s *Store) keepalive(uuid string) {
	if (*s.clients)[uuid] == nil || (*s.clients)[uuid].selfDestructTimer == nil {
		return
	}
	(*s.clients)[uuid].selfDestructTimer.Reset(CONNECTION_DEADLINE)
}
