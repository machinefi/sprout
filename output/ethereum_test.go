package output

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"math/big"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/machinefi/sprout/types"
)

func TestNewEthereum(t *testing.T) {
	r := require.New(t)

	t.Run("AbiNil", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()

		p = p.ApplyFuncReturn(abi.JSON, nil, errors.New(t.Name()))
		_, err := NewEthereum("", "", "", "", "", "")
		r.ErrorContains(err, t.Name())
	})

	t.Run("SecretKeyNil", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()

		p = p.ApplyFuncReturn(abi.JSON, nil, nil)
		_, err := NewEthereum("", "", "", "", "", "")
		r.EqualError(err, "secretkey is empty")
	})

	t.Run("NewEthereumSuccess", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()

		p = p.ApplyFuncReturn(abi.JSON, nil, nil)
		_, err := NewEthereum("", "secretKey", "", "", "", "")
		r.NoError(err)
	})
}

func TestEthereumContract_Output(t *testing.T) {
	r := require.New(t)

	chainEndpoint := "https://iotex"
	secretKey := "b7255a24"
	contractAddress := "0x5Ea91218CB1E329806a746E0816A8BD533637b42"
	receiverAddress := "0x5Ea91218CB1E329806a746E0816A8BD533637b42"
	contractAbiJSON := `[{"inputs":[],"name":"getJournal","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"getPostStateDigest","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"getProjectId","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"getReceiver","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"getSeal","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"bytes","name":"_proof","type":"bytes"}],"name":"setProof","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"uint256","name":"_projectId","type":"uint256"},{"internalType":"address","name":"_receiver","type":"address"},{"internalType":"bytes","name":"_data_snark","type":"bytes"}],"name":"submit","outputs":[],"stateMutability":"nonpayable","type":"function"}]`
	contractMethod := "submit"

	proof := "{\"Snark\":{\"snark\":{\"a\":[[11,176,218,102,82,247],[19,201,71,203,]],\"b\":[[[37,238,237,46],[36,124,137]],[[5,237,77],[41,187,159]]],\"c\":[[31,108,130],[34,189,130]],\"public\":[[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,68],[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,197],[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,5]]},\"journal\":[82,0,0,0,73,32]}}"
	hexProof := hex.EncodeToString([]byte(proof))

	task := &types.Task{
		ID:             1,
		ProjectID:      uint64(0x1),
		ProjectVersion: "0.1",
		Data:           [][]byte{[]byte("data")},
	}

	t.Run("MissMethod", func(t *testing.T) {
		contractMissMethod := "setProof1"
		contract, err := NewEthereum(chainEndpoint, secretKey, contractAddress, "", contractAbiJSON, contractMissMethod)
		r.NoError(err)

		_, err = contract.Output(task, []byte("proof"))
		r.EqualError(err, "contract abi miss the contract method setProof1")
	})

	t.Run("MissReceiverAddress", func(t *testing.T) {
		contract, err := NewEthereum(chainEndpoint, secretKey, contractAddress, "", contractAbiJSON, contractMethod)
		r.NoError(err)

		_, err = contract.Output(task, []byte("this is proof"))
		r.ErrorContains(err, "miss param")
	})

	t.Run("GetPostStateDigestFailed", func(t *testing.T) {
		contract, err := NewEthereum(chainEndpoint, secretKey, contractAddress, receiverAddress, contractAbiJSON, contractMethod)
		r.NoError(err)

		_, err = contract.Output(task, []byte(hexProof))
		r.ErrorContains(err, "get Snark.post_state_digest failed")
	})

	t.Run("MissProofSnarkParam", func(t *testing.T) {
		contractSnarkAbiJSON := `[{"inputs":[],"name":"getJournal","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"getPostStateDigest","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"getProjectId","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"getReceiver","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"getSeal","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"bytes","name":"_proof","type":"bytes"}],"name":"setProof","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"uint256","name":"_projectId","type":"uint256"},{"internalType":"address","name":"_receiver","type":"address"},{"internalType":"bytes","name":"proof_snark_seal","type":"bytes"},{"internalType":"bytes","name":"proof_snark_post_state_digest","type":"bytes"},{"internalType":"bytes","name":"proof_snark_journal","type":"bytes"}],"name":"submit","outputs":[],"stateMutability":"nonpayable","type":"function"}]`
		contract, err := NewEthereum(chainEndpoint, secretKey, contractAddress, receiverAddress, contractSnarkAbiJSON, contractMethod)
		r.NoError(err)

		_, err = contract.Output(task, []byte("this is proof"))
		r.ErrorContains(err, "miss param")
	})

	t.Run("MissParam", func(t *testing.T) {
		contractMissParamAbiJSON := `[{"inputs":[{"internalType":"address","name":"depinRC20Address","type":"address"},{"internalType":"uint256","name":"nonce","type":"uint256"},{"internalType":"address","name":"sender","type":"address"},{"internalType":"bytes","name":"proof","type":"bytes"}],"name":"mine","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"depinRC20","outputs":[{"internalType":"contract IDepinRC20","name":"","type":"address"}],"stateMutability":"view","type":"function"}]`
		contractMissParamMethod := "mine"
		contract, err := NewEthereum(chainEndpoint, secretKey, contractAddress, receiverAddress, contractMissParamAbiJSON, contractMissParamMethod)
		r.NoError(err)

		_, err = contract.Output(task, []byte("this is proof"))
		r.ErrorContains(err, "miss param")
	})

	t.Run("TransactionFailed", func(t *testing.T) {
		contractAbiJSON = `[{"constant":false,"inputs":[{"internalType":"bytes","name":"proof","type":"bytes"}],"name":"setProof","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"getProof","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"payable":false,"stateMutability":"view","type":"function"}]`
		contractMethod = "setProof"
		contract, err := NewEthereum(chainEndpoint, secretKey, contractAddress, receiverAddress, contractAbiJSON, contractMethod)
		r.NoError(err)

		_, err = contract.Output(task, []byte("this is proof"))
		r.ErrorContains(err, "transaction failed")
	})
}

