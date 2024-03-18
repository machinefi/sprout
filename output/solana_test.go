package output

import (
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	"github.com/blocto/solana-go-sdk/client"
	soltypes "github.com/blocto/solana-go-sdk/types"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/machinefi/sprout/types"
)

func TestNewSolanaProgram(t *testing.T) {
	require := require.New(t)

	t.Run("NewSolana", func(t *testing.T) {
		_, err := NewSolanaProgram("", "", "secretKey", "")
		require.NoError(err)
	})

	t.Run("SecretKeyNil", func(t *testing.T) {
		_, err := NewSolanaProgram("", "", "", "")
		require.EqualError(err, "secretkey is empty")
	})
}

func TestSolanaProgram_Output(t *testing.T) {
	require := require.New(t)
	patches := NewPatches()
	defer patches.Reset()

	contract := &SolanaProgram{}

	t.Run("SendTXFailed", func(t *testing.T) {
		patches = solanaProgramPackInstructions(patches, nil)
		patches = solanaProgramSendTX(patches, "", errors.New(t.Name()))
		_, err := contract.Output(&types.Task{}, []byte("proof"))
		require.EqualError(err, t.Name())
	})

	t.Run("SendTXSuccess", func(t *testing.T) {
		patches = solanaProgramSendTX(patches, "hash", nil)
		txHash, err := contract.Output(&types.Task{}, []byte("proof"))
		require.NoError(err)
		require.Equal("hash", txHash)
	})
}

func TestSolanaProgram_SendTX(t *testing.T) {
	require := require.New(t)
	patches := NewPatches()
	defer patches.Reset()

	secretKey := "fd6ac80f1b9886a6d157cd8e71f842a63c52ebd237cf48fba03ae587e197d511f0b2439ae6da236d26f17f56c68f05d48513cd99b33143fa0b1aec7838ce4276"
	contract := &SolanaProgram{}
	ins := contract.packInstructions([]byte("proof"))

	t.Run("MissingInstructionData", func(t *testing.T) {
		patches = patches.ApplyFuncReturn(client.NewClient, &client.Client{})
		_, err := contract.sendTX("", secretKey, nil)
		require.EqualError(err, "missing instruction data")
	})

	t.Run("GetSolanaBlockFailed", func(t *testing.T) {
		patches = patches.ApplyMethodReturn(&client.Client{}, "GetLatestBlockhash", nil, errors.New(t.Name()))
		_, err := contract.sendTX("", secretKey, ins)
		require.ErrorContains(err, t.Name())
	})
	patches = patches.ApplyMethodReturn(&client.Client{}, "GetLatestBlockhash", nil, nil)

	t.Run("BuildSolanaTxFailed", func(t *testing.T) {
		patches = patches.ApplyFuncReturn(soltypes.NewTransaction, nil, errors.New(t.Name()))
		_, err := contract.sendTX("", secretKey, ins)
		require.ErrorContains(err, t.Name())
	})
	patches = patches.ApplyFuncReturn(soltypes.NewTransaction, nil, nil)

	t.Run("SendSolanaTxFailed", func(t *testing.T) {
		patches = patches.ApplyMethodReturn(&client.Client{}, "SendTransaction", "", errors.New(t.Name()))
		_, err := contract.sendTX("", secretKey, ins)
		require.ErrorContains(err, t.Name())
	})

	t.Run("SendSolanaTxSuccess", func(t *testing.T) {
		patches = patches.ApplyMethodReturn(&client.Client{}, "SendTransaction", t.Name(), nil)

		hash, err := contract.sendTX("", secretKey, ins)
		require.NoError(err)
		require.Equal(t.Name(), hash)
	})
}

func solanaProgramPackInstructions(p *Patches, instructions []soltypes.Instruction) *Patches {
	return p.ApplyPrivateMethod(
		&SolanaProgram{},
		"packInstructions",
		func(proof []byte) []soltypes.Instruction {
			return instructions
		},
	)
}

func solanaProgramSendTX(p *Patches, hash string, err error) *Patches {
	return p.ApplyPrivateMethod(
		&SolanaProgram{},
		"sendTX",
		func(endpoint, privateKey string, ins []soltypes.Instruction) (string, error) {
			return hash, err
		},
	)
}
