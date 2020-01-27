package provider

type PriceProvider interface {
	PriceNow() (float32, error)
}
