package storage

import (
	"encoding/binary"
	"github.com/syndtr/goleveldb/leveldb"
)

const (
	path = "db/"
)

func PutKeystore(height int64, value int64) error {
	db, err := leveldb.OpenFile(path+"/"+ priceFeedPath, nil)
	defer db.Close()
	if err != nil {
		return err
	}

	key := make([]byte, 8)
	binary.LittleEndian.PutUint64(key, uint64(height))

	byteValue := make([]byte, 8)
	binary.LittleEndian.PutUint64(byteValue, uint64(value))


	if err := db.Put([]byte(key), byteValue, nil); err != nil {
		return err
	}

	return nil
}

func GetKeystore(height int64) (int64, error) {
	db, err := leveldb.OpenFile(path+"/"+priceFeedPath, nil)
	defer db.Close()

	if err != nil {
		return 0, err
	}

	key := make([]byte, 8)
	binary.LittleEndian.PutUint64(key, uint64(height))

	value, err := db.Get([]byte(key), nil)
	if err != nil {
		return 0, err
	}

	return int64(binary.LittleEndian.Uint64(value)), nil
}
