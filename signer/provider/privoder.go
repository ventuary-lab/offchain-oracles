package provider

type PriceProvider interface {
	PriceNow() (float64, error)
}
