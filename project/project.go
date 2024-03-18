package project

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/machinefi/sprout/utils/ipfs"
)

type ProjectMeta struct {
	ProjectID uint64
	Uri       string
	Hash      [32]byte
	Paused    bool
}

func (m *ProjectMeta) GetConfigData(ipfsEndpoint string) ([]byte, error) {
	u, err := url.Parse(m.Uri)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse project uri %s", m.Uri)
	}

	var data []byte
	switch u.Scheme {
	case "http", "https":
		resp, _err := http.Get(m.Uri)
		if _err != nil {
			return nil, errors.Wrapf(_err, "failed to fetch project config, project_id %d, uri %s", m.ProjectID, m.Uri)
		}
		defer resp.Body.Close()
		// TODO network error should try again
		data, err = io.ReadAll(resp.Body)

	case "ipfs":
		// ipfs url: ipfs://${endpoint}/${cid}
		sh := ipfs.NewIPFS(u.Host)
		cid := strings.Split(strings.Trim(u.Path, "/"), "/")
		data, err = sh.Cat(cid[0])

	default:
		// fetch content by ipfs cid with default endpoint
		sh := ipfs.NewIPFS(ipfsEndpoint)
		cid := strings.Split(strings.Trim(u.Path, "/"), "/")
		data, err = sh.Cat(cid[0])
	}

	if err != nil {
		return nil, errors.Wrapf(err, "failed to read project config, project_id %d, uri %s", m.ProjectID, m.Uri)
	}

	h := sha256.New()
	if _, err := h.Write(data); err != nil {
		return nil, errors.Wrap(err, "failed to generate project config hash")
	}
	if !bytes.Equal(h.Sum(nil), m.Hash[:]) {
		return nil, errors.New("failed to validate project config hash")
	}

	return data, nil
}

func convertConfigs(data []byte) ([]*Config, error) {
	cs := []*Config{}
	if err := json.Unmarshal(data, &cs); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal project config")
	}
	if len(cs) == 0 {
		return nil, errors.Errorf("empty project config")
	}
	for _, c := range cs {
		if c.Code == "" || c.VMType == "" || c.Version == "" {
			return nil, errors.Errorf("invalid project config")
		}
	}
	return cs, nil
}
