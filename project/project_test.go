package project

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/machinefi/sprout/output"
	"github.com/machinefi/sprout/testutil"
	"github.com/machinefi/sprout/types"
	"github.com/machinefi/sprout/utils/ipfs"
)

func TestConfig_GetOutput(t *testing.T) {
	r := require.New(t)

	p := gomonkey.NewPatches()
	defer p.Reset()

	p.ApplyFuncReturn(output.NewEthereum, &output.EthereumContract{}, nil)
	p.ApplyFuncReturn(output.NewSolanaProgram, &output.SolanaProgram{}, nil)
	p.ApplyFuncReturn(output.NewTextileDBAdapter, &output.TextileDB{}, nil)
	p.ApplyFuncReturn(output.NewStdout, &output.Stdout{})

	out, err := (&Config{Output: OutputConfig{Type: types.OutputEthereumContract}}).GetOutput("any", "any")
	r.IsType(out, &output.EthereumContract{})
	r.NoError(err)

	out, err = (&Config{Output: OutputConfig{Type: types.OutputSolanaProgram}}).GetOutput("any", "any")
	r.IsType(out, &output.SolanaProgram{})
	r.NoError(err)

	out, err = (&Config{Output: OutputConfig{Type: types.OutputTextile}}).GetOutput("any", "any")
	r.IsType(out, &output.TextileDB{})
	r.NoError(err)

	out, err = (&Config{Output: OutputConfig{Type: types.OutputStdout}}).GetOutput("any", "any")
	r.IsType(out, &output.Stdout{})
	r.NoError(err)

	out, err = (&Config{}).GetOutput("any", "any")
	r.IsType(out, &output.Stdout{})
	r.NoError(err)
}

type DummyHash struct {
	bytes.Buffer
}

func (h *DummyHash) Sum(b []byte) []byte {
	return b
}

func (h *DummyHash) Reset() {}

func (h *DummyHash) Size() int { return 0 }

func (h *DummyHash) BlockSize() int { return 0 }

func TestProjectMeta_GetConfigs(t *testing.T) {
	r := require.New(t)

	t.Run("ParseURI", func(t *testing.T) {
		t.Run("FailedToParseURL", func(t *testing.T) {
			p := gomonkey.NewPatches()
			defer p.Reset()

			p = p.ApplyFuncReturn(url.Parse, nil, errors.New(t.Name()))

			_, err := (&ProjectMeta{}).GetConfigs("")
			r.ErrorContains(err, t.Name())
		})
	})

	t.Run("FetchProjectConfigByURL", func(t *testing.T) {
		t.Run("FailedToFetchProjectConfig", func(t *testing.T) {
			t.Run("FromIPFS", func(t *testing.T) {
				p := gomonkey.NewPatches()
				defer p.Reset()

				p = p.ApplyMethodReturn(&ipfs.IPFS{}, "Cat", nil, errors.New(t.Name()))

				cases := []*struct {
					Scheme string
					Meta   *ProjectMeta
				}{
					{
						Scheme: "URL",
						Meta:   &ProjectMeta{Uri: "ipfs://ipfshost/endpoint/cid"},
					},
					{
						Scheme: "RawCID",
						Meta:   &ProjectMeta{Uri: "cid"},
					},
				}

				for _, c := range cases {
					configs, err := c.Meta.GetConfigs("any")
					r.Len(configs, 0)
					r.ErrorContains(err, t.Name())
				}
			})
			t.Run("FromHTTP(s)", func(t *testing.T) {
				t.Run("FailedToHTTPGet", func(t *testing.T) {
					p := gomonkey.NewPatches()
					defer p.Reset()

					p = p.ApplyFuncReturn(http.Get, nil, errors.New(t.Name()))

					configs, err := (&ProjectMeta{Uri: "http://ipfshost/endpoint/cid"}).GetConfigs("any")
					r.Len(configs, 0)
					r.ErrorContains(err, t.Name())
				})

				t.Run("FailedToReadHTTPRespondedBody", func(t *testing.T) {
					p := gomonkey.NewPatches()
					defer p.Reset()

					p = p.ApplyFuncReturn(http.Get, &http.Response{
						Body: io.NopCloser(bytes.NewReader([]byte("any"))),
					}, nil)
					p = p.ApplyFuncReturn(io.ReadAll, nil, errors.New(t.Name()))

					configs, err := (&ProjectMeta{Uri: "http://ipfshost/endpoint/cid"}).GetConfigs("any")
					r.Len(configs, 0)
					r.ErrorContains(err, t.Name())
				})
			})
		})
	})

	correcthash := [32]byte{1, 2, 3}
	wronghash := [32]byte{3, 2, 1}
	t.Run("CalAndCheckContentSHA256Sum", func(t *testing.T) {
		pm := &ProjectMeta{
			Uri:  "ipfs://ipfshost/cid",
			Hash: correcthash,
		}

		t.Run("FailedToCalSum", func(t *testing.T) {
			p := gomonkey.NewPatches()
			defer p.Reset()

			p = p.ApplyMethodReturn(&ipfs.IPFS{}, "Cat", []byte("any"), nil)
			p = p.ApplyFuncReturn(sha256.New, &DummyHash{})
			p = p.ApplyMethodReturn(&bytes.Buffer{}, "Write", 0, errors.New(t.Name()))

			configs, err := pm.GetConfigs("any")
			r.Len(configs, 0)
			r.ErrorContains(err, t.Name())
		})

		t.Run("FailedToCheckSHA256Sum", func(t *testing.T) {
			p := gomonkey.NewPatches()
			defer p.Reset()

			p = p.ApplyMethodReturn(&ipfs.IPFS{}, "Cat", []byte("any"), nil)
			p = p.ApplyFuncReturn(sha256.New, &DummyHash{})
			p = p.ApplyMethodReturn(&bytes.Buffer{}, "Write", 0, nil)
			p = p.ApplyMethodReturn(&DummyHash{}, "Sum", wronghash[:])

			configs, err := pm.GetConfigs("any")
			r.Len(configs, 0)
			r.ErrorContains(err, "validate project config hash failed")
		})
	})

	t.Run("ParseAndValidateContent", func(t *testing.T) {
		pm := &ProjectMeta{
			Uri:  "ipfs://ipfshost/cid",
			Hash: [32]byte{1, 2, 3},
		}

		p := gomonkey.NewPatches()
		defer p.Reset()

		p = p.ApplyMethodReturn(&ipfs.IPFS{}, "Cat", []byte("any"), nil)
		p = p.ApplyFuncReturn(sha256.New, &DummyHash{})
		p = p.ApplyMethodReturn(&bytes.Buffer{}, "Write", 0, nil)
		p = p.ApplyMethodReturn(&DummyHash{}, "Sum", correcthash[:])

		t.Run("FailedToJsonUnmarshalContent", func(t *testing.T) {
			p = testutil.JsonUnmarshal2(p, nil, errors.New(t.Name()))

			configs, err := pm.GetConfigs("any")
			r.Len(configs, 0)
			r.ErrorContains(err, t.Name())
		})

		t.Run("EmptyConfigs", func(t *testing.T) {
			p = testutil.JsonUnmarshal2(p, &([]*Config{}), nil)

			configs, err := pm.GetConfigs("any")
			r.Len(configs, 0)
			r.ErrorContains(err, "empty project config")
		})

		t.Run("InvalidConfig", func(t *testing.T) {
			p = testutil.JsonUnmarshal2(p, &([]*Config{{
				Code: "",
			}}), nil)

			configs, err := pm.GetConfigs("any")
			r.Len(configs, 0)
			r.ErrorContains(err, "invalid project config")
		})

		t.Run("Success", func(t *testing.T) {
			p = testutil.JsonUnmarshal2(p, &([]*Config{{
				Code:    "any",
				VMType:  "any",
				Version: "any",
			}}), nil)

			configs, err := pm.GetConfigs("any")
			r.Len(configs, 1)
			r.NoError(err)
		})

	})

}

