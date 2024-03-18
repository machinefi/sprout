package output

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"github.com/tablelandnetwork/basin-cli/pkg/signing"
	"github.com/tidwall/gjson"

	"github.com/machinefi/sprout/types"
)

type TextileDB struct {
	endpoint  string
	secretKey *ecdsa.PrivateKey
}

func (t *TextileDB) Output(task *types.Task, proof []byte) (string, error) {
	slog.Debug("outputing to TextileDB", "chain endpoint", t.endpoint)
	encodedData, err := t.packData(proof)
	if err != nil {
		return "", err
	}
	txHash, err := t.write(encodedData)
	if err != nil {
		return "", err
	}
	return txHash, nil
}

func (t *TextileDB) packData(proof []byte) ([]byte, error) {
	proof, err := hex.DecodeString(string(proof))
	valueJournal := gjson.GetBytes(proof, "Stark.journal.bytes")
	//valueJournal := gjson.GetBytes(proof, "Snark.journal")
	if !valueJournal.Exists() {
		return nil, errors.New("proof does not contain journal")
	}

	// get result from proof
	var result string
	for _, value := range valueJournal.Array() {
		result += fmt.Sprint(value.Int())
	}

	data := map[string]string{
		"result": result,
		"proof":  string(proof),
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "marshal pack data error")
	}

	return jsonData, nil
}

func (t *TextileDB) write(data []byte) (string, error) {
	signatureBytes, err := signing.NewSigner(t.secretKey).SignBytes(data)
	if err != nil {
		return "", errors.Wrap(err, "failed to sign data")
	}

	url := fmt.Sprintf("%s?timestamp=%s&signature=%s",
		t.endpoint,
		strconv.FormatInt(time.Now().Unix(), 10),
		hex.EncodeToString(signatureBytes))
	err = writeEvent(url, data)
	return "", err
}

// writeEvent writes a file to a vault via the Basin API.
func writeEvent(url string, fileData []byte) error {
	req, err := http.NewRequest("POST", url, bytes.NewReader(fileData))
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}

	hn := sha256.New()
	hr := hn.Sum(fileData)
	req.Header.Set("filename", string(hr)[0:7])

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	slog.Debug("Write event", "response", string(responseBody))
	return nil
}

// TODO: refactor textile with a KV database adapter
func NewTextileDBAdapter(vaultID string, secretKey string) (Output, error) {
	if len(secretKey) == 0 {
		return nil, errors.New("secretkey is empty")
	}

	pk := crypto.ToECDSAUnsafe(common.FromHex(secretKey))

	return &TextileDB{
		endpoint:  fmt.Sprintf("https://basin.tableland.xyz/vaults/%s/events", vaultID),
		secretKey: pk,
	}, nil
}
