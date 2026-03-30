package crypto

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"
)

type AgeEncrypter struct {
	identity  *age.X25519Identity
	recipient *age.X25519Recipient
}

func NewAgeEncrypter(keysDir string) (*AgeEncrypter, error) {
	if err := os.MkdirAll(keysDir, 0o700); err != nil {
		return nil, fmt.Errorf("create keys dir: %w", err)
	}

	keysFile := filepath.Join(keysDir, "keys.txt")

	identity, err := loadOrCreateIdentity(keysFile)
	if err != nil {
		return nil, fmt.Errorf("load identity from %s: %w", keysFile, err)
	}

	recipient := identity.Recipient()

	return &AgeEncrypter{
		identity:  identity,
		recipient: recipient,
	}, nil
}

func (e *AgeEncrypter) Encrypt(plaintext string) (string, error) {
	var buf bytes.Buffer

	w, err := age.Encrypt(&buf, e.recipient)
	if err != nil {
		return "", fmt.Errorf("age encrypt init: %w", err)
	}

	if _, err := io.WriteString(w, plaintext); err != nil {
		return "", fmt.Errorf("age encrypt write: %w", err)
	}

	if err := w.Close(); err != nil {
		return "", fmt.Errorf("age encrypt close: %w", err)
	}

	encoded := "age1" + base64.RawStdEncoding.EncodeToString(buf.Bytes())
	return encoded, nil
}

func (e *AgeEncrypter) Decrypt(ciphertext string) (string, error) {
	if !IsEncrypted(ciphertext) {
		return "", fmt.Errorf("value is not age-encrypted")
	}

	raw := strings.TrimPrefix(ciphertext, "age1")
	data, err := base64.RawStdEncoding.DecodeString(raw)
	if err != nil {
		return "", fmt.Errorf("decode age ciphertext: %w", err)
	}

	r, err := age.Decrypt(bytes.NewReader(data), e.identity)
	if err != nil {
		return "", fmt.Errorf("age decrypt: %w", err)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("age decrypt read: %w", err)
	}

	return string(out), nil
}

func loadOrCreateIdentity(path string) (*age.X25519Identity, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		return parseIdentity(data)
	}

	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read key file: %w", err)
	}

	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return nil, fmt.Errorf("generate age identity: %w", err)
	}

	content := fmt.Sprintf("# created by dops\n# public key: %s\n%s\n",
		identity.Recipient().String(),
		identity.String(),
	)

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return nil, fmt.Errorf("write key file: %w", err)
	}

	return identity, nil
}

func parseIdentity(data []byte) (*age.X25519Identity, error) {
	identities, err := age.ParseIdentities(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parse age identities: %w", err)
	}

	if len(identities) == 0 {
		return nil, fmt.Errorf("no identities found in key file")
	}

	id, ok := identities[0].(*age.X25519Identity)
	if !ok {
		return nil, fmt.Errorf("unexpected identity type")
	}

	return id, nil
}

var _ Encrypter = (*AgeEncrypter)(nil)
