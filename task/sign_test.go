package task

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSignAndVerifyEvidence(t *testing.T) {
	s := newStore(t)
	tk, _ := s.Create("Signed", func(x *Task) { x.Acceptance = []string{"x"} })
	privPEM, pubPEM, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	priv, err := ParsePrivateKey(privPEM)
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(s.Layout.Keys(), 0o755)
	os.WriteFile(filepath.Join(s.Layout.Keys(), "runner-01.pub"), pubPEM, 0o644)

	signer := &Signer{KeyID: "runner-01", Key: priv}
	e, p, err := RecordSigned(s.Layout, Evidence{Task: tk.ID, Kind: "run", By: "runner-01", Passed: true, Model: "m", Cost: 0.0421, Currency: "USD", Note: "a <b> & c"}, signer)
	if err != nil || e.Signature == nil || e.Signature.Alg != "ed25519" || e.Signature.KeyID != "runner-01" {
		t.Fatalf("sign: %v %+v", err, e.Signature)
	}
	b, _ := os.ReadFile(p)
	if !strings.Contains(string(b), `"signature": {`) || !strings.Contains(string(b), `"key_id": "runner-01"`) {
		t.Fatalf("file:\n%s", b)
	}
	// Canonical form: sorted keys, compact, no HTML escaping, signature left out.
	c, _ := Canonical(e)
	if !strings.HasPrefix(string(c), `{"at":"`) || strings.Contains(string(c), "signature") || strings.Contains(string(c), `": `) || !strings.Contains(string(c), `"note":"a <b> & c"`) || !strings.Contains(string(c), `"cost":0.0421,"currency"`) {
		t.Fatalf("canonical: %s", c)
	}
	keys, err := ProjectKeys(s.Layout)
	if err != nil || len(keys) != 1 || keys.KeyIDs()[0] != "runner-01" {
		t.Fatalf("keys: %v %v", err, keys)
	}
	// Re-read from disk: the verdict is on what a tool would see.
	list, _ := ListEvidence(s.Layout, tk.ID)
	if v, why := keys.Check(list[0]); v != Valid {
		t.Fatalf("verdict %s: %s", v, why)
	}
	// An unsigned record is unsigned, not invalid.
	Record(s.Layout, Evidence{Task: tk.ID, Kind: "note", By: "me", Note: "n", Passed: true})
	list, _ = ListEvidence(s.Layout, tk.ID)
	checked := keys.CheckAll(list)
	if len(checked) != 2 || checked[0].Verdict != Valid || checked[1].Verdict != Unsigned {
		t.Fatalf("checked: %+v", checked)
	}
	out, _ := json.Marshal(checked[0])
	if !strings.Contains(string(out), `"verdict":"valid"`) || !strings.Contains(string(out), `"seq":1`) {
		t.Fatalf("json: %s", out)
	}
	// Edit the record after signing: invalid.
	edited := list[0]
	edited.Passed = false
	if v, why := keys.Check(edited); v != Invalid || !strings.Contains(why, "does not match") {
		t.Fatalf("edited: %s %s", v, why)
	}
	// Unknown key, wrong algorithm, empty keyring.
	unknown := list[0]
	unknown.Signature = &Signature{Alg: "ed25519", KeyID: "nobody", Sig: unknown.Signature.Sig}
	if v, why := keys.Check(unknown); v != Invalid || !strings.Contains(why, "unknown key nobody") {
		t.Fatalf("unknown key: %s %s", v, why)
	}
	rsa := list[0]
	rsa.Signature = &Signature{Alg: "rsa", KeyID: "runner-01", Sig: rsa.Signature.Sig}
	if v, _ := keys.Check(rsa); v != Invalid {
		t.Fatalf("rsa accepted: %s", v)
	}
	if v, _ := (Keyring{}).Check(list[0]); v != Invalid {
		t.Fatalf("empty keyring accepted: %s", v)
	}
	if empty, err := LoadKeys(filepath.Join(t.TempDir(), "none")); err != nil || len(empty) != 0 {
		t.Fatalf("missing dir: %v %v", err, empty)
	}
	// Bad ids and bad keys are refused.
	if err := (&Signer{KeyID: "Bad Id", Key: priv}).Sign(&e); err == nil {
		t.Fatal("bad key id accepted")
	}
	os.WriteFile(filepath.Join(s.Layout.Keys(), "junk.pub"), []byte("not pem"), 0o644)
	if _, err := ProjectKeys(s.Layout); err == nil {
		t.Fatal("junk key accepted")
	}
	if _, err := ParsePublicKey(privPEM); err == nil {
		t.Fatal("private PEM accepted as public")
	}
	// A private key inside .trilha is a doctor problem.
	os.Remove(filepath.Join(s.Layout.Keys(), "junk.pub"))
	os.WriteFile(filepath.Join(s.Layout.Keys(), "runner-01.key"), privPEM, 0o600)
	found := false
	for _, p := range s.Layout.Doctor() {
		if p.Code == "private-key-in-repo" && strings.HasSuffix(p.Arg, "keys/runner-01.key") {
			found = true
		}
	}
	if !found {
		t.Fatalf("doctor missed the private key: %v", s.Layout.Doctor())
	}
}
