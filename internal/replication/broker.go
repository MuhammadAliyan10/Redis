package replication

import (
	"fmt"
	"net"
	"sync"
)

type Replica struct{
	Conn net.Conn
	Offset int64
}


type Broker struct{
	mu sync.RWMutex
	replicas map[net.Conn]*Replica
	streamChan chan []byte
}


func NewBroker() *Broker{
	b := &Broker{
		replicas: make(map[net.Conn]*Replica),
		streamChan: make(chan []byte, 100000),
	}
	go b.broadcasterWorker()
	return b
}

func (b *Broker) AddReplica(conn net.Conn, offset int64){
	b.mu.Lock()
	defer b.mu.Unlock()

	b.replicas[conn] = &Replica{
		Conn: conn,
		Offset: offset,
	}
	fmt.Printf("Replica registered. Cluster tracking %d active replicas.\n", len(b.replicas))
}

func (b *Broker) RemoveReplica(conn net.Conn){
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.replicas, conn)
}

func (b *Broker) Stream(cmdBytes []byte){
	select{
	case b.streamChan <- cmdBytes:
	default:
		fmt.Println("CRITICAL: Replication stream buffer full. Dropping broadcast.")
	}
}

func (b *Broker) broadcasterWorker() {
	for data := range b.streamChan {
		b.mu.RLock()
		for conn, replica := range b.replicas {
			_, err := conn.Write(data)
			if err != nil {
				fmt.Printf("Replica connection lost: %v\n", err)
				conn.Close()
			} else {
				replica.Offset += int64(len(data))
			}
		}
		b.mu.RUnlock()
	}
}
