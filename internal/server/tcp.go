package server

import (
	"fmt"
	"io"
	"net"
	"redis/internal/resp"
)


type Command struct{
	Args [][]byte
	Conn net.Conn
	RespondCh chan<- []byte
}

func StartTCPServer(addr string, commandCh chan <- Command) error{
	listner, err := net.Listen("tcp", addr)
	if err != nil{
		return err
	}
	defer listner.Close()
	fmt.Printf("TCP Acceptor active on %s\n", addr)

	for {
		conn, err := listner.Accept()
		if err != nil {
			fmt.Printf("Failed to accept connection: %v\n", err)
			continue
		}
		go handleConnection(conn, commandCh)
	}

}


func handleConnection(conn net.Conn, commandCh chan <- Command){
	defer conn.Close()

	respondCh := make(chan []byte, 1024)

	go func(){
		for response := range respondCh{
		_, err := conn.Write(response)
		if err != nil{
			return
		}
		}
	}()

	buffer := make([]byte, 4096)
	readIndex := 0

	for {
		n, err := conn.Read(buffer[readIndex:])
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Connection read error: %v\n", err)
			}
			return
		}
		readIndex += n
		processIndex := 0

		for processIndex < readIndex {
			args, consumed, err := resp.Parse(buffer[processIndex:readIndex])

			if err == resp.ErrIncomplete {
				break
			}
			if err != nil {
				fmt.Printf("Protocol error: %v\n", err)
				return
			}
			commandCh <- Command{
				Args: args,
				Conn: conn,
				RespondCh: respondCh,
			}
			processIndex += consumed
		}
		if processIndex == readIndex {
			readIndex = 0
		} else{
			copy(buffer, buffer[processIndex:readIndex])
			readIndex -= processIndex
		}



	}
}
