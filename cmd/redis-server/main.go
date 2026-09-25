// redis/cmd/redis-server/main.go
package main

import (
	"flag"
	"fmt"

	"os"

	"redis/internal/replication"

	"redis/internal/server"
	"redis/internal/storage"
)





func main(){

port := flag.String("port", "6379", "Port to run the server on")
flag.Parse()




	db := storage.NewDatabase(100 * 1024 * 1024)
	aofFileName := "database_" + *port + ".aof"
	aofLog, err := storage.NewAOF(aofFileName)
	if err != nil {
		fmt.Println("Failed to initialize AOF:", err)
		os.Exit(1)
	}
	 broker := replication.NewBroker()

	 commandCh := make(chan server.Command, 100000)

	 engine := server.NewEngine(commandCh, db, aofLog,broker)
	 go engine.Start()
err = server.StartTCPServer(":"+*port, commandCh)

if err != nil {
		fmt.Println("Server failed to start:", err)
		os.Exit(1)
	}

}
