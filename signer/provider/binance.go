package provider

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type BinanceProvider struct{}

func (BinanceProvider) PriceNow() (float32, error) {
	err, priceWavesUsdt := priceNowByPair("WAVESUSDT")
	if err != nil {
		return 0, err
	}

	err, priceWavesBtc := priceNowByPair("WAVESBTC")
	if err != nil {
		return 0, err
	}

	err, priceBtcUsdt := priceNowByPair("BTCUSDT")
	if err != nil {
		return 0, err
	}

	price := priceWavesUsdt + (priceWavesBtc*priceBtcUsdt)/2

	return price.nil
}

func priceNowByPair(pair string) (float32, error) {
	resp, err := http.Get("https://api.binance.com/api/v3/ticker/price?symbol=" + pair)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	jsonResponse := make(map[string]interface{})
	err = json.Unmarshal(body, &jsonResponse)
	if err != nil {
		return 0, err
	}

	return jsonResponse["price"].(float32), nil
}
