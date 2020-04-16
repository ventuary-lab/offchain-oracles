package provider

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type HuobiProvider struct{}

func (p *HuobiProvider) PriceNow() (float64, error) {
	priceWavesUsdt, err := p.PriceNowByPair("wavesusdt")
	if err != nil {
		return 0, err
	}

	priceWavesBtc, err := p.PriceNowByPair("wavesbtc")
	if err != nil {
		return 0, err
	}

	priceBtcUsdt, err := p.PriceNowByPair("btcusdt")
	if err != nil {
		return 0, err
	}

	price := (priceWavesUsdt + (priceWavesBtc * priceBtcUsdt)) / 2

	return price, nil
}

func (HuobiProvider) PriceNowByPair(pair string) (float64, error) {
	resp, err := http.Get("https://api.huobi.pro/market/history/trade?symbol=" + pair)
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
	price := jsonResponse["data"].([]interface{})[0].(map[string]interface{})["data"].([]interface{})[0].(map[string]interface{})["price"].(float64)
	return price, nil
}
