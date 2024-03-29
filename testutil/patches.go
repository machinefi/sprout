package testutil

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	. "github.com/agiledragon/gomonkey/v2"
	"github.com/ethereum/go-ethereum/ethclient"
)

func JsonMarshal(p *Patches, data []byte, err error) *Patches {
	return p.ApplyFunc(
		json.Marshal,
		func(v any) ([]byte, error) {
			return data, err
		},
	)
}

func JsonUnmarshal(p *Patches, err error) *Patches {
	return p.ApplyFunc(
		json.Unmarshal,
		func([]byte, any) error {
			return err
		},
	)
}

func HttpPost(p *Patches, rsp *http.Response, err error) *Patches {
	return p.ApplyFunc(
		http.Post,
		func(string, string, io.Reader) (*http.Response, error) {
			return rsp, err
		},
	)
}

func HttpGet(p *Patches, rsp *http.Response, err error) *Patches {
	return p.ApplyFunc(
		http.Get,
		func(string) (*http.Response, error) {
			return rsp, err
		},
	)
}

func HttpNewRequest(p *Patches, req *http.Request, err error) *Patches {
	return p.ApplyFunc(
		http.NewRequest,
		func(method, url string, body io.Reader) (*http.Request, error) {
			return req, err
		},
	)
}

func URLParse(p *Patches, rsp *url.URL, err error) *Patches {
	return p.ApplyFunc(
		url.Parse,
		func(string) (*url.URL, error) {
			return rsp, err
		},
	)
}

func IoReadAll(p *Patches, data []byte, err error) *Patches {
	return p.ApplyFunc(
		io.ReadAll,
		func(io.Reader) ([]byte, error) {
			return data, err
		},
	)
}

func EthClientDial(p *Patches, c *ethclient.Client, err error) *Patches {
	return p.ApplyFunc(
		ethclient.Dial,
		func(_ string) (*ethclient.Client, error) {
			return c, err
		},
	)
}
