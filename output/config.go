package output

type (
	// Config is the configuration for the outputter
	Config struct {
		Type Type

		// configuration for the EVM contract outputter
		ChainEndpoint   string
		ContractAddress string
		Sk              string
	}

	// Type is the type of outputter
	Type string
)

// Types of outputters
const (
	Stdout      Type = "stdout"
	EvmContract Type = "evmContract"
)