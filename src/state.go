package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"time"
)

type StoreClient struct {
	uuid string
	n    int64
	seq  []uint32

	progress int

	selfDestructTimer *time.Timer
}

func (c *StoreClient) getChecksum() string {
	marshaled, _ := json.Marshal(c.seq)
	return fmt.Sprintf("%x", md5.Sum(marshaled))
}

type Store struct {
	clients            *map[string]*StoreClient
	expiredClientUuids *map[string]bool
}

func (s *Store) has(uuid string) bool {
	return (*s.clients)[uuid] != nil
}

func (s *Store) set(uuid string, n int64, seq []uint32) error {
	if (*s.expiredClientUuids)[uuid] {
		return fmt.Errorf("Client UUID %s went away and is reserved\n", uuid)
	}

	client := StoreClient{
		uuid:     uuid,
		n:        n,
		seq:      seq,
		progress: 0,
	}

	(*s.clients)[uuid] = &client

	return nil
}

func (s *Store) get(uuid string) (client *StoreClient) {
	client = (*s.clients)[uuid]

	return
}

func (s *Store) progressClient(uuid string) {
	if !(*s).has(uuid) {
		return
	}
	(*s.clients)[uuid].progress++
}

func (s *Store) unset(uuid string) {
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
		(*s.expiredClientUuids)[uuid] = true
	}()
}

func (s *Store) keepalive(uuid string) {
	if (*s.clients)[uuid] == nil || (*s.clients)[uuid].selfDestructTimer == nil {
		return
	}
	(*s.clients)[uuid].selfDestructTimer.Reset(CONNECTION_DEADLINE)
}
