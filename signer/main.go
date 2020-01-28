package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"offchain-oracles/signer/config"
	"offchain-oracles/signer/provider"
	"offchain-oracles/storage"
	"offchain-oracles/wavesapi"
	"offchain-oracles/wavesapi/models"
	"offchain-oracles/wavesapi/transactions"
	"strconv"
	"strings"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
)

const (
	defaultConfigFileName = "config.json"
	signPrefix            = "WAVESNEUTRINOPREFIX"
)

func main() {
	var confFileName string
	var oracleAddress string
	flag.StringVar(&confFileName, "config", defaultConfigFileName, "set config path")
	flag.StringVar(&confFileName, "oracleAddress", "", "set oracle address")
	flag.Parse()

	cfg, err := config.Load(confFileName)
	if err != nil {
		panic(err)
	}

	var priceProvider provider.PriceProvider = provider.BinanceProvider{}

	var nodeClient = wavesapi.New(cfg.NodeURL, cfg.ApiKey)

	for true {
		contractState, err := nodeClient.GetStateByAddress(cfg.ControlContract)
		if err != nil {
			panic(err)
		}

		pubKeyOracles := strings.Split(contractState["oracles"].Value.(string), ",")

		height, err := nodeClient.GetHeight()
		if err != nil {
			panic(err)
		}

		_, priceExist := contractState["price_"+strconv.Itoa(height)]
		if priceExist {
			continue
		}

		signs := make(map[string]string)
		values := make(map[string]string)

		for _, ip := range cfg.OraclesIp {
			var client = &http.Client{Timeout: 10 * time.Second}
			res, err := client.Get(ip + "/api/price?height=" + strconv.Itoa(height))
			if err != nil {
				panic(err)
			}

			if res.StatusCode == 200 {
				var result models.SignedText
				err = json.NewDecoder(res.Body).Decode(result)
				if err != nil {
					panic(err)
				}

				isValidPubKey := false
				for _, v := range pubKeyOracles {
					if v == result.PublicKey {
						isValidPubKey = true
					}
				}

				if !isValidPubKey {
					fmt.Printf("invalid pubKey (%s)", ip)
				}

				values[result.PublicKey] = strings.Split(result.Message, ",")[1]
				signs[result.PublicKey] = result.Signature
			}

			res.Body.Close()
		}

		signedPrice, err := storage.GetKeystore(height)
		if err != nil && err != leveldb.ErrNotFound {
			panic(err)
		} else {
			newNotConvertedPrice, err := priceProvider.PriceNow()
			if err != nil {
				panic(err)
			}
			newPrice := int(newNotConvertedPrice * 100)
			msg := signPrefix + "_" + strconv.Itoa(newPrice) + "_" + strconv.Itoa(height)
			signedText, err := nodeClient.SignMsg(msg, oracleAddress)
			err = storage.PutKeystore(height, signedText)
			if err != nil {
				panic(err)
			}
		}
		fmt.Printf("History price: {%s}", signedPrice.Message)

		if _, ok := contractState["price_" + strconv.Itoa(height)]; len(signs) + 1  >= 3 && !ok {
			sortedValues := ""
			sortedSigns := ""
			for _, pubKey := range pubKeyOracles {
				if len(sortedSigns) > 0 {
					sortedSigns += ";"
				}
				if len(sortedValues) > 0 {
					sortedValues += ";"
				}

				value, ok := values[pubKey]
				if ok {
					sortedValues += value
				} else {
					sortedValues += "0"
				}

				sign, ok := signs[pubKey]
				if ok {
					sortedSigns += sign
				} else {
					sortedSigns += "0"
				}
			}

			tx := transactions.New(transactions.InvokeScript, oracleAddress)
			tx.NewInvokeScript(cfg.ControlContract, transactions.FuncCall{
				Function: "finalizeCurrentPrice",
				Args: []transactions.FuncArg{
					{
						Value: sortedValues,
						Type:  "string",
					},
					{
						Value: sortedSigns,
						Type:  "string",
					},
				},
			}, nil, 500000)

			err = nodeClient.Broadcast(tx)
			if err != nil {
				panic(err)
			}

			err = <- nodeClient.WaitTx(tx.ID)
			if err != nil {
				panic(err)
			}
		}
	}
}