func TestEthereumContract_SendTX(t *testing.T) {
	r := require.New(t)

	contract := &EthereumContract{}
	ctx := context.Background()

	t.Run("DialEthFailed", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()

		p = p.ApplyFuncReturn(ethclient.Dial, nil, errors.New(t.Name()))

		_, err := contract.sendTX(ctx, "", "", "", nil)
		r.ErrorContains(err, t.Name())
	})

	t.Run("SuggestGasFailed", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()

		p = p.ApplyFuncReturn(ethclient.Dial, nil, nil)
		p = p.ApplyFuncReturn(crypto.ToECDSAUnsafe, &ecdsa.PrivateKey{})
		p = p.ApplyFuncReturn(crypto.PubkeyToAddress, common.Address{})
		p = p.ApplyFuncReturn(common.HexToAddress, common.Address{})
		p = p.ApplyMethodReturn(&ethclient.Client{}, "SuggestGasPrice", nil, errors.New(t.Name()))

		_, err := contract.sendTX(ctx, "", "", "", nil)
		r.ErrorContains(err, t.Name())
	})

	t.Run("GetChainIdFailed", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()

		p = p.ApplyFuncReturn(ethclient.Dial, nil, nil)
		p = p.ApplyFuncReturn(crypto.ToECDSAUnsafe, &ecdsa.PrivateKey{})
		p = p.ApplyFuncReturn(crypto.PubkeyToAddress, common.Address{})
		p = p.ApplyFuncReturn(common.HexToAddress, common.Address{})
		p = p.ApplyMethodReturn(&ethclient.Client{}, "SuggestGasPrice", big.NewInt(1), nil)
		p = p.ApplyMethodReturn(&ethclient.Client{}, "ChainID", nil, errors.New(t.Name()))
		_, err := contract.sendTX(ctx, "", "", "", nil)
		r.ErrorContains(err, t.Name())
	})

	t.Run("GetNonceFailed", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()

		p = p.ApplyFuncReturn(ethclient.Dial, nil, nil)
		p = p.ApplyFuncReturn(crypto.ToECDSAUnsafe, &ecdsa.PrivateKey{})
		p = p.ApplyFuncReturn(crypto.PubkeyToAddress, common.Address{})
		p = p.ApplyFuncReturn(common.HexToAddress, common.Address{})
		p = p.ApplyMethodReturn(&ethclient.Client{}, "SuggestGasPrice", big.NewInt(1), nil)
		p = p.ApplyMethodReturn(&ethclient.Client{}, "ChainID", big.NewInt(1), nil)
		p = p.ApplyMethodReturn(&ethclient.Client{}, "PendingNonceAt", nil, errors.New(t.Name()))
		_, err := contract.sendTX(ctx, "", "", "", nil)
		r.ErrorContains(err, t.Name())
	})

	t.Run("EstimateGasFailed", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()

		p = p.ApplyFuncReturn(ethclient.Dial, nil, nil)
		p = p.ApplyFuncReturn(crypto.ToECDSAUnsafe, &ecdsa.PrivateKey{})
		p = p.ApplyFuncReturn(crypto.PubkeyToAddress, common.Address{})
		p = p.ApplyFuncReturn(common.HexToAddress, common.Address{})
		p = p.ApplyMethodReturn(&ethclient.Client{}, "SuggestGasPrice", big.NewInt(1), nil)
		p = p.ApplyMethodReturn(&ethclient.Client{}, "ChainID", big.NewInt(1), nil)
		p = p.ApplyMethodReturn(&ethclient.Client{}, "PendingNonceAt", uint64(1), nil)
		p = p.ApplyMethodReturn(&ethclient.Client{}, "EstimateGas", nil, errors.New(t.Name()))
		_, err := contract.sendTX(ctx, "", "", "", nil)
		r.ErrorContains(err, t.Name())
	})

	t.Run("SignTxFailed", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()

		p = p.ApplyFuncReturn(ethclient.Dial, nil, nil)
		p = p.ApplyFuncReturn(crypto.ToECDSAUnsafe, &ecdsa.PrivateKey{})
		p = p.ApplyFuncReturn(crypto.PubkeyToAddress, common.Address{})
		p = p.ApplyFuncReturn(common.HexToAddress, common.Address{})
		p = p.ApplyMethodReturn(&ethclient.Client{}, "SuggestGasPrice", big.NewInt(1), nil)
		p = p.ApplyMethodReturn(&ethclient.Client{}, "ChainID", big.NewInt(1), nil)
		p = p.ApplyMethodReturn(&ethclient.Client{}, "PendingNonceAt", uint64(1), nil)
		p = p.ApplyMethodReturn(&ethclient.Client{}, "EstimateGas", uint64(1), nil)
		p = p.ApplyFuncReturn(ethtypes.SignTx, nil, errors.New(t.Name()))
		_, err := contract.sendTX(ctx, "", "", "", nil)
		r.ErrorContains(err, t.Name())
	})

	t.Run("TransactionFailed", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()

		p = p.ApplyFuncReturn(ethclient.Dial, nil, nil)
		p = p.ApplyFuncReturn(crypto.ToECDSAUnsafe, &ecdsa.PrivateKey{})
		p = p.ApplyFuncReturn(crypto.PubkeyToAddress, common.Address{})
		p = p.ApplyFuncReturn(common.HexToAddress, common.Address{})
		p = p.ApplyMethodReturn(&ethclient.Client{}, "SuggestGasPrice", big.NewInt(1), nil)
		p = p.ApplyMethodReturn(&ethclient.Client{}, "ChainID", big.NewInt(1), nil)
		p = p.ApplyMethodReturn(&ethclient.Client{}, "PendingNonceAt", uint64(1), nil)
		p = p.ApplyMethodReturn(&ethclient.Client{}, "EstimateGas", uint64(1), nil)
		p = p.ApplyFuncReturn(ethtypes.SignTx, nil, nil)
		p = p.ApplyMethodReturn(&ethclient.Client{}, "SendTransaction", errors.New(t.Name()))
		_, err := contract.sendTX(ctx, "", "", "", nil)
		r.ErrorContains(err, t.Name())
	})

	t.Run("TransactionSuccess", func(t *testing.T) {
		p := NewPatches()
		defer p.Reset()

		p = p.ApplyFuncReturn(ethclient.Dial, nil, nil)
		p = p.ApplyFuncReturn(crypto.ToECDSAUnsafe, &ecdsa.PrivateKey{})
		p = p.ApplyFuncReturn(crypto.PubkeyToAddress, common.Address{})
		p = p.ApplyFuncReturn(common.HexToAddress, common.Address{})
		p = p.ApplyMethodReturn(&ethclient.Client{}, "SuggestGasPrice", big.NewInt(1), nil)
		p = p.ApplyMethodReturn(&ethclient.Client{}, "ChainID", big.NewInt(1), nil)
		p = p.ApplyMethodReturn(&ethclient.Client{}, "PendingNonceAt", uint64(1), nil)
		p = p.ApplyMethodReturn(&ethclient.Client{}, "EstimateGas", uint64(1), nil)
		p = p.ApplyFuncReturn(ethtypes.SignTx, nil, nil)
		p = p.ApplyMethodReturn(&ethclient.Client{}, "SendTransaction", nil)
		p = p.ApplyMethodReturn(&ethtypes.Transaction{}, "Hash", common.Hash{})
		tx, err := contract.sendTX(ctx, "", "", "", nil)
		r.NoError(err)
		r.Equal(tx, "0x0000000000000000000000000000000000000000000000000000000000000000")
	})
}
