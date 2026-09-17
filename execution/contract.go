// Package execution defines the wire contract shared by trilha-spec,
// trilha-runner and trilha-cloud for one logical execution.
package execution

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const Version = "trilha.execution/v1"

type Status string

const (
	Queued    Status = "queued"
	Running   Status = "running"
	Review    Status = "review"
	Failed    Status = "failed"
	Done      Status = "done"
	Cancelled Status = "cancelled"
)

var Statuses = []Status{Queued, Running, Review, Failed, Done, Cancelled}

func (s Status) Valid() bool {
	for _, candidate := range Statuses {
		if s == candidate {
			return true
		}
	}
	return false
}

type Phase string

const (
	PhaseQueued           Phase = "queued"
	PhasePreparingContext Phase = "preparing_context"
	PhaseCreatingWorktree Phase = "creating_worktree"
	PhaseConsultingAI     Phase = "consulting_ai"
	PhaseChangingFiles    Phase = "changing_files"
	PhaseRunningChecks    Phase = "running_checks"
	PhaseCreatingPreview  Phase = "creating_preview"
	PhaseAwaitingReview   Phase = "awaiting_review"
	PhaseRetryWait        Phase = "retry_wait"
	PhaseCancelling       Phase = "cancelling"
	PhaseCancelled        Phase = "cancelled"
	PhaseFailed           Phase = "failed"
	PhaseDone             Phase = "done"
)

var Phases = []Phase{
	PhaseQueued, PhasePreparingContext, PhaseCreatingWorktree, PhaseConsultingAI,
	PhaseChangingFiles, PhaseRunningChecks, PhaseCreatingPreview, PhaseAwaitingReview,
	PhaseRetryWait, PhaseCancelling, PhaseCancelled, PhaseFailed, PhaseDone,
}

func (p Phase) Valid() bool {
	for _, candidate := range Phases {
		if p == candidate {
			return true
		}
	}
	return false
}

type Event struct {
	Sequence int       `json:"sequence"`
	At       time.Time `json:"at"`
	Phase    Phase     `json:"phase"`
	Message  string    `json:"message"`
	Attempt  int       `json:"attempt,omitempty"`
}

type Attempt struct {
	Number        int     `json:"number"`
	Status        string  `json:"status"`
	FailureClass  string  `json:"failure_class,omitempty"`
	ErrorCode     string  `json:"error_code,omitempty"`
	Message       string  `json:"message,omitempty"`
	RepairReason  string  `json:"repair_reason,omitempty"`
	Model         string  `json:"model,omitempty"`
	TotalTokens   int     `json:"total_tokens,omitempty"`
	EstimatedCost float64 `json:"estimated_cost,omitempty"`
	// Cost as observed, with the names a `run` evidence record uses (§5),
	// so a ledger reads the same fields from a runner and from .trilha/.
	Provider  string    `json:"provider,omitempty"`
	TokensIn  int       `json:"tokens_in,omitempty"`
	TokensOut int       `json:"tokens_out,omitempty"`
	Cost      float64   `json:"cost,omitempty"`
	Currency  string    `json:"currency,omitempty"`
	Started   time.Time `json:"started"`
	Finished  time.Time `json:"finished,omitempty"`
}

type Evidence struct {
	Task       string            `json:"task"`
	Kind       string            `json:"kind"`
	By         string            `json:"by,omitempty"`
	Command    string            `json:"command,omitempty"`
	ExitCode   int               `json:"exit_code,omitempty"`
	Passed     bool              `json:"passed"`
	Files      []string          `json:"files,omitempty"`
	Provider   string            `json:"provider,omitempty"`
	Model      string            `json:"model,omitempty"`
	TokensIn   int               `json:"tokens_in,omitempty"`
	TokensOut  int               `json:"tokens_out,omitempty"`
	Cost       float64           `json:"cost,omitempty"`
	Currency   string            `json:"currency,omitempty"`
	Meta       map[string]string `json:"meta,omitempty"`
	OutputHash string            `json:"output_sha256,omitempty"`
}

