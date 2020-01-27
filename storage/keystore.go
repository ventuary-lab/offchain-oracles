package storage

import (
	"encoding/binary"
	"encoding/json"
	"offchain-oracles/wavesapi/models"

	"github.com/syndtr/goleveldb/leveldb"
)

const (
	path = "db/"
)

func PutKeystore(height int, text models.SignedText) error {
	db, err := leveldb.OpenFile(path+"/"+priceFeedPath, nil)
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

func GetKeystore(height int) (models.SignedText, error) {
	db, err := leveldb.OpenFile(path+"/"+priceFeedPath, nil)
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
