package storage

import (
	"encoding/binary"
	"encoding/json"
	"offchain-oracles/wavesapi/models"

	"github.com/syndtr/goleveldb/leveldb"
)

func PutKeystore(db *leveldb.DB, height int, text models.SignedText) error {
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

func GetKeystore(db *leveldb.DB, height int) (models.SignedText, error) {
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
