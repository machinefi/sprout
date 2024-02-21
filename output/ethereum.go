package output

import (
	"context"
	"encoding/hex"
	"log/slog"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	"github.com/txaty/go-merkletree"
	"golang.org/x/crypto/sha3"

	"github.com/machinefi/sprout/types"
)

type ethereumContract struct {
	chainEndpoint   string
	contractAddress string
	secretKey       string
	contractABI     abi.ABI
	contractMethod  string
}

func (e *ethereumContract) Output(task *types.Task, proof []byte) (string, error) {
	slog.Debug("outputing to ethereum contract", "chain endpoint", e.chainEndpoint)

	m := task.Messages[0]

	method, ok := e.contractABI.Methods[e.contractMethod]
	if !ok {
		return "", errors.Errorf("contract abi miss the contract method %s", e.contractMethod)
	}
	params := []interface{}{}
	for _, a := range method.Inputs {
		if a.Name == "proof" {
			p, err := hex.DecodeString(string(proof))
			if err != nil {
				return "", errors.Wrap(err, "proof decode failed")
			}
			params = append(params, p)
			continue
		}
		if a.Name == "merkle_root" {
			bs := make([]merkletree.DataBlock, 0, len(task.Messages))
			for _, m := range task.Messages {
				bs = append(bs, m.GetData())
			}

			tree, err := merkletree.New(&merkletree.Config{
				Mode:             merkletree.ModeTreeBuild,
				SortSiblingPairs: true,
				HashFunc: func(b []byte) ([]byte, error) {
					sha := sha3.NewLegacyKeccak256()
					sha.Write(b)
					return sha.Sum(nil), nil
				},
			}, bs)
			if err != nil {
				return "", errors.Wrap(err, "build merkletree failed")
			}
			params = append(params, tree.Root)
			continue
		}
		value := gjson.Get(m.Data, a.Name)
		param := value.String()
		if param == "" {
			return "", errors.Errorf("miss param %s for contract abi", a.Name)
		}

		switch a.Type.String() {
		case "address":
			params = append(params, common.HexToAddress(param))
		case "uint256":
			i := new(big.Int)
			i.SetString(strings.TrimPrefix(param, "0x"), 16)
			params = append(params, i)
		default:
			params = append(params, param)
		}
	}
	data, err := e.contractABI.Pack(e.contractMethod, params...)
	if err != nil {
		return "", errors.Wrap(err, "contract ABI pack failed")
	}

	txHash, err := e.sendTX(context.Background(), e.chainEndpoint, e.secretKey, e.contractAddress, data)
	if err != nil {
		return "", errors.Wrap(err, "transaction failed")
	}

	return txHash, nil
}

func (e *ethereumContract) sendTX(ctx context.Context, endpoint, privateKey, toStr string, data []byte) (string, error) {
	cli, err := ethclient.Dial(endpoint)
	if err != nil {
		return "", errors.Wrapf(err, "dial eth endpoint %s failed", endpoint)
	}

	pk := crypto.ToECDSAUnsafe(common.FromHex(privateKey))
	sender := crypto.PubkeyToAddress(pk.PublicKey)
	to := common.HexToAddress(toStr)

	gasPrice, err := cli.SuggestGasPrice(ctx)
	if err != nil {
		return "", errors.Wrap(err, "get suggest gas price failed")
	}

	chainid, err := cli.ChainID(ctx)
	if err != nil {
		return "", errors.Wrap(err, "get chain id failed")
	}

	nonce, err := cli.PendingNonceAt(ctx, sender)
	if err != nil {
		return "", errors.Wrap(err, "get pending nonce failed")
	}

	msg := ethereum.CallMsg{
		From:     sender,
		To:       &to,
		GasPrice: gasPrice,
		Data:     data,
	}
	gasLimit, err := cli.EstimateGas(ctx, msg)
	if err != nil {
		return "", errors.Wrap(err, "estimate gas failed")
	}

	tx := ethtypes.NewTx(
		&ethtypes.LegacyTx{
			Nonce:    nonce,
			GasPrice: gasPrice,
			Gas:      gasLimit,
			To:       &to,
			Data:     data,
		})

	signedTx, err := ethtypes.SignTx(tx, ethtypes.NewLondonSigner(chainid), pk)
	if err != nil {
		return "", errors.Wrap(err, "sign tx failed")
	}

	if err = cli.SendTransaction(ctx, signedTx); err != nil {
		return "", errors.Wrap(err, "send transaction failed")
	}

	return signedTx.Hash().Hex(), nil
}

func NewEthereum(chainEndpoint, secretKey, contractAddress, contractAbiJSON, contractMethod string) (Output, error) {
	contractABI, err := abi.JSON(strings.NewReader(contractAbiJSON))
	if err != nil {
		return nil, err
	}
	if len(secretKey) == 0 {
		return nil, errors.New("secretkey is empty")
	}
	return &ethereumContract{
		chainEndpoint:   chainEndpoint,
		secretKey:       secretKey,
		contractAddress: contractAddress,
		contractABI:     contractABI,
		contractMethod:  contractMethod,
	}, nil
}
