package commands

import (
	"redis/internal/resp"
	"redis/internal/storage"
	"strconv"
)

func RegisterStreamCommands(r *Registry){
	r.Register("XADD", xaddCommand)
	r.Register("XREAD", xreadCommand)
}

func xaddCommand(args []resp.Value, db storage.Engine) resp.Value{
	if len(args) < 2{
			return resp.Value{Type: "error", Str: "ERR wrong number of arguments for 'xadd' command"}
	}
	key := args[0].Bulk
	value := []byte(args[1].Bulk)

	id := db.XAdd(key, value)

	return resp.Value{Type: "integer", Num: id}
}

func xreadCommand(args []resp.Value, db storage.Engine) resp.Value{
	if len(args) < 2 {
		return resp.Value{Type: "error", Str: "ERR wrong number of arguments for 'xread' command"}
	}

	key := args[0].Bulk
	idStr := args[1].Bulk

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return resp.Value{Type: "error", Str: "ERR invalid stream ID"}
	}

	msg, exists := db.XRead(key, id)
	if !exists {
		return resp.Value{Type: "null"}
	}
	return resp.Value{Type: "bulk", Bulk: string(msg.Value)}
}
