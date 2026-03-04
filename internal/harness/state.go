package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// State represents a state in the harness state machine.
type State int

const (
	StateSpecReady    State = iota // HUMAN — write the spec
	StateExecuting                 // AUTONOMOUS — agent writes code
	StateVerifying                 // AUTONOMOUS — harness runs verify cmd
	StateRetrying                  // AUTONOMOUS — harness feeds errors back
	StateReviewing                 // AUTONOMOUS — review agent checks diff
	StateFixing                    // AUTONOMOUS — implementer fixes review issues
	StateHumanReview               // HUMAN — verify in deployed env
	StateComplete                  // TERMINAL — done
	StateStuck                     // HUMAN — max retries hit
)

var stateStrings = map[State]string{
	StateSpecReady:   "spec_ready",
	StateExecuting:   "executing",
	StateVerifying:   "verifying",
	StateRetrying:    "retrying",
	StateReviewing:   "reviewing",
	StateFixing:      "fixing",
	StateHumanReview: "human_review",
	StateComplete:    "complete",
	StateStuck:       "stuck",
}

var stringToState = map[string]State{}

func init() {
	for k, v := range stateStrings {
		stringToState[v] = k
	}
}

func (s State) String() string {
	if str, ok := stateStrings[s]; ok {
		return str
	}
	return fmt.Sprintf("unknown(%d)", int(s))
}

func ParseState(s string) (State, error) {
	if st, ok := stringToState[s]; ok {
		return st, nil
	}
	return 0, fmt.Errorf("unknown state: %q", s)
}

func (s State) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *State) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	parsed, err := ParseState(str)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

// Actor indicates who owns a state.
type Actor int

const (
	ActorAutonomous Actor = iota
	ActorHuman
	ActorTerminal
)

// NeedsHuman returns true if this state requires human action.
func (s State) NeedsHuman() bool {
	switch s {
	case StateSpecReady, StateHumanReview, StateStuck:
		return true
	default:
		return false
	}
}

// StateInfo holds metadata about a state for display purposes.
type StateInfo struct {
	Actor  Actor
	Label  string
	Action string // what the human should do (empty if autonomous)
}

var StateInfos = map[State]StateInfo{
	StateSpecReady:   {ActorHuman, "Spec Ready", "Write spec and run `harness run`"},
	StateExecuting:   {ActorAutonomous, "Executing", ""},
	StateVerifying:   {ActorAutonomous, "Verifying", ""},
	StateRetrying:    {ActorAutonomous, "Retrying", ""},
	StateReviewing:   {ActorAutonomous, "Reviewing", ""},
	StateFixing:      {ActorAutonomous, "Fixing", ""},
	StateHumanReview: {ActorHuman, "Human Review", "Verify changes, then `harness approve` or `harness reject`"},
	StateComplete:    {ActorTerminal, "Complete", ""},
	StateStuck:       {ActorHuman, "Stuck", "Check .agent/ logs, revise spec or fix manually"},
}

// Condition represents a guard condition for a state transition.
type Condition int

const (
	CondAlways Condition = iota
	CondVerifyPassed
	CondVerifyFailedRetriesLeft
	CondVerifyFailedNoRetries
	CondReviewClean
	CondReviewHasIssues
	CondHumanApproves
	CondHumanRejects
	CondHumanRevisesSpec
)

// Transition describes a possible state transition.
type Transition struct {
	To   State
	When Condition
}

// Transitions defines the valid state transitions.
var Transitions = map[State][]Transition{
	StateSpecReady: {{To: StateExecuting, When: CondAlways}},
	StateExecuting: {{To: StateVerifying, When: CondAlways}},
	StateVerifying: {
		{To: StateReviewing, When: CondVerifyPassed},
		{To: StateRetrying, When: CondVerifyFailedRetriesLeft},
		{To: StateStuck, When: CondVerifyFailedNoRetries},
	},
	StateRetrying:  {{To: StateExecuting, When: CondAlways}},
	StateReviewing: {
		{To: StateHumanReview, When: CondReviewClean},
		{To: StateFixing, When: CondReviewHasIssues},
	},
	StateFixing:      {{To: StateVerifying, When: CondAlways}},
	StateHumanReview: {
		{To: StateComplete, When: CondHumanApproves},
		{To: StateSpecReady, When: CondHumanRejects},
	},
	StateComplete: {},
	StateStuck: {
		{To: StateSpecReady, When: CondHumanRevisesSpec},
	},
}

// HistoryEntry records a state transition with timestamp.
type HistoryEntry struct {
	State State     `json:"state"`
	At    time.Time `json:"at"`
}

// StateFile is the persistent state stored in .agent/state.json.
type StateFile struct {
	Task       string         `json:"task"`
	Spec       string         `json:"spec"`
	State      State          `json:"state"`
	NeedsHuman bool           `json:"needs_human"`
	Action     string         `json:"action"`
	Attempt    int            `json:"attempt"`
	MaxRetries int            `json:"max_retries"`
	CostUSD    float64        `json:"cost_usd"`
	Backend    string         `json:"backend"`
	Model      string         `json:"model"`
	StartedAt  time.Time      `json:"started_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	History    []HistoryEntry `json:"history"`

	path string // filesystem path, not serialized
}

// NewStateFile creates a new StateFile at the given path.
func NewStateFile(dir string) *StateFile {
	return &StateFile{
		path:      filepath.Join(dir, "state.json"),
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
		History:   []HistoryEntry{},
	}
}

// Persist writes the state file to disk.
func (sf *StateFile) Persist() error {
	dir := filepath.Dir(sf.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating state dir: %w", err)
	}
	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}
	return os.WriteFile(sf.path, data, 0o644)
}

// LoadStateFile reads a state file from disk.
func LoadStateFile(path string) (*StateFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading state file: %w", err)
	}
	var sf StateFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("parsing state file: %w", err)
	}
	sf.path = path
	return &sf, nil
}

// Path returns the filesystem path of the state file.
func (sf *StateFile) Path() string {
	return sf.path
}
