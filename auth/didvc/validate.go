package didvc

import (
	"bytes"
	"encoding/json"
	"github.com/pkg/errors"
	"io"
	"log/slog"
	"net/http"
)

func VerifyJWTCredential(endpoint string, tok string) error {
	reqv := &VerifyJWTCredentialReq{
		VerifiableCredential: tok,
		LinkedDataProofOptions: &LinkedDataProofOptions{
			ProofFormat: ProofFormatJWT,
		},
	}
	reqbody, err := json.Marshal(reqv)
	if err != nil {
		return errors.Wrap(err, "")
	}
	slog.Info(string(reqbody))

	url := "http://" + endpoint + "/verify/credentials"
	slog.Info(url)

	rsp, err := http.Post(url, "application/json", bytes.NewReader(reqbody))
	if err != nil {
		return errors.Wrap(err, "failed to request verification")
	}

	defer rsp.Body.Close()
	rspbody, err := io.ReadAll(rsp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read verification body")
	}

	slog.Info(string(rspbody))

	if rsp.StatusCode != http.StatusOK {
		return errors.Errorf("failed to request verification: %d %s", rsp.StatusCode, string(rspbody))
	}

	rspv := &VerifyCredentialRsp{}
	if err = json.Unmarshal(rspbody, rspv); err != nil {
		return errors.Wrap(err, "failed to parse verification result")
	}
	slog.Info("verification check", "checks", rspv.Checks, "errors", rspv.Errors, "warns", rspv.Warnings)
	return nil
}
