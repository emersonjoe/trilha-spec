package task

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/emersonjoe/trilha-spec/spec"
)

// Signature proves who wrote an evidence record and that nobody edited it
// since. Sig is over Canonical(record) — the record without this field —
// so the file on disk can be re-serialized by any tool and still verify.
type Signature struct {
	// Alg is `ed25519`; a reader refuses any other.
	Alg string `json:"alg"`
	// KeyID names the public key, `<key_id>.pub` in the keys directory.
	KeyID string `json:"key_id"`
	// Sig is the signature, standard base64.
	Sig string `json:"sig"`
}

// AlgEd25519 is the one algorithm the protocol names.
const AlgEd25519 = "ed25519"

// Verdict is what Check answers about one record.
type Verdict string

const (
	Unsigned Verdict = "unsigned" // no signature: valid protocol, no proof
	Valid    Verdict = "valid"    // the signature checks against a known key
	Invalid  Verdict = "invalid"  // signed, but the key is unknown, the algorithm is not ed25519 or the bytes do not match
)

var reKeyID = regexp.MustCompile(`^[a-z0-9]+([.-][a-z0-9]+)*$`)

// ValidKeyID answers whether id can name a key file: lowercase words joined
// by `-` or `.` (`runner-01`, `cloud.2026`).
func ValidKeyID(id string) bool { return reKeyID.MatchString(id) }

// Canonical is the bytes a signature covers: the record without
// `signature`, as JSON with object keys sorted bytewise, no insignificant
// whitespace, no HTML escaping, numbers in their shortest round-trip form.
// Any JSON library that sorts keys produces the same bytes.
func Canonical(e Evidence) ([]byte, error) {
	e.Signature = nil
	raw, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, err
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(generic); err != nil { // sorts map keys
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}

// Signer signs records with one Ed25519 private key.
type Signer struct {
	KeyID string
	Key   ed25519.PrivateKey
}

// Sign fills e.Signature.
func (s *Signer) Sign(e *Evidence) error {
	if !ValidKeyID(s.KeyID) {
		return fmt.Errorf("evidence: %q is not a key id", s.KeyID)
	}
	if len(s.Key) != ed25519.PrivateKeySize {
		return errors.New("evidence: signer has no ed25519 private key")
	}
	msg, err := Canonical(*e)
	if err != nil {
		return err
	}
	e.Signature = &Signature{Alg: AlgEd25519, KeyID: s.KeyID, Sig: base64.StdEncoding.EncodeToString(ed25519.Sign(s.Key, msg))}
	return nil
}

// Keyring is the public keys a reader trusts, by key id.
type Keyring map[string]ed25519.PublicKey

// LoadKeys reads every `<key_id>.pub` in dir: PEM `PUBLIC KEY` (PKIX) of an
// Ed25519 key. A missing directory is an empty keyring: nothing verifies.
func LoadKeys(dir string) (Keyring, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return Keyring{}, nil
		}
		return nil, err
	}
	keys := Keyring{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".pub") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".pub")
		if !ValidKeyID(id) {
			return nil, fmt.Errorf("keys: %s is not a key id", e.Name())
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		pub, err := ParsePublicKey(b)
		if err != nil {
			return nil, fmt.Errorf("keys: %s: %w", e.Name(), err)
		}
		keys[id] = pub
	}
	return keys, nil
}

// Check answers the verdict on one record and, when Invalid, why.
func (k Keyring) Check(e Evidence) (Verdict, string) {
	if e.Signature == nil {
		return Unsigned, ""
	}
	if e.Signature.Alg != AlgEd25519 {
		return Invalid, "algorithm " + e.Signature.Alg + " is not ed25519"
	}
	pub, ok := k[e.Signature.KeyID]
	if !ok {
		return Invalid, "unknown key " + e.Signature.KeyID
	}
	sig, err := base64.StdEncoding.DecodeString(e.Signature.Sig)
	if err != nil {
		return Invalid, "signature is not base64"
	}
	msg, err := Canonical(e)
	if err != nil {
		return Invalid, err.Error()
	}
	if !ed25519.Verify(pub, msg, sig) {
		return Invalid, "signature does not match the record"
	}
	return Valid, ""
}

// Checked is a record with its verdict, for `evidence --verify --json`.
type Checked struct {
	Evidence
	Verdict Verdict `json:"verdict"`
	Reason  string  `json:"reason,omitempty"`
}

// CheckAll answers every record of a task with its verdict.
func (k Keyring) CheckAll(list []Evidence) []Checked {
	out := make([]Checked, 0, len(list))
	for _, e := range list {
		v, why := k.Check(e)
		out = append(out, Checked{Evidence: e, Verdict: v, Reason: why})
	}
	return out
}

// GenerateKey makes a new Ed25519 pair and answers both as PEM: the private
// key as PKCS #8 `PRIVATE KEY`, the public key as PKIX `PUBLIC KEY`.
func GenerateKey() (priv, pub []byte, err error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	p8, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return nil, nil, err
	}
	pkix, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return nil, nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: p8}),
		pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pkix}), nil
}

// ParsePrivateKey reads a PEM PKCS #8 Ed25519 private key.
func ParsePrivateKey(b []byte) (ed25519.PrivateKey, error) {
	block, _ := pem.Decode(b)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, errors.New("not a PEM PRIVATE KEY")
	}
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	priv, ok := k.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("not an ed25519 key")
	}
	return priv, nil
}

// ParsePublicKey reads a PEM PKIX Ed25519 public key.
func ParsePublicKey(b []byte) (ed25519.PublicKey, error) {
	block, _ := pem.Decode(b)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, errors.New("not a PEM PUBLIC KEY")
	}
	k, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub, ok := k.(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("not an ed25519 key")
	}
	return pub, nil
}

// KeyIDs answers the ids in a keyring, sorted.
func (k Keyring) KeyIDs() []string {
	ids := make([]string, 0, len(k))
	for id := range k {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// ProjectKeys loads the keyring of a layout: .trilha/keys.
func ProjectKeys(l spec.Layout) (Keyring, error) { return LoadKeys(l.Keys()) }
