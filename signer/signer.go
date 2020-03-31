package signer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"offchain-oracles/config"
	"offchain-oracles/helpers"
	"offchain-oracles/signer/provider"
	"offchain-oracles/storage"

	"offchain-oracles/models"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/syndtr/goleveldb/leveldb"
	wavesplatform "github.com/wavesplatform/go-lib-crypto"
	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	proto "github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	signPrefix = "WAVESNEUTRINOPREFIX"
)

func StartSigner(cfg config.Config, stringSeed string, chainId byte, ctx context.Context, db *leveldb.DB) error {
	priceProvider := provider.BinanceProvider{}
	nodeHelper := helpers.New(cfg.NodeURL, "")

	nodeClient, err := client.NewClient(client.Options{ApiKey: "", BaseUrl: cfg.NodeURL})
	if err != nil {
		return err
	}

	isTimeout := false
	for true {
		var err error
		isTimeout, err = HandleHeight(cfg, stringSeed, chainId, db, nodeClient, nodeHelper, priceProvider, ctx, isTimeout)
		if err != nil {
			fmt.Printf("Error: %s \n", err.Error())
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	return nil
}

func HandleHeight(cfg config.Config, stringSeed string, chainId byte, db *leveldb.DB, nodeClient *client.Client,
	nodeHelper helpers.Node, priceProvider provider.PriceProvider, ctx context.Context, isTimeout bool) (bool, error) {

	wCrypto := wavesplatform.NewWavesCrypto()
	seed := wavesplatform.Seed(stringSeed)
	secret, err := crypto.NewSecretKeyFromBase58(string(wCrypto.PrivateKey(seed)))
	if err != nil {
		return false, err
	}
	pubKey := crypto.GeneratePublicKey(secret)

	contractAddress, err := proto.NewAddressFromString(cfg.ControlContract)
	if err != nil {
		return false, err
	}

	contractState, err := nodeHelper.GetStateByAddress(cfg.ControlContract)
	if err != nil {
		return false, err
	}

	pubKeyOracles := strings.Split(contractState["oracles"].Value.(string), ",")

	height, _, err := nodeClient.Blocks.Height(ctx)
	if err != nil {
		return false, err
	}

	_, priceExist := contractState["price_"+strconv.Itoa(int(height.Height))]
	if priceExist {
		return false, nil
	}

	signs := make(map[string]string)
	values := make(map[string]string)

	for _, ip := range cfg.Ips {
		var client = &http.Client{Timeout: 10 * time.Second}
		res, err := client.Get(ip + "/api/price?height=" + strconv.Itoa(int(height.Height)))
		if err != nil {
			fmt.Printf("Http error %s: %s \n", ip, err.Error())
			continue
		}

		if res.StatusCode == 200 {
			var result models.SignedText
			err = json.NewDecoder(res.Body).Decode(&result)
			if err != nil {
				fmt.Printf("Parse error %s: %s \n", ip, err.Error())
				continue
			}
			if result.Message == "" {
				fmt.Printf("Oracle (%s) %s: %s \n", ip, result.PublicKey, "empty msg")
				continue
			}
			values[result.PublicKey] = strings.Split(result.Message, "_")[2]
			signs[result.PublicKey] = result.Signature
			fmt.Printf("Oracle (%s) %s: %s \n", ip, result.PublicKey, values[result.PublicKey])
		}
		if res.Body != nil {
			if err := res.Body.Close(); err != nil {
				fmt.Printf("Http close error %s: %s \n", ip, err.Error())
				continue
			}
		}
	}

	signedPrice, err := storage.GetKeystore(db, height.Height)
	if err != nil && err != leveldb.ErrNotFound {
		fmt.Printf("Error: %s \n", err.Error())
	} else {
		newNotConvertedPrice, err := priceProvider.PriceNow()
		if err != nil {
			return false, err
		}

		newPrice := int(newNotConvertedPrice * 100)
		msg := signPrefix + "_" + strconv.Itoa(int(height.Height)) + "_" + strconv.Itoa(newPrice)

		signature := wCrypto.SignBytesBySeed([]byte(msg), seed)

		signedText := models.SignedText{
			Message:   msg,
			PublicKey: string(wCrypto.PublicKey(seed)),
			Signature: base58.Encode(signature),
		}
		err = storage.PutKeystore(db, height.Height, signedText)
		if err != nil {
			return false, err
		}
	}
	fmt.Printf("Price by {%d}: {%s} \n", height, signedPrice.Message)

	if !isTimeout {
		time.Sleep(time.Duration(cfg.Timeout) * time.Second)
		return true, nil
	}

	if _, ok := contractState["price_"+strconv.Itoa(int(height.Height))]; len(signs) >= 3 && !ok {
		funcArgs := new(proto.Arguments)
		for _, pubKey := range pubKeyOracles {

			value, ok := values[pubKey]
			if ok {
				valueInt, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					funcArgs.Append(proto.NewIntegerArgument(0))
				} else {
					funcArgs.Append(proto.NewIntegerArgument(valueInt))
				}
			} else {
				funcArgs.Append(proto.NewIntegerArgument(0))
			}

			sign, ok := signs[pubKey]
			bytesSign := base58.Decode(sign)
			if ok {
				funcArgs.Append(proto.NewBinaryArgument(bytesSign))
			} else {
				funcArgs.Append(proto.NewBinaryArgument(nil))
			}
		}

		asset, err := proto.NewOptionalAssetFromString("WAVES")
		if err != nil {
			return false, err
		}

		tx := &proto.InvokeScriptV1{
			Type:            proto.InvokeScriptTransaction,
			Version:         1,
			SenderPK:        pubKey,
			ChainID:         chainId,
			ScriptRecipient: proto.NewRecipientFromAddress(contractAddress),
			FunctionCall: proto.FunctionCall{
				Name:      "finalizeCurrentPrice",
				Arguments: *funcArgs,
			},
			Payments:  nil,
			FeeAsset:  *asset,
			Fee:       500000,
			Timestamp: client.NewTimestampFromTime(time.Now()),
		}

		err = tx.Sign(secret)
		if err != nil {
			return false, err
		}

		_, err = nodeClient.Transactions.Broadcast(ctx, tx)
		if err != nil {
			return false, err
		}

		err = <-nodeHelper.WaitTx(base58.Encode((*tx.ID)[:]))
		if err != nil {
			return false, err
		}
		fmt.Printf("Tx finilize: %s \n", tx.ID)
	}

	time.Sleep(1 * time.Second)
	return false, nil
}
