package storage

import (
	"bufio"
	"os"
	"sync"
	"time"
)




type AOF struct{
	file *os.File
	writer *bufio.Writer
	writeChan chan []byte
	offset int64
	offsetMu sync.RWMutex
}

func NewAOF(filename string) (*AOF, error){
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_APPEND,0666)
	if err != nil{
		return nil, err
	}
	stat , _ := f.Stat()
	aof := &AOF{
		file: f,
		writer: bufio.NewWriterSize(f, 65536),
		writeChan: make(chan []byte, 100000),
		offset: stat.Size(),

	}
	go aof.groupCommitWorker()
	return aof, nil
}


func (a *AOF) Append(cmdBytes []byte){
	a.writeChan <- cmdBytes
}

func (a *AOF) groupCommitWorker(){
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
			select{
			case data := <- a.writeChan:
				n, err := a.writer.Write(data)
				if err == nil{
					a.offsetMu.Lock()
					a.offset += int64(n)
					a.offsetMu.Unlock()
				}
				case <- ticker.C:
					a.writer.Flush()
					a.file.Sync()
			}
	}
}

func (a *AOF) GetOffset() int64 {
	a.offsetMu.RLock()
	defer a.offsetMu.RUnlock()
	return a.offset
}
