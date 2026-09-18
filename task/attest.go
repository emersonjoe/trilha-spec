package task

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/emersonjoe/trilha-spec/spec"
)

// KindAttestation is the evidence kind a person signs: "I, in this role,
// attest this". A `note` says somebody typed something; an attestation says
// who, in what capacity, and — once it is signed — proves it.
const KindAttestation = "attestation"

var reRole = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ValidRole answers whether r can name a role: lowercase words joined by `-`
// (`uat`, `legal`, `procurement`), the same shape as an agent name.
func ValidRole(r string) bool { return reRole.MatchString(r) }

// ReviewPolicy is what a task needs before it can close: how many people must
// attest, and in which roles. The protocol carries it and the writer enforces
// it on `review → done`; who asks whom is a workflow, not a protocol.
type ReviewPolicy struct {
	Quorum int      `json:"quorum,omitempty"`
	Roles  []string `json:"roles,omitempty"`
}

// reviewFrom reads the `review:` block of a task.
func reviewFrom(f spec.Fields) *ReviewPolicy {
	sub, ok := f.GetFields("review")
	if !ok {
		return nil
	}
	p := &ReviewPolicy{Roles: sub.GetList("roles")}
	p.Quorum, _ = strconv.Atoi(strings.TrimSpace(sub.Get("quorum")))
	return p
}

// fields renders the policy back as a block.
func (p *ReviewPolicy) fields() spec.Fields {
	var f spec.Fields
	f.Set("quorum", strconv.Itoa(p.Quorum))
	if len(p.Roles) > 0 {
		f.SetList("roles", p.Roles)
	}
	return f
}

// validate checks the policy a reader relies on.
func (p *ReviewPolicy) validate() []string {
	if p == nil {
		return nil
	}
	var errs []string
	if p.Quorum < 0 {
		errs = append(errs, "review.quorum cannot be negative")
	}
	for _, r := range p.Roles {
		if !ValidRole(r) {
			errs = append(errs, fmt.Sprintf("review.roles %q is not a role", r))
		}
	}
	return errs
}

// validateAttestation checks an attestation record: who, in what role, saying
// what. A record missing any of the three is a note wearing a costume.
func (e Evidence) validateAttestation() error {
	if e.Kind != KindAttestation {
		return nil
	}
	switch {
	case strings.TrimSpace(e.By) == "":
		return fmt.Errorf("evidence: an attestation needs `by`")
	case !ValidRole(e.Role):
		return fmt.Errorf("evidence: %q is not a role (lowercase words joined by -)", e.Role)
	case strings.TrimSpace(e.Statement) == "":
		return fmt.Errorf("evidence: an attestation needs a statement")
	}
	for _, r := range e.Refs {
		if strings.TrimSpace(r) == "" {
			return fmt.Errorf("evidence: an attestation has an empty ref")
		}
	}
	return nil
}

// Attestation builds one record. Signing it is the caller's business: an
// unsigned attestation is a claim, and a claim does not make quorum.
func Attestation(taskID, by, role, statement string, refs ...string) Evidence {
	return Evidence{Task: taskID, Kind: KindAttestation, By: by, Role: role,
		Statement: statement, Refs: refs, Passed: true}
}

// Attested is one signed attestation that counts towards a quorum.
type Attested struct {
	Seq       int    `json:"seq"`
	By        string `json:"by"`
	Role      string `json:"role"`
	KeyID     string `json:"key_id"`
	Statement string `json:"statement,omitempty"`
}

// Quorum is where a task stands against its review policy: what it needs,
// what it has and what is still missing. It is what the context pack shows an
// agent and what a refused `done` explains.
type Quorum struct {
	Required int        `json:"required"`
	Roles    []string   `json:"roles,omitempty"`
	Have     []Attested `json:"have,omitempty"`
	Missing  int        `json:"missing"`
	// Rejected is the attestations that do not count and why: unsigned, a
	// key nobody knows, a role the policy does not ask for, or a person who
	// already attested.
	Rejected []string `json:"rejected,omitempty"`
}

