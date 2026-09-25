package storage

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"time"
)


func (db *DataBase) SnapShot() map[string]*Object{
	clone := make(map[string]*Object, len(db.dict))

	for k, v := range db.dict{
		var ptrCopy any
		switch v.Type {
		case TypeString:
			orignalBytes := v.Ptr.([]byte)
			newBytes := make([]byte, len(orignalBytes))
			copy(newBytes, orignalBytes)
			ptrCopy = newBytes
		case TypeHash:
			orginalHash := v.Ptr.(map[string][]byte)
			newHash := make(map[string][]byte, len(orginalHash))
			for hk, hv := range orginalHash{
				newHb := make([]byte, len(hv))
				copy(newHb, hv)
				newHash[hk] = newHb
			}
			ptrCopy = newHash
		}
		clone[k] = &Object{
			Type:      v.Type,
			Ptr:       ptrCopy,
			ExpiresAt: v.ExpiresAt,
			Size:      v.Size,
		}

	}
	return clone
}



func marshalCommand(args [][]byte) []byte {
	var buf bytes.Buffer
	buf.WriteByte('*')
	buf.WriteString(fmt.Sprintf("%d", len(args)))
	buf.WriteByte('\r')
	buf.WriteByte('\n')
	for _, arg := range args {
		buf.WriteByte('$')
		buf.WriteString(fmt.Sprintf("%d", len(arg)))
		buf.WriteByte('\r')
		buf.WriteByte('\n')
		buf.Write(arg)
		buf.WriteByte('\r')
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}


func (a *AOF) RewriteAOF(snapshot map[string]*Object, originalFilename string){
	tempFile := originalFilename + ".temp"
	f, err := os.OpenFile(tempFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		fmt.Printf("BGREWRITEAOF Failed to open temp file: %v\n", err)
		return
	}
	writer := bufio.NewWriterSize(f, 65536)

	for key, obj := range snapshot {
		if obj.ExpiresAt > 0 && time.Now().UnixMilli() > obj.ExpiresAt {
			continue
		}
		switch obj.Type {
		case TypeString:
			// Reconstruct the raw RESP SET command
			cmd := marshalCommand([][]byte{
				[]byte("SET"),
				[]byte(key),
				obj.Ptr.([]byte),
			})
			writer.Write(cmd)

			// If it has a TTL, append the PEXPIREAT command
			if obj.ExpiresAt > 0 {
				expireCmd := marshalCommand([][]byte{
					[]byte("PEXPIREAT"),
					[]byte(key),
					[]byte(strconv.FormatInt(obj.ExpiresAt, 10)),
				})
				writer.Write(expireCmd)
			}
		}
	}
	// Flush to OS and force hardware disk sync
	writer.Flush()
	f.Sync()
	f.Close()
	// Atomic rename to instantly replace the old bloated AOF with the perfect snapshot
	a.offsetMu.Lock()
	os.Rename(tempFile, originalFilename)
	a.offsetMu.Unlock()

	fmt.Println("BGREWRITEAOF Complete: Log compacted successfully.")
	}

