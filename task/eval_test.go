package task

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emersonjoe/trilha-spec/spec"
)

func TestEvalRecordAndGate(t *testing.T) {
	st := newStore(t)
	l := st.Layout
	th := 0.85
	e, path, err := Record(l, Eval("TASK-001", "harness", Metric{Metric: "triage_top1", Value: 0.87, Threshold: &th, Comparator: AtLeast, Dataset: &Dataset{ID: "golden-2026", SHA256: "abc"}}))
	if err != nil || !e.Passed || *e.Value != 0.87 {
		t.Fatalf("eval: %v %+v", err, e)
	}
	if !strings.HasSuffix(filepath.ToSlash(path), "001-eval.json") {
		t.Fatalf("path = %s", path)
	}
	raw, _ := os.ReadFile(path)
	for _, want := range []string{`"kind": "eval"`, `"metric": "triage_top1"`, `"value": 0.87`, `"threshold": 0.85`, `"comparator": ">="`, `"id": "golden-2026"`, `"passed": true`} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("record missing %s:\n%s", want, raw)
		}
	}
	// Zero is a number a gate cares about: it must survive the round trip.
	zero := 0.0
	e, _, err = Record(l, Eval("TASK-001", "harness", Metric{Metric: "wcag_violations", Value: 0, Threshold: &zero, Comparator: Exactly}))
	if err != nil || !e.Passed || e.Value == nil || *e.Value != 0 {
		t.Fatalf("zero value: %v %+v", err, e)
	}
	// A value that misses the threshold does not pass.
	e, _, err = Record(l, Eval("TASK-001", "harness", Metric{Metric: "p95_latency", Value: 7, Unit: "s", Threshold: ptr(5), Comparator: AtMost}))
	if err != nil || e.Passed {
		t.Fatalf("failing gate passed: %v %+v", err, e)
	}
	// No threshold is a measurement, not a gate.
	e, _, err = Record(l, Eval("TASK-001", "harness", Metric{Metric: "answers", Value: 120}))
	if err != nil || !e.Passed || e.Threshold != nil {
		t.Fatalf("measurement: %v %+v", err, e)
	}
	// What the reader refuses.
	for _, bad := range []Evidence{
		{Task: "TASK-001", Kind: KindEval, Metric: "Triage Top1", Value: ptr(1)},
		{Task: "TASK-001", Kind: KindEval, Metric: "triage_top1"},
		{Task: "TASK-001", Kind: KindEval, Metric: "triage_top1", Value: ptr(1), Threshold: ptr(1)},
		{Task: "TASK-001", Kind: KindEval, Metric: "triage_top1", Value: ptr(1), Comparator: ">"},
	} {
		if _, _, err := Record(l, bad); err == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
	// The last value per metric, sorted by name.
	list, err := ListEvidence(l, "TASK-001")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, m := range Metrics(list) {
		got = append(got, m.String())
	}
	want := "answers 120|p95_latency 7 s <= 5|triage_top1 0.87 >= 0.85 on golden-2026|wcag_violations 0 == 0"
	if strings.Join(got, "|") != want {
		t.Fatalf("metrics = %q, wanted %q", strings.Join(got, "|"), want)
	}
}

func ptr(f float64) *float64 { return &f }

func TestParseMetricLine(t *testing.T) {
	m, why, ok := ParseMetricLine(`  {"metric":"triage_top1","value":0.87,"threshold":0.85,"comparator":">="}  `)
	if !ok || why != "" || m.Metric != "triage_top1" || m.Value != 0.87 || *m.Threshold != 0.85 {
		t.Fatalf("metric line: %+v %q %v", m, why, ok)
	}
	// Plain output is plain output.
	for _, line := range []string{"ok  	app	0.4s", "", "{not json}", `{"value":1}`, `{"metric":"x"}`} {
		if _, _, ok := ParseMetricLine(line); ok {
			t.Fatalf("%q read as a metric", line)
		}
	}
	// A threshold nobody said how to read is recorded as a measurement, never
	// as a guessed gate.
	m, why, ok = ParseMetricLine(`{"metric":"p95_latency","value":7,"threshold":5}`)
	if !ok || m.Threshold != nil || !strings.Contains(why, "without a comparator") {
		t.Fatalf("guessed a comparator: %+v %q", m, why)
	}
	if _, why, _ := ParseMetricLine(`{"metric":"P95","value":7}`); !strings.Contains(why, "not lowercase words") {
		t.Fatalf("bad name accepted: %q", why)
	}
}

// needs skips a test when the program it prints its output with is not on
// this machine: the metric lines matter, the tool that prints them does not.
func needs(t *testing.T, program string) {
	t.Helper()
	if _, err := exec.LookPath(program); err != nil {
		t.Skipf("%s is not on PATH", program)
	}
}

// printing writes a file the check will print, so the metric lines reach
// stdout without a shell quoting them into something else.
func printing(t *testing.T, l spec.Layout, name string, lines ...string) string {
	t.Helper()
	p := filepath.Join(l.Root, name)
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return "cat " + name
}

