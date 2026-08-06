package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/rcliao/comments/pkg/comment"
)

// exitError carries an explicit process exit code through the error return
// path. Handlers return it instead of calling os.Exit, so run() stays the
// single exit point and handlers remain testable.
type exitError struct {
	code int
	msg  string // printed to stderr by run(); empty = exit silently
}

func (e *exitError) Error() string {
	if e.msg != "" {
		return e.msg
	}
	return fmt.Sprintf("exit status %d", e.code)
}

// exitSilent sets the exit code without printing anything (e.g. gate's 10).
func exitSilent(code int) error {
	return &exitError{code: code}
}

// failf returns an exit-code-1 error with a formatted message.
func failf(format string, a ...any) error {
	return &exitError{code: 1, msg: fmt.Sprintf(format, a...)}
}

// run dispatches the command and converts the returned error into an exit
// code, printing error messages to stderr. It is the CLI's single exit point.
func run(args []string) int {
	err := dispatch(args)
	if err == nil {
		return 0
	}
	var ee *exitError
	if errors.As(err, &ee) {
		if ee.msg != "" {
			fmt.Fprintln(os.Stderr, ee.msg)
		}
		return ee.code
	}
	fmt.Fprintln(os.Stderr, err)
	return 1
}

// loadDocument loads a document plus its comments, printing the human-facing
// load warnings to stderr and persisting any re-anchoring/orphan-status
// migrations back to the sidecar — the side effects that used to live inside
// comment.LoadFromSidecar, now owned by the CLI layer.
func loadDocument(filename string) (*comment.DocumentWithComments, error) {
	doc, report, err := comment.LoadFromSidecar(filename)
	if err != nil {
		return nil, err
	}
	if out := comment.FormatLoadReport(report); out != "" {
		fmt.Fprint(os.Stderr, out)
	}
	if report.Dirty {
		if err := comment.SaveToSidecar(filename, doc); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to save updated sidecar: %v\n", err)
		}
	}
	return doc, nil
}
