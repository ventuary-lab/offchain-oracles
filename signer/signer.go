package signer

import (
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

	"github.com/syndtr/goleveldb/leveldb"
)

const (
	signPrefix = "WAVESNEUTRINOPREFIX"
)

func StartSigner(cfg config.Config, oracleAddress string, dbPath string) {
	var priceProvider provider.PriceProvider = provider.BinanceProvider{}

	var nodeClient = wavesapi.New(cfg.NodeURL, cfg.ApiKey)

	isTimeout := false
	for true {
		var err error
		isTimeout, err = HandleHeight(cfg, oracleAddress, dbPath, nodeClient, priceProvider, isTimeout)
		fmt.Printf("Error: %s \n", err.Error())
	}
}

func HandleHeight(cfg config.Config, oracleAddress string, dbPath string,
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

	ipsByPubKeyOracle := make(map[string][]string)
	for _, pubKeyOracle := range pubKeyOracles {
		ip, ok := contractState["ips_"+pubKeyOracle]
		if !ok {
			continue
		}

		ipsByPubKeyOracle[pubKeyOracle] = strings.Split(ip.Value.(string), ";")
	}

	signs := make(map[string]string)
	values := make(map[string]string)

	for pubKeyOracle, ips := range ipsByPubKeyOracle {
		var client = &http.Client{Timeout: 10 * time.Second}
		res, err := client.Get(ips[0] + "/api/price?height=" + strconv.Itoa(height))
		if err != nil {
			return false, err
		}

		if res.StatusCode == 200 {
			var result models.SignedText
			err = json.NewDecoder(res.Body).Decode(&result)
			if err != nil {
				return false, err
			}

			if pubKeyOracle != result.PublicKey {
				fmt.Printf("invalid pubKey (%s) \n", pubKeyOracle)
				continue
			}

			values[result.PublicKey] = strings.Split(result.Message, "_")[2]
			signs[result.PublicKey] = result.Signature
			fmt.Printf("Oracle %s: %s \n", result.PublicKey, values[result.PublicKey])
		}

		res.Body.Close()
	}

	signedPrice, err := storage.GetKeystore(dbPath, height)
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

		err = storage.PutKeystore(dbPath, height, signedText)
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
		sortedValues := ""
		sortedSigns := ""
		for _, pubKey := range pubKeyOracles {
			if len(sortedSigns) > 0 {
				sortedSigns += ","
			}
			if len(sortedValues) > 0 {
				sortedValues += ","
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
				sortedSigns += "q"
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
