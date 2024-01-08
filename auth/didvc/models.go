package didvc

type IssueCredentialReq struct {
	// Credential vc issue payload
	Credential Credential `json:"credential"`
	// Options linked data proof
	Options *LinkedDataProofOptions `json:"options,omitempty"`
}

type IssueCredentialRsp = VerifiableCredential

type VerifyCredentialReq struct {
	VerifiableCredential   `json:"verifiableCredential"`
	LinkedDataProofOptions `json:"options"`
}

type VerifyJWTCredentialReq struct {
	VerifiableCredential    string `json:"verifiableCredential"`
	*LinkedDataProofOptions `json:"options"`
}

type VerifyCredentialRsp struct {
	Checks   []string `json:"checks"`
	Warnings []string `json:"warnings"`
	Errors   []string `json:"errors"`
}
