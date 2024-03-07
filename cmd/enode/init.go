package main

import (
	"log/slog"
	"os"

	solanaTypes "github.com/blocto/solana-go-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/machinefi/sprout/cmd/enode/api"
)

func initLogger() {
	var programLevel = slog.LevelDebug
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: programLevel})
	slog.SetDefault(slog.New(h))
}

func initConfig() {
	viper.SetDefault(ChainEndpoint, "https://babel-api.testnet.iotex.io")
	viper.SetDefault(HttpServiceEndpoint, ":9000")
	viper.SetDefault(DatabaseDSN, "postgres://test_user:test_passwd@postgres:5432/test?sslmode=disable")
	viper.SetDefault(BootNodeMultiaddr, "/dns4/bootnode-0.testnet.iotex.one/tcp/4689/ipfs/12D3KooWFnaTYuLo8Mkbm3wzaWHtUuaxBRe24Uiopu15Wr5EhD3o")
	viper.SetDefault(IotexChainID, 2)
	viper.SetDefault(ProjectContractAddress, "0x02feBE78F3A740b3e9a1CaFAA1b23a2ac0793D26")
	viper.SetDefault(IPFSEndpoint, "ipfs.mainnet.iotex.io")
	viper.SetDefault(DIDAuthServerEndpoint, "didkit:9999")

	viper.MustBindEnv(ChainEndpoint)
	viper.MustBindEnv(HttpServiceEndpoint)
	viper.MustBindEnv(DatabaseDSN)
	viper.MustBindEnv(BootNodeMultiaddr)
	viper.MustBindEnv(IotexChainID)
	viper.MustBindEnv(ProjectContractAddress)
	viper.MustBindEnv(IPFSEndpoint)
	viper.MustBindEnv(DIDAuthServerEndpoint)

	viper.BindEnv(OperatorPrivateKey)
	viper.BindEnv(OperatorPrivateKeyED25519)
	viper.BindEnv(ProjectFileDirectory)
}

func getENodeConfig() (*api.ENodeConfigResp, error) {
	enodeConf := &api.ENodeConfigResp{ProjectContractAddress: viper.GetString(ProjectContractAddress)}

	if len(viper.GetString(OperatorPrivateKey)) > 0 {
		pk := crypto.ToECDSAUnsafe(common.FromHex(viper.GetString(OperatorPrivateKey)))
		sender := crypto.PubkeyToAddress(pk.PublicKey)
		enodeConf.OperatorETHAddress = sender.String()
	}

	if len(viper.GetString(OperatorPrivateKeyED25519)) > 0 {
		wallet, err := solanaTypes.AccountFromHex(viper.GetString(OperatorPrivateKeyED25519))
		if err != nil {
			return nil, errors.Wrap(err, "get solana wallet failed")
		}
		enodeConf.OperatorSolanaAddress = wallet.PublicKey.String()
	}

	return enodeConf, nil
}
