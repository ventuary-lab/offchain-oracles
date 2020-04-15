package signer

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"offchain-oracles/config"
	"offchain-oracles/signer/provider"
	"offchain-oracles/storage"
	"offchain-oracles/wavesapi"
	"offchain-oracles/wavesapi/models"
	"offchain-oracles/wavesapi/transactions"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcutil/base58"

	"github.com/syndtr/goleveldb/leveldb"
)

const (
	signPrefix = "WAVESNEUTRINOPREFIX"
)

func StartSigner(cfg config.Config, oracleAddress string, priceProvider provider.PriceProvider, db *leveldb.DB) {

	var nodeClient = wavesapi.New(cfg.NodeURL, cfg.ApiKey)

	isTimeout := false
	for true {
		var err error
		isTimeout, err = HandleHeight(cfg, oracleAddress, db, nodeClient, priceProvider, isTimeout)
		if err != nil {
			fmt.Printf("Error: %s \n", err.Error())
		}
	}
}

func HandleHeight(cfg config.Config, oracleAddress string, db *leveldb.DB,
	nodeClient wavesapi.Node, priceProvider provider.PriceProvider, isTimeout bool) (bool, error) {

	contractState, err := nodeClient.GetStateByAddress(cfg.ControlContract)
	if err != nil {
		return false, err
	}

	pubKeyOracles := strings.Split(contractState["oracles"].Value.(string), ",")

	height, err := nodeClient.GetHeight()
	if err != nil {
		return false, err
	}

	_, priceExist := contractState["price_"+strconv.Itoa(height)]
	if priceExist {
		return false, nil
	}

	signs := make(map[string]string)
	values := make(map[string]string)

	for _, ip := range cfg.Ips {
		var client = &http.Client{Timeout: 10 * time.Second}
		res, err := client.Get(ip + "/api/price?height=" + strconv.Itoa(height))
		if err != nil {
			fmt.Printf("Http error %s: %s \n", ip, err.Error())
			continue
		}

		if res.StatusCode == 200 {
			var result models.SignedText
			err = json.NewDecoder(res.Body).Decode(&result)
			if err != nil {
				fmt.Printf("Parse error %s: %s \n", ip, err.Error())
				continue
			}
			if result.Message == "" {
				fmt.Printf("Oracle (%s) %s: %s \n", ip, result.PublicKey, "empty msg")
				continue
			}
			values[result.PublicKey] = strings.Split(result.Message, "_")[2]
			signs[result.PublicKey] = result.Signature
			fmt.Printf("Oracle (%s) %s: %s \n", ip, result.PublicKey, values[result.PublicKey])
		}
		if res.Body != nil {
			if err := res.Body.Close(); err != nil {
				fmt.Printf("Http close error %s: %s \n", ip, err.Error())
				continue
			}
		}
	}

	signedPrice, err := storage.GetKeystore(db, height)
	if err != nil && err != leveldb.ErrNotFound {
		fmt.Printf("Error: %s \n", err.Error())
	} else {
		newNotConvertedPrice, err := priceProvider.PriceNow()
		if err != nil {
			return false, err
		}

		newPrice := int(newNotConvertedPrice * 100)
		msg := signPrefix + "_" + strconv.Itoa(height) + "_" + strconv.Itoa(newPrice)
		signedText, err := nodeClient.SignMsg(msg, oracleAddress)

		err = storage.PutKeystore(db, height, signedText)
		if err != nil {
			return false, err
		}
	}
	fmt.Printf("Price by {%d}: {%s} \n", height, signedPrice.Message)

	if !isTimeout {
		time.Sleep(time.Duration(cfg.Timeout) * time.Second)
		return true, nil
	}

	if _, ok := contractState["price_"+strconv.Itoa(height)]; len(signs) >= 3 && !ok {
		var funcArgs []transactions.FuncArg

		for _, pubKey := range pubKeyOracles {

			value, ok := values[pubKey]
			if ok {
				valueInt, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					funcArgs = append(funcArgs, transactions.FuncArg{
						Type:  "integer",
						Value: 0,
					})
				}
				funcArgs = append(funcArgs, transactions.FuncArg{
					Type:  "integer",
					Value: valueInt,
				})
			} else {
				funcArgs = append(funcArgs, transactions.FuncArg{
					Type:  "integer",
					Value: 0,
				})
			}

			sign, ok := signs[pubKey]
			bytesSign := base58.Decode(sign)
			base64Sing := base64.StdEncoding.EncodeToString(bytesSign)
			if ok {
				funcArgs = append(funcArgs, transactions.FuncArg{
					Type:  "binary",
					Value: "base64:" + base64Sing,
				})
			} else {
				funcArgs = append(funcArgs, transactions.FuncArg{
					Type:  "binary",
					Value: "",
				})
			}
		}

		tx := transactions.New(transactions.InvokeScript, oracleAddress)
		tx.NewInvokeScript(cfg.ControlContract, transactions.FuncCall{
			Function: "finalizeCurrentPrice",
			Args:     funcArgs,
		}, nil, 500000)
		err = nodeClient.SignTx(&tx)
		if err != nil {
			return false, err
		}

		err = nodeClient.Broadcast(tx)
		if err != nil {
			return false, err
		}

		err = <-nodeClient.WaitTx(tx.ID)
		if err != nil {
			return false, err
		}
		fmt.Printf("Tx finilize: %s \n", tx.ID)
	}

	time.Sleep(1 * time.Second)
	return false, nil
}
