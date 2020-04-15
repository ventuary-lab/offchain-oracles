package provider

type PriceProvider interface {
	PriceNow() (float64, error)
	PriceNowByPair(pair string) (float64, error)
}
