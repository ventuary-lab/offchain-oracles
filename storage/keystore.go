package storage

import (
	"encoding/binary"
	"encoding/json"
	"offchain-oracles/wavesapi/models"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
)

var heightLocker map[int]*sync.Mutex = make(map[int]*sync.Mutex)

func PutKeystore(path string, height int, text models.SignedText) error {
	_, ok := heightLocker[height]
	if !ok {
		heightLocker[height] = &sync.Mutex{}
	}
	heightLocker[height].Lock()
	defer heightLocker[height].Unlock()
	db, err := leveldb.OpenFile(path+"/"+"prices", nil)
	defer db.Close()
	if err != nil {
		return err
	}

	key := make([]byte, 8)
	binary.LittleEndian.PutUint64(key, uint64(height))

	byteJson, err := json.Marshal(text)
	if err != nil {
		return err
	}

	if err := db.Put(key, byteJson, nil); err != nil {
		return err
	}

	return nil
}

func GetKeystore(path string, height int) (models.SignedText, error) {
	_, ok := heightLocker[height]
	if !ok {
		heightLocker[height] = &sync.Mutex{}
	}
	heightLocker[height].Lock()
	defer heightLocker[height].Unlock()
	db, err := leveldb.OpenFile(path+"/"+"prices", nil)
	defer db.Close()

	if err != nil {
		return models.SignedText{}, err
	}

	key := make([]byte, 8)
	binary.LittleEndian.PutUint64(key, uint64(height))

	value, err := db.Get([]byte(key), nil)
	if err != nil {
		return models.SignedText{}, err
	}

	result := models.SignedText{}
	err = json.Unmarshal(value, &result)
	if err != nil {
		return models.SignedText{}, err
	}

	return result, err
}
