package tests

import (
	"encoding/json"
	"fmt"
	"offchain-oracles/config"
	"offchain-oracles/signer"
	"offchain-oracles/signer/provider"
	"offchain-oracles/wavesapi"
	"offchain-oracles/wavesapi/models"
	"offchain-oracles/wavesapi/state"
	"strconv"
	"testing"

	"github.com/jarcoal/httpmock"

	"github.com/stretchr/testify/assert"
)

const (
	signPrefix = "WAVESNEUTRINOPREFIX"
)

func TestHandler(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	nodeUrl := "test.com"
	controlContract := "testContract"
	oracleAddress := "testOracle"
	height := 100
	testDb := "db/"
	testNode := wavesapi.New(nodeUrl, "")

	provider := provider.BinanceProvider{}

	httpmock.RegisterResponder("GET", "https://api.binance.com/api/v3/ticker/price?symbol=WAVESUSDT",
		httpmock.NewStringResponder(200, `{"price": "`+strconv.FormatFloat(1, 'f', 6, 64)+`"}`))
	httpmock.RegisterResponder("GET", "https://api.binance.com/api/v3/ticker/price?symbol=BTCUSDT",
		httpmock.NewStringResponder(200, `{"price": "`+strconv.FormatFloat(1, 'f', 6, 64)+`"}`))
	httpmock.RegisterResponder("GET", "https://api.binance.com/api/v3/ticker/price?symbol=WAVESBTC",
		httpmock.NewStringResponder(200, `{"price": "`+strconv.FormatFloat(1, 'f', 6, 64)+`"}`))

	jsonState, _ := json.Marshal([]state.State{
		{
			Key:   "oracles",
			Value: "",
			Type:  "string",
		},
	})

	httpmock.RegisterResponder("GET", nodeUrl+"/addresses/data/"+controlContract,
		httpmock.NewStringResponder(200, string(jsonState)))

	httpmock.RegisterResponder("GET", nodeUrl+"/blocks/height",
		httpmock.NewStringResponder(200, `{"height":`+strconv.Itoa(height)+`}`))

	expectedSignedText := models.SignedText{
		Message:   "123",
		PublicKey: "",
		Signature: "",
	}

	jsonSignedText, err := json.Marshal(expectedSignedText)
	if err != nil {
		fmt.Printf("Error:%s \n", err.Error())
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", nodeUrl+"/addresses/signText/"+oracleAddress,
		httpmock.NewStringResponder(200, string(jsonSignedText)))

	isTimeout, err := signer.HandleHeight(config.Config{
		NodeURL:         nodeUrl,
		ApiKey:          "",
		ControlContract: controlContract,
		Timeout:         0,
	}, oracleAddress, testDb, testNode, provider, false)

	if err != nil {
		fmt.Printf("Error:%s \n", err.Error())
	}
	assert.True(t, isTimeout)
	//	signedText, _ := storage.GetKeystore(testDb, height)
	//	assert.Equal(t, expectedSignedText, signedText)
	//assert.Equal(t, signPrefix+"_"+strconv.Itoa(height)+"_"+strconv.Itoa(int(expectedPrice*100)), signedText.Message)
}