func TestVerifyRecordsMetrics(t *testing.T) {
	needs(t, "cat")
	st := newStore(t)
	l := st.Layout
	tk, err := st.Create("Triage", func(x *Task) {
		x.Status = Verify
		x.Acceptance = []string{"metric: triage_top1 >= 0.85", "the reviewer agrees"}
		x.Checks = []string{printing(t, l, "metrics.txt",
			"running",
			`{"metric":"triage_top1","value":0.87,"threshold":0.85,"comparator":">="}`)}
	})
	if err != nil {
		t.Fatal(err)
	}
	v, err := RunChecks(context.Background(), l, tk, l.Root, "harness")
	if err != nil || !v.Passed {
		t.Fatalf("verify: %v %+v", err, v)
	}
	if len(v.Evidence) != 2 || v.Evidence[0].Kind != "check" || v.Evidence[1].Kind != KindEval {
		t.Fatalf("records = %+v", v.Evidence)
	}
	if v.Evidence[1].Command == "" || !v.Evidence[1].Passed {
		t.Fatalf("eval record = %+v", v.Evidence[1])
	}
	// A metric below its threshold fails the verification even when the
	// command exits 0: that is what a quality gate is for.
	tk2, err := st.Create("Latency", func(x *Task) {
		x.Status = Verify
		x.Acceptance = []string{"metric: p95_latency <= 5"}
		x.Checks = []string{printing(t, l, "slow.txt",
			`{"metric":"p95_latency","value":7,"unit":"s","threshold":5,"comparator":"<="}`)}
	})
	if err != nil {
		t.Fatal(err)
	}
	v, err = RunChecks(context.Background(), l, tk2, l.Root, "harness")
	if err != nil || v.Passed {
		t.Fatalf("a failing metric must fail verify: %v %+v", err, v)
	}
}

func TestAcceptanceMetricsAndDoctor(t *testing.T) {
	st := newStore(t)
	l := st.Layout
	m, ok := ParseAcceptanceMetric(" metric: triage_top1 >= 0.85 ")
	if !ok || m.Metric != "triage_top1" || m.Comparator != AtLeast || m.Threshold != 0.85 {
		t.Fatalf("acceptance metric: %+v %v", m, ok)
	}
	for _, prose := range []string{"the reviewer agrees", "metric: triage top1 >= 0.85", "metric: x ~ 1", "metric: x >= many"} {
		if _, ok := ParseAcceptanceMetric(prose); ok {
			t.Fatalf("%q read as a gate", prose)
		}
	}
	tk, err := st.Create("Triage", func(x *Task) {
		x.Status = Review
		x.Acceptance = []string{"metric: triage_top1 >= 0.85", "metric: latency_p95 <= 5", "the reviewer agrees"}
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := tk.AcceptanceMetrics(); len(got) != 2 {
		t.Fatalf("gates = %+v", got)
	}
	if _, _, err := Record(l, Eval(tk.ID, "harness", Metric{Metric: "triage_top1", Value: 0.9, Threshold: ptr(0.85), Comparator: AtLeast})); err != nil {
		t.Fatal(err)
	}
	tasks, err := st.List()
	if err != nil {
		t.Fatal(err)
	}
	problems, err := CheckMetrics(l, tasks)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 1 || problems[0].Code != spec.ProblemMetricNotEvidenced || problems[0].Arg != tk.ID+" latency_p95" {
		t.Fatalf("doctor = %+v", problems)
	}
	// A task nobody is closing yet is not nagged about numbers.
	tasks[0].Status = Ready
	if problems, err := CheckMetrics(l, tasks); err != nil || len(problems) != 0 {
		t.Fatalf("early task reported: %v %+v", err, problems)
	}
}

// Signing does not care what a record holds, and an eval is a record: the
// new fields go into the canonical form like any other.
func TestSignedEvalRecord(t *testing.T) {
	st := newStore(t)
	l := st.Layout
	privPEM, pubPEM, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	priv, err := ParsePrivateKey(privPEM)
	if err != nil {
		t.Fatal(err)
	}
	pub, err := ParsePublicKey(pubPEM)
	if err != nil {
		t.Fatal(err)
	}
	e, path, err := RecordSigned(l, Eval("TASK-001", "runner-01", Metric{Metric: "triage_top1", Value: 0.87, Threshold: ptr(0.85), Comparator: AtLeast}), &Signer{KeyID: "runner-01", Key: priv})
	if err != nil || e.Signature == nil {
		t.Fatalf("signed eval: %v %+v", err, e)
	}
	keys := Keyring{"runner-01": pub}
	if v, why := keys.Check(e); v != Valid {
		t.Fatalf("verdict = %s (%s)", v, why)
	}
	// Change the number on disk and the signature no longer covers it.
	raw, _ := os.ReadFile(path)
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), "0.87", "0.97", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	list, err := ListEvidence(l, "TASK-001")
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := keys.Check(list[0]); v != Invalid {
		t.Fatalf("an edited metric verified as %s", v)
	}
}
