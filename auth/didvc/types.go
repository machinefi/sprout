// this file defines Verifiable Credential datatype.
// ref: https://w3c-ccg.github.io/vc-api/
package didvc

// Issuer defines a JSON-LD Verifiable Credential Issuer
type Issuer struct {
	ID string `json:"id"`
}

// Credential defines a json-ld Verifiable Credential without a proof
type Credential struct {
	// Context the JSON-LD context of the credential
	Context []string `json:"@context"`
	// Type the JSON-LD type of the credential
	Type []string `json:"type"`
	// ID the ID of the credential
	ID string `json:"id"`
	// Issuer expressing the issuer of the VC
	Issuer Issuer `json:"issuer"`
	// IssuanceDate expressing the time when a VC becomes valid
	IssuanceDate string `json:"issuanceDate"`
	// ExpirationDate expressing the time when a VC expired
	ExpirationDate string `json:"expirationDate,omitempty"`
	// CredentialSubject expression of claims about one or more subjects
	CredentialSubject interface{} `json:"credentialSubject"`
}

// LinkedDataProofOptions options for specifying how the LinkedDataProof is created
type LinkedDataProofOptions struct {
	// VerificationMethod the URI of the VerificationMethod used for the proof
	// If omitted, a default assertionMethod will be used.
	VerificationMethod string `json:"verificationMethod,omitempty"`
	// ProofPurpose the purpose of the proof
	// If omitted, "assertionMethod" will be used.
	ProofPurpose string `json:"proofPurpose,omitempty"`
	// ProofFormat the format of proof
	ProofFormat `json:"proofFormat,omitempty"`
	// Created the date of the proof.
	// If omitted system time will be used.
	Created string `json:"created,omitempty"`
	// Challenge the challenge of the proof
	Challenge string `json:"challenge,omitempty"`
	// Domain the domain of the proof
	Domain string `json:"domain,omitempty"`
}

// LinkedDataProof defines a json-d linked data proof
type LinkedDataProof struct {
	// Type linked data signature suite used to produce proof.
	Type string `json:"type"`
	// Created date the proof was created
	Created string `json:"created"`
	// VerificationMethod verification method ued to verify proof
	VerificationMethod string `json:"verificationMethod"`
	// ProofPurpose the purpose of the proof to be used
	ProofPurpose string `json:"proofPurpose"`
	// JWS detached JSON Web Signature
	JWS string `json:"jws"`
}

// VerifiableCredential a json-ld Verifiable Credential with a proof.
type VerifiableCredential struct {
	Credential
	LinkedDataProof `json:"proof"`
}

type ProofFormat string

const (
	ProofFormatLDP ProofFormat = "ldp"
	ProofFormatJWT ProofFormat = "jwt"
)

// TODO define Verifiable Presentation datatype

// TODO define VC exchange
