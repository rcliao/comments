// Command verdict stands in for a human's review pass in the surface-parity
// driver. It is test tooling and is not part of the comments binary: the tool
// deliberately has no command that records a verdict, because a scripted
// signoff let an agent approve under a human's name. This calls the same core
// function `comments view` and `comments serve` call.
package main

import (
	"fmt"
	"os"

	"github.com/rcliao/comments/pkg/comment"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "usage: verdict <doc.md> <author> <approved|changes_requested|commented> [note]")
		os.Exit(2)
	}
	note := ""
	if len(os.Args) > 4 {
		note = os.Args[4]
	}
	doc, _, err := comment.LoadDocument(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	record, err := comment.RecordVerdict(os.Args[1], doc, os.Args[2], os.Args[3], note)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("verdict recorded: %s by @%s\n", record.Decision, record.Author)
}
