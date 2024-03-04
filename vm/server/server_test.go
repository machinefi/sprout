package server

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	"google.golang.org/grpc"
)

func TestNewInstance(t *testing.T) {
	t.SkipNow()
	require := require.New(t)
	patches := NewPatches()

	t.Run("RpcFailed", func(t *testing.T) {
		patches = grpcDial(patches, errors.New(t.Name()))
		_, err := NewInstance(nil, "", uint64(0x1), "", "")
		fmt.Println(err.Error())
		require.ErrorContains(err, t.Name())
	})
}

func grpcDial(p *Patches, err error) *Patches {
	return p.ApplyFunc(
		grpc.Dial,
		func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
			return nil, err
		},
	)
}
