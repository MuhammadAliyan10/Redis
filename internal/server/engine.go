// internal/server/engine.go
package server

import (
	"bytes"
	"fmt"
	"redis/internal/replication"
	"redis/internal/storage"
	"strconv"
	"time"
)

type Engine struct {
	commandCh <-chan Command
	db        *storage.DataBase
	aof       *storage.AOF
	broker    *replication.Broker
}

func NewEngine(commandCh <-chan Command, db *storage.DataBase, aof *storage.AOF, broker *replication.Broker) *Engine {
	return &Engine{
		commandCh: commandCh,
		db:        db,
		aof:       aof,
		broker:    broker,
	}
}

func (e *Engine) Start() {
	fmt.Println("Lock-Free Execution Engine wired and active.")
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case cmd := <-e.commandCh:
			e.execute(cmd)
		case <-ticker.C:
			e.db.ActiveExpireCron()
		}
	}
}

func (e *Engine) execute(cmd Command) {
	if len(cmd.Args) == 0 {
		cmd.RespondCh <- []byte("-ERR empty command\r\n")
		return
	}

	cmdName := bytes.ToUpper(cmd.Args[0])
	var response []byte
	isMutation := false

	switch string(cmdName) {
	case "PING":
		if len(cmd.Args) > 1 {
			response = e.formatBulkString(cmd.Args[1])
		} else {
			response = []byte("+PONG\r\n")
		}

	case "SET":
		if len(cmd.Args) < 3 {
			response = []byte("-ERR wrong number of arguments for 'SET'\r\n")
			break
		}
		key := string(cmd.Args[1])
		val := cmd.Args[2]

		ttlMs := int64(0)
		if len(cmd.Args) == 5 && string(bytes.ToUpper(cmd.Args[3])) == "EX" {
			sec, _ := strconv.ParseInt(string(cmd.Args[4]), 10, 64)
			ttlMs = sec * 1000
		}

		e.db.Set(key, val, ttlMs)
		response = []byte("+OK\r\n")
		isMutation = true

	case "GET":
		if len(cmd.Args) != 2 {
			response = []byte("-ERR wrong number of arguments for 'GET'\r\n")
			break
		}
		obj, exists := e.db.Get(string(cmd.Args[1]))
		if !exists || obj.Type != storage.TypeString {
			response = []byte("$-1\r\n")
		} else {
			response = e.formatBulkString(obj.Ptr.([]byte))
		}

	case "DEL":
		if len(cmd.Args) != 2 {
			response = []byte("-ERR wrong number of arguments for 'DEL'\r\n")
			break
		}
		deleted := e.db.Delete(string(cmd.Args[1]))
		if deleted {
			response = []byte(":1\r\n")
			isMutation = true
		} else {
			response = []byte(":0\r\n")
		}

	case "HSET":
		if len(cmd.Args) != 4 {
			response = []byte("-ERR wrong number of arguments for 'HSET'\r\n")
			break
		}
		e.db.HSet(string(cmd.Args[1]), string(cmd.Args[2]), cmd.Args[3])
		response = []byte(":1\r\n")
		isMutation = true

	case "HGET":
		if len(cmd.Args) != 3 {
			response = []byte("-ERR wrong number of arguments for 'HGET'\r\n")
			break
		}
		val, exists := e.db.HGet(string(cmd.Args[1]), string(cmd.Args[2]))
		if !exists {
			response = []byte("$-1\r\n")
		} else {
			response = e.formatBulkString(val)
		}

	case "BGREWRITEAOF":
		snapshot := e.db.SnapShot()
		if e.aof != nil {
			go e.aof.RewriteAOF(snapshot, "database_6379.aof")
		}
		response = []byte("+Background append only file rewriting started\r\n")
	case "SYNC":
		snapshot := e.db.SnapShot()
		currentOffset := int64(0)
		if e.aof != nil{
			currentOffset = e.aof.GetOffset()
		}
		go func (respondCh chan<- []byte, snap map[string]*storage.Object)  {
			respondCh<- []byte("+FULLRESYNC\r\n")
			for key, obj := range snap {
				if obj.ExpiresAt > 0 && time.Now().UnixMilli() > obj.ExpiresAt {
					continue // Skip Ghost Keys
				}
				if obj.Type == storage.TypeString {
					cmdBytes := e.marshalCommand([][]byte{
						[]byte("SET"),
						[]byte(key),
						obj.Ptr.([]byte),
					})
					respondCh <- cmdBytes
				}
			}
		}(cmd.RespondCh, snapshot)

		if e.broker != nil {
			e.broker.AddReplica(cmd.Conn, currentOffset)
		}
		return

	default:
		response = []byte(fmt.Sprintf("-ERR unknown command '%s'\r\n", string(cmdName)))
	}

	if isMutation {
		rawRESP := e.marshalCommand(cmd.Args)
		if e.aof != nil {
			e.aof.Append(rawRESP)
		}
		if e.broker != nil {
			e.broker.Stream(rawRESP)
		}
	}

	select {
	case cmd.RespondCh <- response:
	default:
		fmt.Println("Client buffer full, severing connection.")
		cmd.Conn.Close()
	}
}

func (e *Engine) formatBulkString(data []byte) []byte {
	var buf bytes.Buffer
	buf.WriteByte('$')
	buf.WriteString(fmt.Sprintf("%d", len(data)))
	buf.WriteByte('\r')
	buf.WriteByte('\n')
	buf.Write(data)
	buf.WriteByte('\r')
	buf.WriteByte('\n')
	return buf.Bytes()
}

func (e *Engine) marshalCommand(args [][]byte) []byte {
	var buf bytes.Buffer
	buf.WriteByte('*')
	buf.WriteString(fmt.Sprintf("%d", len(args)))
	buf.WriteByte('\r')
	buf.WriteByte('\n')
	for _, arg := range args {
		buf.Write(e.formatBulkString(arg))
	}
	return buf.Bytes()
}
