package compositeactionlint

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	al "github.com/rhysd/actionlint"
)

type shellcheckError struct {
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Level   string `json:"level"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// RuleShellcheck checks shell scripts in "run:" steps using shellcheck.
type RuleShellcheck struct {
	RuleBase
	executablePath string
}

func NewRuleShellcheck(executablePath string) *RuleShellcheck {
	return &RuleShellcheck{
		RuleBase: RuleBase{
			name: "shellcheck",
			desc: "Checks shell scripts in \"run:\" steps using shellcheck",
		},
		executablePath: executablePath,
	}
}

func (rule *RuleShellcheck) VisitActionMetadataPre(*ActionMetadata) error  { return nil }
func (rule *RuleShellcheck) VisitActionMetadataPost(*ActionMetadata) error { return nil }

func (rule *RuleShellcheck) VisitStep(n *Step) error {
	run, ok := n.Exec.(*al.ExecRun)
	if !ok || run.Run == nil || run.Shell == nil {
		return nil
	}

	sh := normalizeShellName(run.Shell.Value)
	if sh == "" {
		return nil
	}

	setup := "set -e"
	if sh == "bash" {
		setup = "set -eo pipefail"
	}
	script := fmt.Sprintf("%s\n%s\n", setup, sanitizeExpressionsInScript(run.Run.Value))

	errs, err := invokeShellcheck(rule.executablePath, sh, script)
	if err != nil {
		return err
	}

	for _, e := range errs {
		line := e.Line - 1 // subtract the setup line we prepended
		msg := strings.TrimSuffix(e.Message, ".")
		rule.Errorf(run.RunPos, "shellcheck reported issue in this script: SC%d:%s:%d:%d: %s", e.Code, e.Level, line, e.Column, msg)
	}

	return nil
}

// normalizeShellName returns "bash" or "sh" for supported shells, and "" for
// shells that shellcheck cannot check (e.g. pwsh, python).
func normalizeShellName(shell string) string {
	switch {
	case shell == "bash" || strings.HasPrefix(shell, "bash "):
		return "bash"
	case shell == "sh" || strings.HasPrefix(shell, "sh "):
		return "sh"
	default:
		return ""
	}
}

// sanitizeExpressionsInScript replaces ${{ ... }} expressions with underscores
// of the same length so shellcheck does not see GitHub expression syntax as
// shell errors, while keeping line and column numbers intact.
func sanitizeExpressionsInScript(src string) string {
	b := strings.Builder{}
	for {
		s := strings.Index(src, "${{")
		if s == -1 {
			if b.Len() == 0 {
				return src
			}
			b.WriteString(src)
			return b.String()
		}
		e := strings.Index(src[s:], "}}")
		if e == -1 {
			if b.Len() == 0 {
				return src
			}
			b.WriteString(src)
			return b.String()
		}
		e += s + 2 // 2 is len("}}")
		b.WriteString(src[:s])
		for i := 0; i < e-s; i++ {
			b.WriteByte('_')
		}
		src = src[e:]
	}
}

func invokeShellcheck(executable, shell, script string) ([]shellcheckError, error) {
	// Reasons to exclude specific rules:
	// SC1091: File not found — scripts run in CI environments where sourced files may not exist locally.
	// SC2194: Word is constant — can occur when ${{ }} placeholders are replaced with underscores.
	// SC2050: Expression is constant — same cause as SC2194, e.g. in `if` conditions.
	// SC2154: Var referenced but not assigned — env vars injected by the runner are not visible to shellcheck.
	// SC2157: -z argument always false — occurs when ${{ }} inside [ -z ... ] is replaced with a literal string.
	// SC2043: Loop runs once — occurs when a for-loop iterates over a single placeholder underscore string.
	args := []string{
		"--norc", "-f", "json", "-x",
		"--shell", shell,
		"-e", "SC1091,SC2194,SC2050,SC2154,SC2157,SC2043",
		"-",
	}
	cmd := exec.Command(executable, args...)
	cmd.Stdin = strings.NewReader(script)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return nil, fmt.Errorf("shellcheck failed to run: %w", err)
		}
		// A non-zero exit code from shellcheck means it found issues; that is
		// expected and handled by parsing the JSON output below.
	}

	if stdout.Len() == 0 {
		return nil, nil
	}

	var errs []shellcheckError
	if err := json.Unmarshal(stdout.Bytes(), &errs); err != nil {
		return nil, fmt.Errorf("could not parse shellcheck JSON output: %w: stdout=%q", err, stdout.String())
	}
	return errs, nil
}
