package main

import (
	"crypto/ecdsa"
	"fmt"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	solanaTypes "github.com/blocto/solana-go-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/machinefi/sprout/cmd/enode/api"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestGetENodeConfig(t *testing.T) {
	require := require.New(t)
	patches := NewPatches()
	defer patches.Reset()

	t.Run("JustContract", func(t *testing.T) {
		patches = viperGetStringSeq(patches, []OutputCell{
			{Values: Params{"contractAddress"}},
			{Values: Params{nil}},
			{Values: Params{nil}},
		})
		config, err := getENodeConfig()
		fmt.Println(config)
		require.NoError(err)
		require.Equal(&api.ENodeConfigResp{ProjectContractAddress: "contractAddress"}, config)
	})

	t.Run("IncludeEthPK", func(t *testing.T) {
		patches = viperGetStringSeq(patches, []OutputCell{
			{Values: Params{"contractAddress"}},
			{Values: Params{"0x0000000000000000000000000000000000000000"}},
			{Values: Params{nil}},
			{Values: Params{nil}},
		})
		patches = patches.ApplyFuncReturn(crypto.ToECDSAUnsafe, &ecdsa.PrivateKey{PublicKey: ecdsa.PublicKey{}})
		patches = patches.ApplyFuncReturn(crypto.PubkeyToAddress, common.Address{})
		config, err := getENodeConfig()
		fmt.Println(config)
		require.NoError(err)
		require.Equal(&api.ENodeConfigResp{
			ProjectContractAddress: "contractAddress",
			OperatorETHAddress:     "0x0000000000000000000000000000000000000000",
		}, config)
	})

	t.Run("IncludeSolanaFail", func(t *testing.T) {
		patches = viperGetStringSeq(patches, []OutputCell{
			{Values: Params{"contractAddress"}},
			{Values: Params{nil}},
			{Values: Params{"11111111111111111111111111111111"}},
			{Values: Params{nil}},
		})
		patches = patches.ApplyFuncReturn(solanaTypes.AccountFromHex, nil, errors.New(t.Name()))
		_, err := getENodeConfig()
		require.ErrorContains(err, t.Name())
	})

	t.Run("IncludeSolana", func(t *testing.T) {
		patches = viperGetStringSeq(patches, []OutputCell{
			{Values: Params{"contractAddress"}},
			{Values: Params{nil}},
			{Values: Params{"11111111111111111111111111111111"}},
			{Values: Params{nil}},
		})
		patches = patches.ApplyFuncReturn(solanaTypes.AccountFromHex, solanaTypes.Account{}, nil)
		config, err := getENodeConfig()
		fmt.Println(config)
		require.NoError(err)
		require.Equal(&api.ENodeConfigResp{
			ProjectContractAddress: "contractAddress",
			OperatorSolanaAddress:  "11111111111111111111111111111111",
		}, config)
	})
}

func viperGetStringSeq(p *Patches, outputs []OutputCell) *Patches {
	return p.ApplyFuncSeq(
		viper.GetString,
		outputs,
	)
}