// Met answers whether the task may close.
func (q *Quorum) Met() bool { return q == nil || q.Missing <= 0 }

// Reason says, in one line, what is still missing.
func (q *Quorum) Reason() string {
	if q.Met() {
		return ""
	}
	s := fmt.Sprintf("needs %d signed attestation(s)", q.Required)
	if len(q.Roles) > 0 {
		s += " in roles " + strings.Join(q.Roles, ", ")
	}
	s += fmt.Sprintf("; has %d", len(q.Have))
	for _, a := range q.Have {
		s += fmt.Sprintf(" (%s as %s)", a.KeyID, a.Role)
	}
	if len(q.Rejected) > 0 {
		s += "; not counted: " + strings.Join(q.Rejected, ", ")
	}
	return s
}

// QuorumOf answers where a task stands. A task without a review policy has no
// quorum and answers nil: nothing changes for a project that does not use
// this.
func QuorumOf(l spec.Layout, t *Task) (*Quorum, error) {
	if t.Review == nil || t.Review.Quorum <= 0 {
		return nil, nil
	}
	list, err := ListEvidence(l, t.ID)
	if err != nil {
		return nil, err
	}
	keys, err := ProjectKeys(l)
	if err != nil {
		return nil, err
	}
	q := &Quorum{Required: t.Review.Quorum, Roles: t.Review.Roles}
	wanted := map[string]bool{}
	for _, r := range t.Review.Roles {
		wanted[r] = true
	}
	seen := map[string]bool{}
	for _, e := range list {
		if e.Kind != KindAttestation {
			continue
		}
		who := fmt.Sprintf("#%d %s", e.Seq, e.By)
		verdict, why := keys.Check(e)
		switch {
		case verdict == Unsigned:
			q.Rejected = append(q.Rejected, who+" is unsigned")
		case verdict == Invalid:
			q.Rejected = append(q.Rejected, who+": "+why)
		case len(wanted) > 0 && !wanted[e.Role]:
			q.Rejected = append(q.Rejected, who+" is "+e.Role+", not a role the policy asks for")
		case seen[e.Signature.KeyID]:
			q.Rejected = append(q.Rejected, who+" is "+e.Signature.KeyID+" again")
		default:
			seen[e.Signature.KeyID] = true
			q.Have = append(q.Have, Attested{Seq: e.Seq, By: e.By, Role: e.Role, KeyID: e.Signature.KeyID, Statement: e.Statement})
		}
	}
	q.Missing = q.Required - len(q.Have)
	return q, nil
}

// Attestation problem codes live with the others in the spec package; these
// are the checks themselves.

// CheckAttestations answers what is wrong with the human side of review: a
// quorum declared without the roles that satisfy it — which would let any
// role close anything — and an attestation signed with a key the project does
// not know, which proves nothing.
func CheckAttestations(l spec.Layout, tasks []*Task) ([]spec.Problem, error) {
	keys, err := ProjectKeys(l)
	if err != nil {
		return nil, err
	}
	var problems []spec.Problem
	for _, t := range tasks {
		if t.Review != nil && t.Review.Quorum > 0 && len(t.Review.Roles) == 0 {
			problems = append(problems, spec.Problem{Code: spec.ProblemQuorumWithoutRoles, Arg: t.ID})
		}
		list, err := ListEvidence(l, t.ID)
		if err != nil {
			return nil, err
		}
		for _, e := range list {
			if e.Kind != KindAttestation || e.Signature == nil {
				continue
			}
			if v, why := keys.Check(e); v == Invalid {
				problems = append(problems, spec.Problem{Code: spec.ProblemAttestationUnknownKey,
					Arg: fmt.Sprintf("%s #%d %s (%s)", t.ID, e.Seq, e.Signature.KeyID, why)})
			}
		}
	}
	return problems, nil
}