type Outcome struct {
	Passed           bool       `json:"passed"`
	Status           Status     `json:"status"`
	Branch           string     `json:"branch,omitempty"`
	Commit           string     `json:"commit,omitempty"`
	Evidence         []Evidence `json:"evidence,omitempty"`
	Log              string     `json:"log,omitempty"`
	Error            string     `json:"error,omitempty"`
	ErrorCode        string     `json:"error_code,omitempty"`
	TechnicalDetails string     `json:"technical_details,omitempty"`
	FailureClass     string     `json:"failure_class,omitempty"`
	Attempt          int        `json:"attempt,omitempty"`
	RetryOf          string     `json:"retry_of,omitempty"`
	RepairReason     string     `json:"repair_reason,omitempty"`
	Model            string     `json:"model,omitempty"`
	InputTokens      int        `json:"input_tokens,omitempty"`
	OutputTokens     int        `json:"output_tokens,omitempty"`
	TotalTokens      int        `json:"total_tokens,omitempty"`
	EstimatedCost    float64    `json:"estimated_cost,omitempty"`
	DurationMS       int64      `json:"duration_ms,omitempty"`
	FilesChanged     []string   `json:"files_changed,omitempty"`
	DiffSummary      string     `json:"diff_summary,omitempty"`
	AgentReport      string     `json:"agent_report,omitempty"`
}

type Run struct {
	ProtocolVersion string    `json:"protocol_version"`
	ID              string    `json:"id"`
	Project         string    `json:"project"`
	TaskID          string    `json:"task_id"`
	Status          Status    `json:"status"`
	Phase           Phase     `json:"phase,omitempty"`
	Progress        int       `json:"progress,omitempty"`
	Attempt         int       `json:"attempt,omitempty"`
	MaxAttempts     int       `json:"max_attempts,omitempty"`
	RetryOf         string    `json:"retry_of,omitempty"`
	RepairReason    string    `json:"repair_reason,omitempty"`
	CancelRequested bool      `json:"cancel_requested,omitempty"`
	Worker          string    `json:"worker,omitempty"`
	Created         time.Time `json:"created"`
	Started         time.Time `json:"started,omitempty"`
	Finished        time.Time `json:"finished,omitempty"`
	Events          []Event   `json:"events,omitempty"`
	Attempts        []Attempt `json:"attempts,omitempty"`
	Result          *Outcome  `json:"result,omitempty"`
}

var runIDPattern = regexp.MustCompile(`^run-[0-9]{6,}$`)
var taskIDPattern = regexp.MustCompile(`^TASK-[0-9]{3,}$`)

func ValidRunID(id string) bool { return runIDPattern.MatchString(id) }

func (r Run) Validate() error {
	var problems []string
	if r.ProtocolVersion != Version {
		problems = append(problems, fmt.Sprintf("protocol_version must be %q", Version))
	}
	if !ValidRunID(r.ID) {
		problems = append(problems, "id must look like run-000001")
	}
	if strings.TrimSpace(r.Project) == "" {
		problems = append(problems, "project is required")
	}
	if !taskIDPattern.MatchString(r.TaskID) {
		problems = append(problems, "task_id must look like TASK-001")
	}
	if !r.Status.Valid() {
		problems = append(problems, "status is invalid")
	}
	if r.Phase != "" && !r.Phase.Valid() {
		problems = append(problems, "phase is invalid")
	}
	if r.Progress < 0 || r.Progress > 100 {
		problems = append(problems, "progress must be between 0 and 100")
	}
	if r.RetryOf != "" && !ValidRunID(r.RetryOf) {
		problems = append(problems, "retry_of must be a run id")
	}
	if len(problems) > 0 {
		return errors.New("execution " + r.ID + ": " + strings.Join(problems, "; "))
	}
	return nil
}
