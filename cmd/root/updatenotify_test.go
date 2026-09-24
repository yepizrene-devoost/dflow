package root

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/yepizrene-devoost/dflow/cmd/selfupdate"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

// releaseVersion is a marker with release provenance, so the eligibility and
// render tests exercise the version comparison instead of the provenance gate.
const releaseVersion = "v0.1.0"

// resetUpdateNoticeState isolates one test from the process-wide notification
// state: the probe slot, the two guards, the environment seam, the version
// marker and the output format. Each is restored on cleanup so a failure in one
// test cannot leak into the next.
func resetUpdateNoticeState(t *testing.T) {
	t.Helper()

	previousLookup := updateNoticeLookupEnv
	previousVersion := utils.VersionMarker()
	previousFormat := utils.CurrentFormat()

	t.Cleanup(func() {
		updateNoticeSlot = make(chan *selfupdate.Release, 1)
		updateProbeArmed.Store(false)
		updateNoticeRendered.Store(false)
		updateNoticeLookupEnv = previousLookup
		utils.SetVersion(previousVersion)
		utils.SetFormat(previousFormat)
	})

	updateNoticeSlot = make(chan *selfupdate.Release, 1)
	updateProbeArmed.Store(false)
	updateNoticeRendered.Store(false)
	updateNoticeLookupEnv = func(string) string { return "" }
	utils.SetVersion(releaseVersion)
	utils.SetFormat(utils.FormatHuman)
}

// withUpdateNoticeArgs pins os.Args for the duration of one test, so the
// banner-skip gate reads a controlled argument vector instead of the flags the
// test binary happens to be running with.
func withUpdateNoticeArgs(t *testing.T, args ...string) {
	t.Helper()
	previous := os.Args
	os.Args = args
	t.Cleanup(func() { os.Args = previous })
}

// captureUpdateNoticeStderr runs fn with os.Stderr redirected into a pipe and
// returns everything written to it. Notice is the only writer under test, so
// the swap is deterministic; the helper exists because the notice's whole
// contract is "one line on stderr".
func captureUpdateNoticeStderr(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create the stderr pipe: %v", err)
	}
	os.Stderr = writer
	captured := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(reader)
		captured <- string(data)
	}()

	fn()

	os.Stderr = original
	if err := writer.Close(); err != nil {
		t.Fatalf("close the stderr pipe: %v", err)
	}
	got := <-captured
	_ = reader.Close()
	return got
}

// TestUpdateProbeEligibleGates pins every suppression decision in one place.
// Each case changes exactly one input and asserts the gate flips, so a
// regression names the reason it broke instead of only the fact.
func TestUpdateProbeEligibleGates(t *testing.T) {
	statusCmd := &cobra.Command{Use: "status"}
	updateCmd := &cobra.Command{Use: "update"}

	cases := []struct {
		name    string
		command *cobra.Command
		prepare func()
		want    bool
	}{
		{
			name:    "plain human invocation is eligible",
			command: statusCmd,
			want:    true,
		},
		{
			name:    "json format suppresses",
			command: statusCmd,
			prepare: func() { utils.SetFormat(utils.FormatJSON) },
		},
		{
			name:    "a non-empty DFLOW_NO_UPDATE_CHECK suppresses",
			command: statusCmd,
			prepare: func() { updateNoticeLookupEnv = func(string) string { return "1" } },
		},
		{
			name:    "a whitespace-only DFLOW_NO_UPDATE_CHECK does not suppress",
			command: statusCmd,
			prepare: func() { updateNoticeLookupEnv = func(string) string { return "   " } },
			want:    true,
		},
		{
			name:    "the update command suppresses",
			command: updateCmd,
		},
		{
			name:    "a build without release provenance suppresses",
			command: statusCmd,
			prepare: func() { utils.SetVersion("dev") },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetUpdateNoticeState(t)
			withUpdateNoticeArgs(t, "dflow", "status")
			if tc.prepare != nil {
				tc.prepare()
			}

			if got := updateProbeEligible(tc.command); got != tc.want {
				t.Fatalf("updateProbeEligible = %v, want %v", got, tc.want)
			}
		})
	}

	t.Run("an invocation the banner skips suppresses", func(t *testing.T) {
		resetUpdateNoticeState(t)
		withUpdateNoticeArgs(t, "dflow", "--help")

		if updateProbeEligible(statusCmd) {
			t.Fatalf("help has no output moment to attach a notice to, so it must not probe")
		}
	})
}

