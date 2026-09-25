// cmd/replica/main.go
package main

import (
	"bytes"
	"flag"
	"fmt"
	"net"
	"os"
	"redis/internal/resp"
	"redis/internal/server"
	"redis/internal/storage"
)

func main() {
	port := flag.String("port", "6380", "Port to run the replica on")
	masterAddr := flag.String("replicaof", "127.0.0.1:6379", "Master address")
	flag.Parse()


	db := storage.NewDatabase(100 * 1024 * 1024)


	commandCh := make(chan server.Command, 100000)
	engine := server.NewEngine(commandCh, db, nil, nil)
	go engine.Start()


	fmt.Printf("Booting Replica Node. Syncing from Master at %s...\n", *masterAddr)
	masterConn, err := net.Dial("tcp", *masterAddr)
	if err != nil {
		fmt.Println("Failed to connect to Master:", err)
		os.Exit(1)
	}


	masterConn.Write([]byte("*1\r\n$4\r\nSYNC\r\n"))


	go func() {
		defer masterConn.Close()
		buffer := make([]byte, 4096)
		readIndex := 0

		for {
			n, err := masterConn.Read(buffer[readIndex:])
			if err != nil {
				fmt.Println("CRITICAL: Lost connection to Master.")
				os.Exit(1)
			}
			readIndex += n
			processIndex := 0


			for processIndex < readIndex {
				args, consumed, err := resp.Parse(buffer[processIndex:readIndex])

				if err == resp.ErrIncomplete {
					break
				}
				if err != nil {

					if buffer[processIndex] == '+' {
						crlf := bytes.Index(buffer[processIndex:readIndex], []byte("\r\n"))
						if crlf != -1 {
							processIndex += crlf + 2
							continue
						}
					}
					break
				}

				commandCh <- server.Command{
					Args:      args,
					Conn:      masterConn,
					RespondCh: make(chan []byte, 1),
				}
				processIndex += consumed
			}

			if processIndex == readIndex {
				readIndex = 0
			} else {
				copy(buffer, buffer[processIndex:readIndex])
				readIndex -= processIndex
			}
		}
	}()

	fmt.Printf("Replica ready for read-only traffic on :%s\n", *port)
	err = server.StartTCPServer(":"+*port, commandCh)
	if err != nil {
		fmt.Println("Replica server failed to start:", err)
		os.Exit(1)
	}
}
