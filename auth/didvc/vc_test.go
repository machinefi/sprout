package didvc_test

import (
	"bytes"
	"encoding/json"
	"github.com/machinefi/sprout/auth/didvc"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
	"time"
)

var (
	issueURI  = "http://127.0.0.1:3000/issue/cridentials"
	verifyURI = "http://127.0.0.1:3000/verify/cridentials"

	vc     *didvc.VerifiableCredential
	client = &http.Client{Timeout: time.Second * 5}
)

const (
	gDefaultContext  = "https://www.w3.org/2018/credentials/v1"
	gDefaultCredType = "VerifiableCredential"
)

func DISABLED_TestVCIssueAndVerify(t *testing.T) {
	// issue
	issuereq := didvc.IssueCredentialReq{
		Credential: didvc.Credential{
			Context: []string{gDefaultContext},
			Type:    []string{gDefaultCredType},
			ID:      "urn:uuid:040d4921-4756-447b-99ad-8d4978420e91",
			Issuer:  didvc.Issuer{ID: "did:key:z6MkgYAGxLBSXa6Ygk1PnUbK2F7zya8juE9nfsZhrvY7c9GD"},
		},
	}
	body, err := json.Marshal(issuereq)
	require.NoError(t, err)

	issuersp, err := client.Post(issueURI, "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	require.Equal(t, issuersp.StatusCode, http.StatusCreated)

	body, err = io.ReadAll(issuersp.Body)
	require.NoError(t, err)
	t.Log(string(body))

	// verify
	vc = &didvc.VerifiableCredential{}
	err = json.Unmarshal(body, vc)
	require.NoError(t, err)

	verifyreq := &didvc.VerifyCredentialReq{VerifiableCredential: *vc}

	body, err = json.Marshal(verifyreq)
	require.NoError(t, err)

	verifyrsp, err := client.Post(verifyURI, "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	require.Equal(t, verifyrsp.StatusCode, http.StatusOK)

	body, err = io.ReadAll(verifyrsp.Body)
	require.NoError(t, err)
	t.Log(string(body))
}
