package vm

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"

	"github.com/machinefi/w3bstream-mainnet/pkg/msg"
	"github.com/machinefi/w3bstream-mainnet/pkg/vm/instance/manager"
)

type Handler struct {
	instanceMgr *manager.Mgr
}

func (r *Handler) Handle(msg *msg.Msg) ([]byte, error) {
	ins, err := r.instanceMgr.Acquire(msg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get instance")
	}
	slog.Debug("acquire risc0 instance success")
	defer r.instanceMgr.Release(msg.Key(), ins)

	res, err := ins.Execute(context.Background(), msg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute instance")
	}
	slog.Debug("ask risc0 generate proof success, the proof is")
	slog.Debug(string(res))
	return res, nil
}

func NewHandler(risc0ServerAddr, projectConfigFilePath string) *Handler {
	return &Handler{
		manager.NewMgr(&manager.Config{
			Risc0ServerAddr:       risc0ServerAddr,
			ProjectConfigFilePath: projectConfigFilePath,
		}),
	}
}