func TestProjectMeta_GetConfigs_http(t *testing.T) {
	r := require.New(t)
	p := gomonkey.NewPatches()
	defer p.Reset()

	cs := []*Config{
		{
			Code:    "i am code",
			VMType:  types.VMHalo2,
			Version: "0.1",
		},
	}
	jc, err := json.Marshal(cs)
	r.NoError(err)

	h := sha256.New()
	_, err = h.Write(jc)
	r.NoError(err)
	hash := h.Sum(nil)

	pm := &ProjectMeta{
		ProjectID: 1,
		Uri:       "https://test.com/project_config",
		Hash:      [32]byte(hash),
	}

	t.Run("FailedToGetHTTP", func(t *testing.T) {
		p = p.ApplyFuncReturn(http.Get, nil, errors.New(t.Name()))

		_, err := pm.GetConfigs("")
		r.ErrorContains(err, t.Name())
	})
	t.Run("FailedToIOReadAll", func(t *testing.T) {
		p = p.ApplyFuncReturn(http.Get, &http.Response{
			Body: io.NopCloser(bytes.NewReader(jc)),
		}, nil)
		p = p.ApplyFuncReturn(io.ReadAll, nil, errors.New(t.Name()))

		_, err := pm.GetConfigs("")
		r.ErrorContains(err, t.Name())
	})
	t.Run("HashMismatch", func(t *testing.T) {
		p = p.ApplyFuncReturn(io.ReadAll, jc, nil)

		npm := *pm
		npm.Hash = [32]byte{}
		_, err := npm.GetConfigs("")
		r.ErrorContains(err, "validate project config hash failed")
	})
	t.Run("Success", func(t *testing.T) {
		resultConfigs, err := pm.GetConfigs("")
		r.NoError(err)
		r.Equal(len(resultConfigs), len(cs))
		r.Equal(resultConfigs[0].Code, "i am code")
	})
	t.Run("FailedToUnmarshalJson", func(t *testing.T) {
		p = p.ApplyFuncReturn(json.Unmarshal, errors.New(t.Name()))

		_, err := pm.GetConfigs("")
		r.ErrorContains(err, t.Name())
	})
}

func TestProjectMeta_GetConfigs_ipfs(t *testing.T) {
	r := require.New(t)
	p := gomonkey.NewPatches()
	defer p.Reset()

	pm := &ProjectMeta{
		Uri: "ipfs://test.com/123",
	}
	t.Run("FailedToGetIPFS", func(t *testing.T) {
		p = p.ApplyMethodReturn(&ipfs.IPFS{}, "Cat", nil, errors.New(t.Name()))

		_, err := pm.GetConfigs("")
		r.ErrorContains(err, t.Name())
	})
}

func TestProjectMeta_GetConfigs_default(t *testing.T) {
	r := require.New(t)
	p := gomonkey.NewPatches()
	defer p.Reset()

	pm := &ProjectMeta{
		Uri: "test.com/123",
	}

	t.Run("FailedToGetIPFS", func(t *testing.T) {
		p = p.ApplyMethodReturn(&ipfs.IPFS{}, "Cat", nil, errors.New(t.Name()))

		_, err := pm.GetConfigs("")
		r.ErrorContains(err, t.Name())
	})
}
