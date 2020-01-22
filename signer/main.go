package main

import (
	"../wavesapi"
	"../wavesapi/transactions"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)
func main() {
	controlContract := "" // TODO: config
	var nodeClient = wavesapi.New("127.0.0.1:6869", "apiKey") // TODO: config
	contractState, err := nodeClient.GetStateByAddress(controlContract)
	if err != nil {
		panic(err)
	}

	oracles := strings.Split(contractState["oracles"].Value.(string), ",")
	var ips []string
	for _,v := range oracles {
		ips = append(ips, contractState[v + "ip"].Value.(string))
	}

	height, err := nodeClient.GetHeight()
	if err != nil {
		panic(err)
	}

	var signs []string
	var values []string

	//TODO: Get binance and sign
	for i,_ := range oracles {
		var client = &http.Client{Timeout: 10 * time.Second}
		res, err := client.Get(ips[i] + "/api/price/" + strconv.Itoa(height))
		if err != nil {
			panic(err)
		}

		if res.StatusCode == 200 {
			var result map[string]interface{}
			err = json.NewDecoder(res.Body).Decode(result)
			if err != nil {
				panic(err)
			}
			signs = append(signs, result["sign"].(string))
			values = append(values, result["value"].(string))
		}

		res.Body.Close()
	}

	if len(signs) > 3 {
		tx := transactions.New(transactions.InvokeScript, "sender") //TODO: config
		tx.NewInvokeScript(controlContract, transactions.FuncCall {
			Function: "finalizeCurrentPrice",
			Args: []transactions.FuncArg{
				{
					Value: strings.Join(signs, ","),
					Type: "string",
				},
				{
					Value: strings.Join(values, ","),
					Type: "string",
				},
			},
		}, nil, 500000)
	}

}
