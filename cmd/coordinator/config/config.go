package config

import (
	"log/slog"
	"os"

	"github.com/machinefi/sprout/cmd/internal"
)

type Config struct {
	ServiceEndpoint           string `env:"HTTP_SERVICE_ENDPOINT"`
	ChainEndpoint             string `env:"CHAIN_ENDPOINT"`
	DatabaseDSN               string `env:"DATABASE_DSN"`
	DatasourceDSN             string `env:"DATASOURCE_DSN"`
	BootNodeMultiAddr         string `env:"BOOTNODE_MULTIADDR"`
	IoTeXChainID              int    `env:"IOTEX_CHAINID"`
	ProjectContractAddress    string `env:"PROJECT_CONTRACT_ADDRESS,optional"`
	IPFSEndpoint              string `env:"IPFS_ENDPOINT"`
	DIDAuthServerEndpoint     string `env:"DIDAUTH_SERVER_ENDPOINT"`
	OperatorPrivateKey        string `env:"OPERATOR_PRIVATE_KEY,optional"`
	OperatorPrivateKeyED25519 string `env:"OPERATOR_PRIVATE_KEY_ED25519,optional"`
	ProjectFileDirectory      string `env:"PROJECT_FILE_DIRECTORY,optional"`
	ProjectCacheDirectory     string `env:"PROJECT_CACHE_DIRECTORY,optional"`
	LogLevel                  int    `env:"LOG_LEVEL,optional"`
	env                       string `env:"-"`
}

var (
	// prod default config for coordinator; all config elements come from docker-compose.yaml in root of project
	defaultConfig = &Config{
		ServiceEndpoint:        ":9001",
		ChainEndpoint:          "https://babel-api.testnet.iotex.io",
		DatabaseDSN:            "postgres://test_user:test_passwd@postgres:5432/test?sslmode=disable",
		DatasourceDSN:          "postgres://test_user:test_passwd@postgres:5432/test?sslmode=disable",
		BootNodeMultiAddr:      "/dns4/bootnode-0.testnet.iotex.one/tcp/4689/ipfs/12D3KooWFnaTYuLo8Mkbm3wzaWHtUuaxBRe24Uiopu15Wr5EhD3o",
		IoTeXChainID:           2,
		ProjectContractAddress: "0x02feBE78F3A740b3e9a1CaFAA1b23a2ac0793D26",
		IPFSEndpoint:           "ipfs.mainnet.iotex.io",
		DIDAuthServerEndpoint:  "didkit:9999",
		LogLevel:               int(slog.LevelDebug),
	}
	// local debug default config for coordinator; all config elements come from docker-compose-dev.yaml in root of project
	defaultDebugConfig = &Config{
		ServiceEndpoint:        ":9001",
		ChainEndpoint:          "https://babel-api.testnet.iotex.io",
		DatabaseDSN:            "postgres://test_user:test_passwd@localhost:5432/test?sslmode=disable",
		DatasourceDSN:          "postgres://test_user:test_passwd@localhost:5432/test?sslmode=disable",
		BootNodeMultiAddr:      "/dns4/bootnode-0.testnet.iotex.one/tcp/4689/ipfs/12D3KooWFnaTYuLo8Mkbm3wzaWHtUuaxBRe24Uiopu15Wr5EhD3o",
		IoTeXChainID:           2,
		ProjectContractAddress: "0x02feBE78F3A740b3e9a1CaFAA1b23a2ac0793D26",
		IPFSEndpoint:           "ipfs.mainnet.iotex.io",
		DIDAuthServerEndpoint:  "localhost:9999",
		ProjectCacheDirectory:  "./project_cache",
		LogLevel:               int(slog.LevelDebug),
	}
	// integration default config for coordinator; all config elements come from Makefile in `integration_test` entry
	defaultTestConfig = &Config{
		ServiceEndpoint:        ":19001",
		ChainEndpoint:          "https://babel-api.testnet.iotex.io",
		DatabaseDSN:            "postgres://test_user:test_passwd@localhost:15432/test?sslmode=disable",
		DatasourceDSN:          "postgres://test_user:test_passwd@localhost:15432/test?sslmode=disable",
		BootNodeMultiAddr:      "/dns4/bootnode-0.testnet.iotex.one/tcp/4689/ipfs/12D3KooWFnaTYuLo8Mkbm3wzaWHtUuaxBRe24Uiopu15Wr5EhD3o",
		IoTeXChainID:           2,
		ProjectContractAddress: "", //"0x02feBE78F3A740b3e9a1CaFAA1b23a2ac0793D26",
		IPFSEndpoint:           "ipfs.mainnet.iotex.io",
		DIDAuthServerEndpoint:  "localhost:19999",
		ProjectFileDirectory:   "./testdata",
		LogLevel:               int(slog.LevelDebug),
	}
)

func (c *Config) Init() error {
	if err := internal.ParseEnv(c); err != nil {
		return err
	}
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.Level(c.LogLevel)})
	slog.SetDefault(slog.New(h))
	return nil
}

func (c *Config) Env() string {
	return c.env
}

func Get() (*Config, error) {
	var conf *Config
	env := os.Getenv("COORDINATOR_ENV")
	switch env {
	case "INTEGRATION_TEST":
		conf = defaultTestConfig
	case "LOCAL_DEBUG":
		conf = defaultDebugConfig
	default:
		env = "PROD"
		conf = defaultConfig
	}
	conf.env = env
	if err := conf.Init(); err != nil {
		return nil, err
	}
	return conf, nil
}

func (c *Config) Print() {
	internal.Print(c)
}