// TestRenderPendingUpdateNotice covers the decision the renderer makes: print
// the exact one-line notice for a newer release, stay silent otherwise, and
// consume the result exactly once.
func TestRenderPendingUpdateNotice(t *testing.T) {
	t.Run("renders the exact line for a newer release", func(t *testing.T) {
		resetUpdateNoticeState(t)
		updateProbeArmed.Store(true)
		offerUpdateNotice(&selfupdate.Release{Tag: "v9.9.9"})

		got := captureUpdateNoticeStderr(t, renderPendingUpdateNotice)
		want := "A new dflow release is available: v9.9.9 — run dflow update\n"
		if got != want {
			t.Fatalf("stderr = %q, want %q", got, want)
		}
		if !updateNoticeRendered.Load() {
			t.Fatalf("rendering a notice must set the once guard")
		}
	})

	t.Run("renders at most once per process", func(t *testing.T) {
		resetUpdateNoticeState(t)
		updateProbeArmed.Store(true)
		offerUpdateNotice(&selfupdate.Release{Tag: "v9.9.9"})

		first := captureUpdateNoticeStderr(t, renderPendingUpdateNotice)
		if first == "" {
			t.Fatalf("the first render must produce the notice")
		}
		second := captureUpdateNoticeStderr(t, renderPendingUpdateNotice)
		if second != "" {
			t.Fatalf("a second render produced %q, want the guard to keep it silent", second)
		}
	})

	t.Run("does not wait when nothing was armed", func(t *testing.T) {
		resetUpdateNoticeState(t)

		start := time.Now()
		got := captureUpdateNoticeStderr(t, renderPendingUpdateNotice)
		elapsed := time.Since(start)

		if got != "" {
			t.Fatalf("stderr = %q, want nothing when no probe was armed", got)
		}
		if elapsed >= noticeGrace {
			t.Fatalf("render waited %s on an unarmed probe; it must return immediately", elapsed)
		}
	})

	t.Run("stays silent when the release is not newer", func(t *testing.T) {
		resetUpdateNoticeState(t)
		updateProbeArmed.Store(true)
		offerUpdateNotice(&selfupdate.Release{Tag: releaseVersion})

		got := captureUpdateNoticeStderr(t, renderPendingUpdateNotice)
		if got != "" {
			t.Fatalf("stderr = %q, want no notice for an equal or older release", got)
		}
		if !updateNoticeRendered.Load() {
			t.Fatalf("a consumed result must set the once guard, so a later post-run cannot wait again")
		}
	})

	t.Run("json mode suppresses the line", func(t *testing.T) {
		resetUpdateNoticeState(t)
		utils.SetFormat(utils.FormatJSON)
		updateProbeArmed.Store(true)
		offerUpdateNotice(&selfupdate.Release{Tag: "v9.9.9"})

		got := captureUpdateNoticeStderr(t, renderPendingUpdateNotice)
		if got != "" {
			t.Fatalf("stderr = %q, want no notice in a machine-readable run", got)
		}
	})

	t.Run("discards an absent result after the grace budget", func(t *testing.T) {
		resetUpdateNoticeState(t)
		updateProbeArmed.Store(true)

		start := time.Now()
		got := captureUpdateNoticeStderr(t, renderPendingUpdateNotice)
		elapsed := time.Since(start)

		if got != "" {
			t.Fatalf("stderr = %q, want the dropped result to stay silent", got)
		}
		if elapsed < noticeGrace {
			t.Fatalf("render returned in %s, expected it to wait the %s grace budget", elapsed, noticeGrace)
		}
		if updateNoticeRendered.Load() {
			t.Fatalf("a dropped result must not burn the once guard")
		}
	})
}
