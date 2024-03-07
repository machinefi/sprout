package config_test

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/machinefi/sprout/cmd/znode/config"
)

func TestConfig_Init(t *testing.T) {
	r := require.New(t)

	t.Run("UseEnvConfig", func(t *testing.T) {
		os.Clearenv()
		expected := config.Config{
			Risc0ServerEndpoint:    "risc0:1111",
			Halo2ServerEndpoint:    "halo2:2222",
			ZKWasmServerEndpoint:   "zkwasm:3333",
			ChainEndpoint:          "http://abc.def.com",
			ProjectContractAddress: "0x123",
			DatabaseDSN:            "postgres://root@localhost/abc?ext=666",
			BootNodeMultiAddr:      "/dsn4/abc/123",
			ZnodeContractAddress:   "0x456",
			IoTeXChainID:           5,
			IPFSEndpoint:           "abc.ipfs.net",
			IoID:                   "did:key:pub",
			ProjectFileDirectory:   "/path/to/project/configs",
		}

		_ = os.Setenv("RISC0_SERVER_ENDPOINT", expected.Risc0ServerEndpoint)
		_ = os.Setenv("HALO2_SERVER_ENDPOINT", expected.Halo2ServerEndpoint)
		_ = os.Setenv("ZKWASM_SERVER_ENDPOINT", expected.ZKWasmServerEndpoint)
		_ = os.Setenv("CHAIN_ENDPOINT", expected.ChainEndpoint)
		_ = os.Setenv("DATABASE_DSN", expected.DatabaseDSN)
		_ = os.Setenv("BOOTNODE_MULTIADDR", expected.BootNodeMultiAddr)
		_ = os.Setenv("ZNODE_CONTRACT_ADDRESS", expected.ZnodeContractAddress)
		_ = os.Setenv("IOTEX_CHAINID", strconv.Itoa(expected.IoTeXChainID))
		_ = os.Setenv("PROJECT_CONTRACT_ADDRESS", expected.ProjectContractAddress)
		_ = os.Setenv("IPFS_ENDPOINT", expected.IPFSEndpoint)
		_ = os.Setenv("IO_ID", expected.IoID)
		_ = os.Setenv("PROJECT_FILE_DIRECTORY", expected.ProjectFileDirectory)

		c := &config.Config{}
		r.Nil(c.Init())
		r.Equal(*c, expected)
	})

	t.Run("CatchPanicCausedByEmptyRequiredEnvVar", func(t *testing.T) {
		os.Clearenv()

		c := &config.Config{}
		defer func() {
			r.NotNil(recover())
		}()
		_ = c.Init()
	})
}
