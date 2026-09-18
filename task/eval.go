package task

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/emersonjoe/trilha-spec/spec"
)

// KindEval is the evidence kind that carries a number: what a harness
// measured, against what it had to beat. A `check` hides the value in an exit
// code, so nobody can see it, trend it or gate on it; an `eval` shows it.
const KindEval = "eval"

// The comparators an eval may use. There are three on purpose: a gate is
// "at least", "at most" or "exactly", and anything subtler is a statistical
// test, which is not the protocol's business.
const (
	AtLeast = ">="
	AtMost  = "<="
	Exactly = "=="
)

// Comparators lists them for a reader that validates.
var Comparators = []string{AtLeast, AtMost, Exactly}

// Dataset names the set a metric was measured on. It carries the hash of the
// set's **manifest**, never its content: a golden set holds real cases, and
// real cases hold personal data.
type Dataset struct {
	ID     string `json:"id"`
	SHA256 string `json:"sha256,omitempty"`
}

var reMetric = regexp.MustCompile(`^[a-z0-9]+([._-][a-z0-9]+)*$`)

// ValidMetric answers whether name can be a metric: lowercase words joined by
// `_`, `-` or `.`, so two harnesses that measure the same thing agree on the
// name and a ledger can group by it.
func ValidMetric(name string) bool { return reMetric.MatchString(name) }

// ValidComparator answers whether c is one of the three.
func ValidComparator(c string) bool {
	for _, x := range Comparators {
		if x == c {
			return true
		}
	}
	return false
}

// Compare answers whether value passes threshold under comparator.
func Compare(value float64, comparator string, threshold float64) bool {
	switch comparator {
	case AtLeast:
		return value >= threshold
	case AtMost:
		return value <= threshold
	case Exactly:
		return value == threshold
	}
	return false
}

// validateEval checks an eval record: a metric name, a value, and a threshold
// that comes with the comparator that reads it. A threshold without a
// comparator would have to be guessed, and a guessed gate is a wrong gate.
func (e Evidence) validateEval() error {
	if e.Kind != KindEval {
		return nil
	}
	switch {
	case !ValidMetric(e.Metric):
		return fmt.Errorf("evidence: %q is not a metric name", e.Metric)
	case e.Value == nil:
		return fmt.Errorf("evidence: metric %s has no value", e.Metric)
	case e.Threshold != nil && !ValidComparator(e.Comparator):
		return fmt.Errorf("evidence: metric %s has a threshold but no comparator (%s)", e.Metric, strings.Join(Comparators, ", "))
	case e.Comparator != "" && !ValidComparator(e.Comparator):
		return fmt.Errorf("evidence: comparator %q is not one of %s", e.Comparator, strings.Join(Comparators, ", "))
	case e.Dataset != nil && strings.TrimSpace(e.Dataset.ID) == "":
		return fmt.Errorf("evidence: metric %s has a dataset with no id", e.Metric)
	}
	return nil
}

// Eval builds an eval record and settles whether it passed. A metric with no
// threshold is a measurement, not a gate: it passes, and the number is there
// to be trended.
func Eval(taskID, by string, m Metric) Evidence {
	e := Evidence{Task: taskID, Kind: KindEval, By: by, Metric: m.Metric, Unit: m.Unit,
		Value: &m.Value, Comparator: m.Comparator, Dataset: m.Dataset, Passed: true}
	if m.Threshold != nil {
		e.Threshold = m.Threshold
		e.Passed = Compare(m.Value, m.Comparator, *m.Threshold)
	}
	return e
}

// Metric is one measurement, as a harness prints it and as the context pack
// reads it back.
type Metric struct {
	Metric     string   `json:"metric"`
	Value      float64  `json:"value"`
	Unit       string   `json:"unit,omitempty"`
	Threshold  *float64 `json:"threshold,omitempty"`
	Comparator string   `json:"comparator,omitempty"`
	Dataset    *Dataset `json:"dataset,omitempty"`
	// Passed and Seq are filled when the metric is read back from evidence.
	Passed bool `json:"passed"`
	Seq    int  `json:"seq,omitempty"`
}

