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
)

var (
	errSnarkProofDataMissingFieldSnark           = errors.New("missing field `Snark.snark`")
	errSnarkProofDataMissingFieldPostStateDigest = errors.New("missing field `Snark.post_state_digest`")
	errSnarkProofDataMissingFieldJournal         = errors.New("missing field `Snark.journal`")
	errMissingReceiverParam                      = errors.New("missing receiver param")
)

type ethereumContract struct {
	chainEndpoint   string
	contractAddress string
	receiverAddress string
	secretKey       string
	contractABI     abi.ABI
	contractMethod  abi.Method
}

func (e *ethereumContract) Output(projectID uint64, taskData [][]byte, proof []byte) (string, error) {
	slog.Debug("outputing to ethereum contract", "chain endpoint", e.chainEndpoint)

	params := []interface{}{}
	for _, a := range e.contractMethod.Inputs {
		switch a.Name {
		case "proof", "_proof":
			params = append(params, proof)

		case "projectId", "_projectId":
			i := new(big.Int).SetUint64(projectID)
			params = append(params, i)

		case "receiver", "_receiver":
			if e.receiverAddress == "" {
				return "", errMissingReceiverParam
			}
			params = append(params, common.HexToAddress(e.receiverAddress))

		case "data_snark", "_data_snark":
			proof, err := hex.DecodeString(string(proof))
			if err != nil {
				return "", errors.Wrap(err, "failed to decode proof by hex format")
			}
			valueSeal := gjson.GetBytes(proof, "Snark.snark").String()
			if valueSeal == "" {
				return "", errSnarkProofDataMissingFieldSnark
			}
			valueDigest := gjson.GetBytes(proof, "Snark.post_state_digest").String()
			if valueDigest == "" {
				return "", errSnarkProofDataMissingFieldPostStateDigest
			}
			valueJournal := gjson.GetBytes(proof, "Snark.journal").String()
			if valueJournal == "" {
				return "", errSnarkProofDataMissingFieldJournal
			}

			abiBytes, err := abi.NewType("bytes", "", nil)
			if err != nil {
				return "", errors.Wrap(err, "new ethereum accounts abi pack failed")
			}
			args := abi.Arguments{
				{Type: abiBytes, Name: "proof_snark_seal"},
				{Type: abiBytes, Name: "proof_snark_post_state_digest"},
				{Type: abiBytes, Name: "proof_snark_journal"},
			}

			packed, err := args.Pack([]byte(valueSeal), []byte(valueDigest), []byte(valueJournal))
			if err != nil {
				return "", errors.Wrap(err, "ethereum accounts abi pack failed")
			}
			params = append(params, packed)

		default:
			value := gjson.GetBytes(taskData[0], a.Name)
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
	}
	calldata, err := e.contractABI.Pack(e.contractMethod.Name, params...)
	if err != nil {
		return "", errors.Wrap(err, "failed to pack by contract abi")
	}

	txHash, err := e.sendTX(context.Background(), calldata)
	if err != nil {
		return "", errors.Wrap(err, "failed to send transaction")
	}

	return txHash, nil
}

func (e *ethereumContract) sendTX(ctx context.Context, data []byte) (string, error) {
	cli, err := ethclient.Dial(e.chainEndpoint)
	if err != nil {
		return "", errors.Wrapf(err, "dial eth endpoint %s failed", e.chainEndpoint)
	}

	pk := crypto.ToECDSAUnsafe(common.FromHex(e.secretKey))
	sender := crypto.PubkeyToAddress(pk.PublicKey)
	to := common.HexToAddress(e.contractAddress)

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

func newEthereum(chainEndpoint, secretKey, contractAddress, receiverAddress, contractAbiJSON, contractMethod string) (Output, error) {
	if secretKey == "" {
		return nil, errors.New("secret key is empty")
	}
	contractABI, err := abi.JSON(strings.NewReader(contractAbiJSON))
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode contract abi")
	}
	method, ok := contractABI.Methods[contractMethod]
	if !ok {
		return nil, errors.New("the contract method not exist in abi")
	}
	return &ethereumContract{
		chainEndpoint:   chainEndpoint,
		secretKey:       secretKey,
		contractAddress: contractAddress,
		receiverAddress: receiverAddress,
		contractABI:     contractABI,
		contractMethod:  method,
	}, nil
}
