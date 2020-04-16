package models

type SignedText struct {
	Message   string `json:"message"`
	PublicKey string `json:"publicKey"`
	Signature string `json:"signature"`
}
