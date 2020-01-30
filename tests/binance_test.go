package tests

import (
	"offchain-oracles/signer/provider"
	"strconv"
	"testing"

	"github.com/jarcoal/httpmock"

	"github.com/stretchr/testify/assert"
)

func TestBinance(t *testing.T) {
	priceWavesUsdt := 1.0
	priceBtcUsdt := 1.20
	priceWavesBtc := 1.5

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://api.binance.com/api/v3/ticker/price?symbol=WAVESUSDT",
		httpmock.NewStringResponder(200, `{"price": "`+strconv.FormatFloat(priceWavesUsdt, 'f', 6, 64)+`"}`))
	httpmock.RegisterResponder("GET", "https://api.binance.com/api/v3/ticker/price?symbol=BTCUSDT",
		httpmock.NewStringResponder(200, `{"price": "`+strconv.FormatFloat(priceBtcUsdt, 'f', 6, 64)+`"}`))
	httpmock.RegisterResponder("GET", "https://api.binance.com/api/v3/ticker/price?symbol=WAVESBTC",
		httpmock.NewStringResponder(200, `{"price": "`+strconv.FormatFloat(priceWavesBtc, 'f', 6, 64)+`"}`))

	expectedPrice := (priceWavesUsdt + (priceWavesBtc * priceBtcUsdt)) / 2
	provider := provider.BinanceProvider{}
	price, _ := provider.PriceNow()

	assert.Equal(t, expectedPrice, price)
}
