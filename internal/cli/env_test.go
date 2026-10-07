package cli

import (
	"os"
	"testing"
)

// TestProdEnv_PasswordPromptGoesToStderr pins the production streams: the
// password prompt writes to stderr so piped stdout stays clean.
func TestProdEnv_PasswordPromptGoesToStderr(t *testing.T) {
	env := prodEnv()

	if env.Out != os.Stderr {
		t.Errorf("env.Out = %s, want os.Stderr", env.Out.Name())
	}
	if env.Stdout != os.Stdout {
		t.Errorf("env.Stdout is not os.Stdout")
	}

	prompt := terminalPasswords(env)
	if prompt.Out != os.Stderr {
		t.Errorf("password prompt writes to %s, want os.Stderr", prompt.Out.Name())
	}
	if prompt.In != os.Stdin {
		t.Errorf("password prompt reads from %s, want os.Stdin", prompt.In.Name())
	}
}
