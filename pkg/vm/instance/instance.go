package instance

import (
	"context"

	"github.com/machinefi/w3bstream-mainnet/pkg/msg"
)

type Instance interface {
	Start() error
	Stop()
	Execute(ctx context.Context, msg *msg.Msg) ([]byte, error)
	Release()
}
