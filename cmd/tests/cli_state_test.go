package tests

import (
	"errors"
	"os"
	"runtime"
	"testing"

	"github.com/spf13/pflag"
	"github.com/yepizrene-devoost/dflow/cmd/commands"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

type cliFlagState struct {
	flag    *pflag.Flag
	value   string
	changed bool
}

// withCLIState snapshots the process-wide state used by direct command tests.
// Callers defer the returned restore function so cleanup also runs when a test
// exits through testing.T.Fatal or another runtime.Goexit path.
func withCLIState(t *testing.T) func() {
	t.Helper()

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("snapshot working directory: %v", err)
	}

	stdout := os.Stdout
	format := utils.CurrentFormat()
	var flags []cliFlagState
	commands.StartCmd.Flags().VisitAll(func(flag *pflag.Flag) {
		flags = append(flags, cliFlagState{
			flag:    flag,
			value:   flag.Value.String(),
			changed: flag.Changed,
		})
	})

	return func() {
		if err := os.Chdir(workingDir); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
		os.Stdout = stdout
		utils.SetFormat(format)
		for _, state := range flags {
			if err := state.flag.Value.Set(state.value); err != nil {
				t.Errorf("restore --%s value: %v", state.flag.Name, err)
			}
			state.flag.Changed = state.changed
		}
	}
}

func TestWithCLIStateRestoresProcessStateAfterErrorAndEarlyGoexit(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get original working directory: %v", err)
	}
	originalStdout := os.Stdout
	originalFormat := utils.CurrentFormat()
	fromFlag := commands.StartCmd.Flags().Lookup("from")
	if fromFlag == nil {
		t.Fatal("StartCmd --from flag is not registered")
	}
	pushFlag := commands.StartCmd.Flags().Lookup("push")
	if pushFlag == nil {
		t.Fatal("StartCmd --push flag is not registered")
	}
	originalFromValue := fromFlag.Value.String()
	originalFromChanged := fromFlag.Changed
	originalPushValue := pushFlag.Value.String()
	originalPushChanged := pushFlag.Changed

	t.Cleanup(func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("restore original working directory: %v", err)
		}
		os.Stdout = originalStdout
		utils.SetFormat(originalFormat)
		if err := fromFlag.Value.Set(originalFromValue); err != nil {
			t.Errorf("restore original --from value: %v", err)
		}
		fromFlag.Changed = originalFromChanged
		if err := pushFlag.Value.Set(originalPushValue); err != nil {
			t.Errorf("restore original --push value: %v", err)
		}
		pushFlag.Changed = originalPushChanged
	})

	utils.SetFormat(utils.FormatHuman)
	if err := fromFlag.Value.Set("feature/original"); err != nil {
		t.Fatalf("set baseline --from value: %v", err)
	}
	fromFlag.Changed = false
	if err := pushFlag.Value.Set("true"); err != nil {
		t.Fatalf("set baseline --push value: %v", err)
	}
	pushFlag.Changed = true

	assertBaseline := func() {
		t.Helper()

		gotDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("get restored working directory: %v", err)
		}
		if gotDir != originalDir {
			t.Errorf("working directory after restore = %q, want %q", gotDir, originalDir)
		}
		if os.Stdout != originalStdout {
			t.Errorf("stdout after restore = %p, want %p", os.Stdout, originalStdout)
		}
		if got := utils.CurrentFormat(); got != utils.FormatHuman {
			t.Errorf("output format after restore = %v, want %v", got, utils.FormatHuman)
		}
		if got := fromFlag.Value.String(); got != "feature/original" {
			t.Errorf("--from value after restore = %q, want %q", got, "feature/original")
		}
		if fromFlag.Changed {
			t.Error("--from Changed after restore = true, want false")
		}
		if got := pushFlag.Value.String(); got != "true" {
			t.Errorf("--push value after restore = %q, want %q", got, "true")
		}
		if !pushFlag.Changed {
			t.Error("--push Changed after restore = false, want true")
		}
	}

	mutateState := func(stdout *os.File) {
		t.Helper()

		if err := os.Chdir(t.TempDir()); err != nil {
			t.Fatalf("change working directory: %v", err)
		}
		os.Stdout = stdout
		utils.SetFormat(utils.FormatJSON)
		if err := commands.StartCmd.Flags().Set("from", "feature/mutated"); err != nil {
			t.Fatalf("mutate --from: %v", err)
		}
		if err := commands.StartCmd.Flags().Set("push", "false"); err != nil {
			t.Fatalf("mutate --push: %v", err)
		}
		pushFlag.Changed = false
	}

	errorWriter, err := os.CreateTemp(t.TempDir(), "error-stdout")
	if err != nil {
		t.Fatalf("create stdout file for error case: %v", err)
	}
	defer errorWriter.Close()

	sentinel := errors.New("sentinel")
	if err := func() error {
		defer withCLIState(t)()
		mutateState(errorWriter)
		return sentinel
	}(); !errors.Is(err, sentinel) {
		t.Fatalf("guarded function error = %v, want %v", err, sentinel)
	}
	assertBaseline()

	exitWriter, err := os.CreateTemp(t.TempDir(), "exit-stdout")
	if err != nil {
		t.Fatalf("create stdout file for early-exit case: %v", err)
	}
	defer exitWriter.Close()

	exited := make(chan struct{})
	go func() {
		defer close(exited)
		defer withCLIState(t)()
		mutateState(exitWriter)
		runtime.Goexit()
	}()
	<-exited
	assertBaseline()
}
