package p2p

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	"github.com/stretchr/testify/require"
)

func TestDiscoverPeers(t *testing.T) {
	require := require.New(t)

	ctx := context.Background()
	bootNodeMultiaddr := "/dns4/bootnode-0.testnet.iotex.one/tcp/4689/ipfs/12D3KooWFnaTYuLo8Mkbm3wzaWHtUuaxBRe24Uiopu15Wr5EhD3o"
	iotexChainID := 2
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"), libp2p.Muxer("/yamux/2.0.0", yamux.DefaultTransport))
	require.NoError(err)

	t.Run("discoverPeers ok", func(t *testing.T) {
		err := discoverPeers(ctx, h, bootNodeMultiaddr, iotexChainID)
		require.NoError(err)
	})

	t.Run("bootNodeMultiaddr nil", func(t *testing.T) {
		err := discoverPeers(ctx, h, "", iotexChainID)
		require.EqualError(err, "parse boot node multiaddr failed: failed to parse multiaddr \"\": empty multiaddr")
	})

}
