package task

import (
	"os"
	"strings"
	"testing"

	"github.com/emersonjoe/trilha-spec/spec"
)

// signerFor makes a key pair, commits the public half to .trilha/keys and
// answers a signer for it — a person's key is a key like a runner's.
func signerFor(t *testing.T, l spec.Layout, keyID string) *Signer {
	t.Helper()
	privPEM, pubPEM, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(l.Keys(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(l.Keys()+string(os.PathSeparator)+keyID+".pub", pubPEM, 0o644); err != nil {
		t.Fatal(err)
	}
	priv, err := ParsePrivateKey(privPEM)
	if err != nil {
		t.Fatal(err)
	}
	return &Signer{KeyID: keyID, Key: priv}
}

func TestReviewPolicyRoundTrip(t *testing.T) {
	st := newStore(t)
	tk, err := st.Create("Homologacao", func(x *Task) {
		x.Status = Review
		x.Acceptance = []string{"o cliente aceita"}
		x.Review = &ReviewPolicy{Quorum: 2, Roles: []string{"uat", "legal"}}
	})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(st.Layout.TaskFile(tk.ID))
	if want := "review:\n  quorum: 2\n  roles:\n    - uat\n    - legal\n"; !strings.Contains(string(b), want) {
		t.Fatalf("review written as:\n%s", b)
	}
	got, err := st.Get(tk.ID)
	if err != nil || got.Review == nil || got.Review.Quorum != 2 || strings.Join(got.Review.Roles, ",") != "uat,legal" {
		t.Fatalf("round trip: %v %+v", err, got.Review)
	}
	got.Review = nil
	if err := st.Save(got); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(st.Layout.TaskFile(tk.ID)); strings.Contains(string(b), "review:") {
		t.Fatalf("empty policy written:\n%s", b)
	}
	bad := &Task{ID: "TASK-002", Title: "B", Status: Idea, Review: &ReviewPolicy{Quorum: 1, Roles: []string{"Legal Team"}}}
	if err := bad.Validate(); err == nil || !strings.Contains(err.Error(), "is not a role") {
		t.Fatalf("bad role accepted: %v", err)
	}
}

func TestAttestationValidation(t *testing.T) {
	st := newStore(t)
	l := st.Layout
	e, path, err := Record(l, Attestation("TASK-001", "Ana Souza", "uat", "Homologado em 2027-04-28 com a equipe da prefeitura.", "#2", "docs/ata.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	for _, want := range []string{`"kind": "attestation"`, `"by": "Ana Souza"`, `"role": "uat"`, `"statement": "Homologado`, `"docs/ata.pdf"`} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("record missing %s:\n%s", want, raw)
		}
	}
	if !strings.HasSuffix(strings.ReplaceAll(path, "\\", "/"), "001-attestation.json") || e.Seq != 1 {
		t.Fatalf("path = %s", path)
	}
	for _, bad := range []Evidence{
		{Task: "TASK-001", Kind: KindAttestation, Role: "uat", Statement: "ok"},
		{Task: "TASK-001", Kind: KindAttestation, By: "Ana", Role: "UAT Team", Statement: "ok"},
		{Task: "TASK-001", Kind: KindAttestation, By: "Ana", Role: "uat"},
		{Task: "TASK-001", Kind: KindAttestation, By: "Ana", Role: "uat", Statement: "ok", Refs: []string{" "}},
	} {
		if _, _, err := Record(l, bad); err == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
}

func TestQuorumClosesTheTask(t *testing.T) {
	st := newStore(t)
	l := st.Layout
	ana := signerFor(t, l, "ana")
	bruno := signerFor(t, l, "bruno")
	tk, err := st.Create("Homologacao", func(x *Task) {
		x.Status = Ready
		x.Acceptance = []string{"o cliente aceita"}
		x.Review = &ReviewPolicy{Quorum: 2, Roles: []string{"uat", "legal"}}
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, to := range []Status{Running, Verify, Review} {
		if _, err := st.Move(tk.ID, to); err != nil {
			t.Fatal(err)
		}
	}
	// Nobody has attested: the move says what is missing.
	_, err = st.Move(tk.ID, Done)
	if err == nil || !strings.Contains(err.Error(), "needs 2 signed attestation(s) in roles uat, legal") {
		t.Fatalf("closed without attestations: %v", err)
	}
	// An unsigned attestation is a claim; it does not count.
	if _, _, err := Record(l, Attestation(tk.ID, "Carla", "uat", "parece ok")); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Move(tk.ID, Done); err == nil || !strings.Contains(err.Error(), "is unsigned") {
		t.Fatalf("an unsigned attestation counted: %v", err)
	}
	// One signed attestation is not two.
	if _, _, err := RecordSigned(l, Attestation(tk.ID, "Ana Souza", "uat", "homologado"), ana); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Move(tk.ID, Done); err == nil || !strings.Contains(err.Error(), "has 1 (ana as uat)") {
		t.Fatalf("one attestation closed a quorum of two: %v", err)
	}
	// The same person twice is still one person.
	if _, _, err := RecordSigned(l, Attestation(tk.ID, "Ana Souza", "legal", "e tambem juridicamente"), ana); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Move(tk.ID, Done); err == nil || !strings.Contains(err.Error(), "is ana again") {
		t.Fatalf("the same key counted twice: %v", err)
	}
	// A role the policy does not ask for does not count either.
	if _, _, err := RecordSigned(l, Attestation(tk.ID, "Bruno Lima", "marketing", "gostei"), bruno); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Move(tk.ID, Done); err == nil || !strings.Contains(err.Error(), "not a role the policy asks for") {
		t.Fatalf("an unwanted role counted: %v", err)
	}
	// Two distinct people, in the roles the policy asks for: the task closes.
	if _, _, err := RecordSigned(l, Attestation(tk.ID, "Bruno Lima", "legal", "sem impedimento"), bruno); err != nil {
		t.Fatal(err)
	}
	got, err := st.Move(tk.ID, Done)
	if err != nil || got.Status != Done {
		t.Fatalf("quorum met but refused: %v", err)
	}
	q, err := QuorumOf(l, got)
	if err != nil || !q.Met() || len(q.Have) != 2 {
		t.Fatalf("quorum = %+v %v", q, err)
	}
	// A task with no policy closes as it always did.
	plain, err := st.Create("Sem quorum", func(x *Task) {
		x.Status = Ready
		x.Acceptance = []string{"ok"}
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, to := range []Status{Running, Verify, Review, Done} {
		if _, err := st.Move(plain.ID, to); err != nil {
			t.Fatalf("a task without a policy must close: %v", err)
		}
	}
}

func TestAttestationDoctor(t *testing.T) {
	st := newStore(t)
	l := st.Layout
	// A key that is never committed: what it signs proves nothing here.
	privPEM, _, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	priv, err := ParsePrivateKey(privPEM)
	if err != nil {
		t.Fatal(err)
	}
	tk, err := st.Create("Homologacao", func(x *Task) {
		x.Status = Review
		x.Acceptance = []string{"ok"}
		x.Review = &ReviewPolicy{Quorum: 2}
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := RecordSigned(l, Attestation(tk.ID, "Dora", "uat", "ok"), &Signer{KeyID: "dora", Key: priv}); err != nil {
		t.Fatal(err)
	}
	tasks, err := st.List()
	if err != nil {
		t.Fatal(err)
	}
	problems, err := CheckAttestations(l, tasks)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, p := range problems {
		got = append(got, p.Code+":"+p.Arg)
	}
	joined := strings.Join(got, " ")
	if !strings.Contains(joined, spec.ProblemQuorumWithoutRoles+":"+tk.ID) {
		t.Fatalf("quorum without roles not reported: %s", joined)
	}
	if !strings.Contains(joined, spec.ProblemAttestationUnknownKey+":"+tk.ID+" #1 dora (unknown key dora)") {
		t.Fatalf("unknown key not reported: %s", joined)
	}
	// With no roles asked for, any role satisfies the quorum — which is why
	// doctor complains about it.
	q, err := QuorumOf(l, tasks[0])
	if err != nil || q.Missing != 2 {
		t.Fatalf("quorum = %+v %v", q, err)
	}
}
