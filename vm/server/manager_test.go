package server

import (
	"context"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestAcquire(t *testing.T) {
	require := require.New(t)
	patches := NewPatches()

	mgr := &Mgr{
		idle: make(map[uint64]*Instance),
	}

	projectID := uint64(0x1)

	t.Run("NotExist", func(t *testing.T) {
		patches = vmServerNewInstance(patches, errors.New(t.Name()))
		_, err := mgr.Acquire(projectID, "", "", "")
		require.ErrorContains(err, t.Name())
	})

	t.Run("Exist", func(t *testing.T) {
		mgr.idle[projectID] = &Instance{}
		_, err := mgr.Acquire(projectID, "", "", "")
		require.NoError(err)
	})
}

func vmServerNewInstance(p *Patches, err error) *Patches {
	return p.ApplyFunc(
		NewInstance,
		func(ctx context.Context, endpoint string, projectID uint64, executeBinary string, expParam string) (*Instance, error) {
			return nil, err
		},
	)
}
