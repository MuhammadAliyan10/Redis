package storage

import (
	"container/list"
	"time"
)


type ObjectType uint8

const (
	TypeString ObjectType = iota
	TypeHash
	TypeList
	TypeSet
)

type Object struct{
	Type ObjectType
	Ptr   any
	ExpiresAt int64
	Size int64
}

type DataBase struct{
	dict map[string]*Object
	ttlMap map[string]struct{}
	lurList *list.List
	lurNodes map[string]*list.Element
	maxMemory int64
	usedMemory int64
}


func NewDatabase(maxMemoryBytes int64) *DataBase{
	return &DataBase{
		dict: make(map[string]*Object),
		ttlMap: make(map[string]struct{}),
		lurList: list.New(),
		lurNodes: make(map[string]*list.Element),
		maxMemory: maxMemoryBytes,
	}
}


func (db *DataBase) updateLUR(key string){
	if db.maxMemory <= 0{
		return
	}
	if node, exists := db.lurNodes[key]; exists{
		db.lurList.MoveToFront(node)
	} else{
		db.lurNodes[key] = db.lurList.PushFront(key)
	}
}


func (db *DataBase) enforceMaxMemory(){
	if db.maxMemory <= 0{
		return
	}

	for db.usedMemory > db.maxMemory && db.lurList.Len() > 0{
		back := db.lurList.Back()
		if back != nil{
			evictKey := back.Value.(string)
			db.Delete(evictKey)
		}
	}
}

func (db *DataBase) Get(key string) (*Object, bool){
	obj, exists := db.dict[key]
	if !exists{
		return nil, false
	}
	if obj.ExpiresAt > 0 && time.Now().UnixMilli() > obj.ExpiresAt{
		db.Delete(key)
		return nil, false
	}
	db.updateLUR(key)
	return obj, true
}


func (db *DataBase) Set(key string, val []byte, ttlMs int64){
	db.Delete(key)

	expiresAt := int64(0)
	if ttlMs > 0{
		expiresAt = time.Now().UnixMilli() + ttlMs
		db.ttlMap[key] = struct{}{}
	}
	size := int64(len(key) + len(val) + 32)
	obj := &Object{
		Type:      TypeString,
		Ptr:       val,
		ExpiresAt: expiresAt,
		Size:      size,
	}

	db.dict[key] = obj
	db.usedMemory += size

	db.updateLUR(key)
	db.enforceMaxMemory()

}

func (db *DataBase) Delete(key string) bool{
	obj, exists := db.dict[key]
	if !exists{
		return false
	}

	db.usedMemory -= obj.Size
	delete(db.dict, key)
	delete(db.ttlMap, key)

	if node, ok := db.lurNodes[key]; ok{
		db.lurList.Remove(node)
		delete(db.lurNodes, key)
	}
	return true
}



func (db *DataBase) ActiveExpireCron(){
	now := time.Now().UnixMilli()
	samples := 20
	expiredCount := 0

	for key := range db.ttlMap{
		if samples <= 0{
			break
		}
		samples --

		obj, exists := db.dict[key]
		if exists && obj.ExpiresAt > 0 && now > obj.ExpiresAt{
			db.Delete(key)
			expiredCount++
		}
	}

	if expiredCount > 5{
		db.ActiveExpireCron()
	}


}


func (db *DataBase) HSet(key string, field string, val []byte){
	obj, exists := db.Get(key)
	var hash map[string][]byte
	if !exists || obj.Type != TypeHash{
		db.Delete(key)
		hash = make(map[string][]byte)
		obj = &Object{
			Type: TypeHash,
			Ptr : hash,
			Size: int64(len(key) + 32),
		}

		db.dict[key] = obj
		db.updateLUR(key)
	} else{
		hash = obj.Ptr.(map[string][]byte)
	}

	oldVal, fieldExists := hash[field]

	if fieldExists{
		db.usedMemory -= int64(len(oldVal))

	} else {
			db.usedMemory += int64(len(field))
				obj.Size += int64(len(field))
	}

	hash[field] = val
	db.usedMemory += int64(len(val))
	obj.Size += int64(len(val))
	db.enforceMaxMemory()

}

func (db *DataBase) HGet(key string, field string) ([]byte, bool){
	obj, exists := db.Get(key)
	if !exists || obj.Type != TypeHash{
		return nil, false
	}

	hash := obj.Ptr.(map[string][]byte)

	val,ok := hash[field]
	return val, ok
}