// ParseMetricLine reads one line a check printed. A metric line is a JSON
// object with a `metric` and a `value`; anything else is just output and is
// left alone, so a check that prints plain text is unaffected.
//
//	{"metric":"triage_top1","value":0.87,"threshold":0.85,"comparator":">="}
//
// The second answer is the reason the line cannot be used as a gate, when
// there is one: the caller records the number anyway and says why it did not
// gate, rather than guessing a comparator.
func ParseMetricLine(line string) (Metric, string, bool) {
	t := strings.TrimSpace(line)
	if !strings.HasPrefix(t, "{") || !strings.HasSuffix(t, "}") {
		return Metric{}, "", false
	}
	var raw struct {
		Metric     string   `json:"metric"`
		Value      *float64 `json:"value"`
		Unit       string   `json:"unit"`
		Threshold  *float64 `json:"threshold"`
		Comparator string   `json:"comparator"`
		Dataset    *Dataset `json:"dataset"`
	}
	if err := json.Unmarshal([]byte(t), &raw); err != nil || raw.Metric == "" || raw.Value == nil {
		return Metric{}, "", false
	}
	m := Metric{Metric: raw.Metric, Value: *raw.Value, Unit: raw.Unit, Threshold: raw.Threshold, Comparator: raw.Comparator, Dataset: raw.Dataset}
	switch {
	case !ValidMetric(m.Metric):
		return m, "metric name " + strconv.Quote(m.Metric) + " is not lowercase words joined by _, - or .", true
	case m.Threshold != nil && !ValidComparator(m.Comparator):
		m.Threshold = nil
		return m, "threshold without a comparator (" + strings.Join(Comparators, ", ") + "): recorded as a measurement, not a gate", true
	case m.Comparator != "" && !ValidComparator(m.Comparator):
		return m, "comparator " + strconv.Quote(m.Comparator) + " is not one of " + strings.Join(Comparators, ", "), true
	}
	return m, "", true
}

// Metrics answers the last value of every metric a task has evidence for,
// sorted by name: what a reviewer, a context pack and a control plane read
// to know where the numbers stand right now.
func Metrics(list []Evidence) []Metric {
	last := map[string]Metric{}
	for _, e := range list {
		if e.Kind != KindEval || e.Metric == "" || e.Value == nil {
			continue
		}
		last[e.Metric] = Metric{Metric: e.Metric, Value: *e.Value, Unit: e.Unit,
			Threshold: e.Threshold, Comparator: e.Comparator, Dataset: e.Dataset, Passed: e.Passed, Seq: e.Seq}
	}
	out := make([]Metric, 0, len(last))
	for _, name := range spec.SortedKeys(last) {
		out = append(out, last[name])
	}
	return out
}

// String renders a metric the way the CLI and the context pack show it:
// `triage_top1 0.87 >= 0.85 ✓`.
func (m Metric) String() string {
	s := m.Metric + " " + num(m.Value)
	if m.Unit != "" {
		s += " " + m.Unit
	}
	if m.Threshold != nil {
		s += " " + m.Comparator + " " + num(*m.Threshold)
	}
	if m.Dataset != nil {
		s += " on " + m.Dataset.ID
	}
	return s
}

func num(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

var reAcceptanceMetric = regexp.MustCompile(`^metric:\s*(\S+)\s*(>=|<=|==)\s*(\S+)$`)

// AcceptanceMetric is an acceptance criterion written as a gate:
//
//	acceptance:
//	  - "metric: triage_top1 >= 0.85"
//
// It is still a criterion in words for a reviewer; it is also a name the
// evidence has to answer, which is what doctor checks.
type AcceptanceMetric struct {
	Metric     string  `json:"metric"`
	Comparator string  `json:"comparator"`
	Threshold  float64 `json:"threshold"`
}

// ParseAcceptanceMetric reads the metric form of an acceptance criterion.
// Anything else is prose and answers false.
func ParseAcceptanceMetric(s string) (AcceptanceMetric, bool) {
	m := reAcceptanceMetric.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil || !ValidMetric(m[1]) {
		return AcceptanceMetric{}, false
	}
	threshold, err := strconv.ParseFloat(m[3], 64)
	if err != nil {
		return AcceptanceMetric{}, false
	}
	return AcceptanceMetric{Metric: m[1], Comparator: m[2], Threshold: threshold}, true
}

// AcceptanceMetrics answers the gates a task declares, in the order they are
// written.
func (t *Task) AcceptanceMetrics() []AcceptanceMetric {
	var out []AcceptanceMetric
	for _, a := range t.Acceptance {
		if m, ok := ParseAcceptanceMetric(a); ok {
			out = append(out, m)
		}
	}
	return out
}

// CheckMetrics reports an acceptance metric a task never evidenced. It only
// looks at tasks a reviewer is about to close or has closed: a metric on an
// `idea` has simply not been measured yet, and saying so every day is noise.
func CheckMetrics(l spec.Layout, tasks []*Task) ([]spec.Problem, error) {
	var problems []spec.Problem
	for _, t := range tasks {
		gates := t.AcceptanceMetrics()
		if len(gates) == 0 || (t.Status != Review && t.Status != Done) {
			continue
		}
		list, err := ListEvidence(l, t.ID)
		if err != nil {
			return nil, err
		}
		seen := map[string]bool{}
		for _, e := range list {
			if e.Kind == KindEval {
				seen[e.Metric] = true
			}
		}
		var missing []string
		for _, g := range gates {
			if !seen[g.Metric] {
				missing = append(missing, g.Metric)
			}
		}
		sort.Strings(missing)
		for _, name := range missing {
			problems = append(problems, spec.Problem{Code: spec.ProblemMetricNotEvidenced, Arg: t.ID + " " + name})
		}
	}
	return problems, nil
}
