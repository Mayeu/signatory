package signatory

import (
	"context"
	"encoding/base64"

	"github.com/ecadlabs/signatory/pkg/tezos"
	log "github.com/sirupsen/logrus"
)

// Importer interface representing an importer backend
type Importer interface {
	Import(ctx context.Context, jwk *JWK) (string, error)
	Name() string
}

// JWK struct containing a standard key format
type JWK struct {
	KeyType string `json:"kty"`
	D       string `json:"d"`
	X       string `json:"x"`
	Y       string `json:"y"`
	Curve   string `json:"crv"`
}

// ImportedKey struct containing information about an imported key
type ImportedKey struct {
	Hash  string
	KeyID string
}

// KeyPair interface that represent an elliptic curve key pair
type KeyPair interface {
	X() []byte
	Y() []byte
	D() []byte
	CurveName() string
}

// Import a keyPair inside the vault
func (s *Signatory) Import(pubkey string, secretKey string, importer Importer) (*ImportedKey, error) {
	keyPair := tezos.NewKeyPair(pubkey, secretKey)
	err := keyPair.Validate()
	if err != nil {
		return nil, err
	}

	jwk, err := ToJWK(keyPair)
	if err != nil {
		return nil, err
	}

	hash, err := keyPair.PubKeyHash()
	if err != nil {
		return nil, err
	}

	logfields := log.Fields{
		LogPKH:   hash,
		LogVault: importer.Name(),
	}
	if n, ok := importer.(VaultNamer); ok {
		logfields[LogVaultName] = n.VaultName()
	}
	l := s.logger.WithFields(logfields)

	l.Info("Requesting import operation")

	keyID, err := importer.Import(context.TODO(), jwk)
	if err != nil {
		return nil, err
	}

	l.WithField(LogKeyID, keyID).Info("Successfully imported")

	importedKey := &ImportedKey{
		KeyID: keyID,
		Hash:  hash,
	}

	return importedKey, nil
}

// ToJWK Convert a keyPair to a JWK
func ToJWK(k KeyPair) (*JWK, error) {
	return &JWK{
		X:       base64.RawURLEncoding.EncodeToString(k.X()),
		Y:       base64.RawURLEncoding.EncodeToString(k.Y()),
		D:       base64.RawURLEncoding.EncodeToString(k.D()),
		KeyType: "EC",
		Curve:   k.CurveName(),
	}, nil
}
