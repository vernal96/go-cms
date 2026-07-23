package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vernal96/go-cms/kernel/console"
)

func TestPasswordIsReadOnlyFromStdin(t *testing.T) {
	t.Parallel()

	password, err := readPassword(
		strings.NewReader("  password with spaces  \r\n"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if password != "  password with spaces  " {
		t.Fatalf("password = %q", password)
	}

	var flagErrors bytes.Buffer
	err = (&usersCommand{}).Run(
		t.Context(),
		[]string{
			"create",
			"-login=admin",
			"-email=admin@example.test",
			"-name=Admin",
			"-password=must-not-be-supported",
		},
		console.IO{
			In:  strings.NewReader("stdin-password\n"),
			Out: &bytes.Buffer{},
			Err: &flagErrors,
		},
	)
	if err == nil ||
		!strings.Contains(flagErrors.String(), "flag provided but not defined") {
		t.Fatalf("password flag error = %v, stderr = %q", err, flagErrors.String())
	}
}

func TestPasswordRejectsEmptyStdin(t *testing.T) {
	t.Parallel()

	if _, err := readPassword(strings.NewReader("\n")); err == nil {
		t.Fatal("empty password stdin was accepted")
	}
}
