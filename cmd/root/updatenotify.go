package root

import (
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/spf13/cobra"
	"github.com/yepizrene-devoost/dflow/cmd/selfupdate"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

// updateNoCheckEnv is the environment variable a user sets to opt out of the
// startup release check for good.
//
// It is named "no check" rather than "no notify" because setting it also skips
// the network probe: an opt-out that still phoned home would be a lie, and the
// only reason to skip the probe is that the user does not want dflow to ask.
const updateNoCheckEnv = "DFLOW_NO_UPDATE_CHECK"

// updateNoticeFormat is the single line every available update renders. It is
// the whole notification contract: one plain line naming the new version and
// the exact command that installs it.
const updateNoticeFormat = "A new dflow release is available: %s — run dflow update"

// noticeGrace bounds how long the end of a command waits for the background
// update probe.
//
// The notice is a courtesy, so it must never be the reason a command feels
// slow: the probe runs while the command does its own work, and if it has not
// answered within this budget the result is dropped for this run and the next
// command tries again. 150ms is long enough that the request is usually in
// flight before a fast command finishes, and short enough to be invisible next
// to the command's own output.
const noticeGrace = 150 * time.Millisecond

// updateNoticeSlot carries the probe result from the goroutine started in
// PersistentPreRun to the renderer that runs in PersistentPostRun.
//
// Capacity 1 is deliberate: there is exactly one result to hand over, and the
// non-blocking send in offerUpdateNotice means a redundant second probe cannot
// block a goroutine nobody will ever read from.
var updateNoticeSlot = make(chan *selfupdate.Release, 1)

// updateProbeArmed records that this process actually committed to producing a
// result — a fresh cached answer was handed over, or the network probe was
// started.
//
// Without it, every invocation the probe is skipped for (`--version`, `--help`,
// a `--json` run, `dflow update`) would still make PersistentPostRun wait out
// noticeGrace for a result that was never coming, taxing exactly the commands
// the gate exists to keep quiet. It also keeps the renderer honest: it waits
// only when waiting can succeed.
var updateProbeArmed atomic.Bool

// updateNoticeRendered is the process-level once guard.
//
// The bare `dflow` invocation runs the root's Run, which re-executes the root
// command with `--help`; the guard makes "at most one notice per process"
// explicit rather than a property of Cobra's hook order, so a future change to
// that ordering cannot turn the notice into a duplicate line. It is set on
// every path that consumes a result, so a later post-run neither waits again
// nor prints a second copy.
var updateNoticeRendered atomic.Bool

// updateNoticeLookupEnv is the environment seam the opt-out gate reads.
//
// It is a variable rather than a direct os.Getenv call so the unit tests can
// drive the gate without mutating the process environment, mirroring the
// lookupEnv seam cmd/selfupdate already established for the same reason. The
// end-to-end tests are unaffected: they pass DFLOW_NO_UPDATE_CHECK to the child
// process, where this variable still holds os.Getenv.
var updateNoticeLookupEnv = os.Getenv

// maybeStartUpdateProbe launches the background "is there a newer release?"
// check for the command about to run.
//
// It is called from PersistentPreRun, after the banner, so the probe overlaps
// with the work the user asked for instead of adding to it. Every gate below is
// a reason NOT to spend a network request or show a notice:
//
//   - a machine-readable run must carry only its document;
//   - the same invocations the banner already skips (help, completion, version)
//     have no "after the output" moment to attach a notice to;
//   - DFLOW_NO_UPDATE_CHECK is the user's explicit, persistent opt-out;
//   - `dflow update` is already answering the question, so a notice printed
//     after it would contradict the command's own report;
//   - a build with no release provenance (a plain `go build`, `go install`, or
//     a source checkout) has no version to compare and no release channel to
//     point at.
//
// The cache is consulted first: a fresh recorded check is handed to the
// renderer as a synthetic release without any network call, so a probe runs at
// most once per TTL. Only a stale or absent cache starts the goroutine, whose
// network failure is silent by design and whose successful result is recorded
// before it is offered, so a run that renders a notice has also paid for the
// next run's silence.
func maybeStartUpdateProbe(cmd *cobra.Command) {
	if !updateProbeEligible(cmd) {
		return
	}

	path, err := selfupdate.UpdateCheckCachePath()
	if err != nil {
		// No cache location means no place to read or record the answer. A
		// notice that cannot be kept honest across runs is not worth a network
		// request, so stay silent.
		return
	}

	state, err := selfupdate.ReadUpdateCheckState(path)
	if err != nil {
		// A cache that cannot be read is not an answer. Probing again would
		// overwrite it without knowing whether it was valid, so skip.
		return
	}

	// From here a result is guaranteed (immediately from the cache, or from the
	// goroutine), so the renderer is worth a bounded wait for it.
	updateProbeArmed.Store(true)

	if selfupdate.IsUpdateCheckFresh(state, time.Now()) {
		// The cached answer is the whole point of the cache: report it without
		// touching the network. A zero or empty cached version simply fails the
		// IsNewer check at render time.
		offerUpdateNotice(&selfupdate.Release{Tag: state.LatestVersion, HTMLURL: state.ReleaseURL})
		return
	}

	go func() {
		release, err := selfupdate.ProbeLatestRelease(selfupdate.ResolveBaseURL())
		if err != nil {
			// Any network failure is silent: this is a courtesy check, so an
			// offline machine or an exhausted rate limit must print nothing the
			// user did not ask for.
			return
		}
		// Persisting is best-effort and deliberately before the handoff: the
		// notice is shown either way, and recording first means a process that
		// exits right after rendering has already written the cache the next
		// run reads.
		_ = selfupdate.RecordUpdateCheck(path, release, time.Now())
		offerUpdateNotice(release)
	}()
}

// updateProbeEligible reports whether this invocation may start the probe at
// all. It is only the ordered test; the reasons live in maybeStartUpdateProbe's
// doc comment so each one is a named, individually testable gate.
func updateProbeEligible(cmd *cobra.Command) bool {
	if utils.CurrentFormat() == utils.FormatJSON {
		return false
	}
	if shouldSkipBanner(os.Args[1:]) {
		return false
	}
	if strings.TrimSpace(updateNoticeLookupEnv(updateNoCheckEnv)) != "" {
		return false
	}
	if cmd != nil && cmd.Name() == "update" {
		return false
	}
	if !selfupdate.HasReleaseProvenance(utils.VersionMarker()) {
		return false
	}
	return true
}

// offerUpdateNotice hands a release to the renderer without ever blocking.
func offerUpdateNotice(release *selfupdate.Release) {
	if release == nil {
		return
	}
	select {
	case updateNoticeSlot <- release:
	default:
		// A second probe already filled the slot; the first answer is the one
		// that will be rendered.
	}
}

// renderPendingUpdateNotice prints the one-line notice when the probe has an
// answer ready before the grace budget expires.
//
// It runs from PersistentPostRun, so a command that failed already exited
// non-zero in Execute and never reaches this point: the notice never competes
// with an error, and no caller has to reason about a notice after a failure.
// When nothing was armed it returns without waiting, and when the result is not
// newer than the running binary it says nothing — an "up to date" line after
// every command is noise, not a courtesy.
func renderPendingUpdateNotice() {
	if !updateProbeArmed.Load() || updateNoticeRendered.Load() {
		return
	}

	select {
	case release := <-updateNoticeSlot:
		// The result has been consumed; the once guard is set before anything
		// else so no later post-run can wait for it again or print it twice.
		updateNoticeRendered.Store(true)
		if release == nil || !selfupdate.IsNewer(release.Tag, utils.VersionMarker()) {
			return
		}
		utils.Notice(updateNoticeFormat, release.Tag)
	case <-time.After(noticeGrace):
		// The probe is too slow to be worth waiting for. It is dropped for this
		// run and the gate is left unset, so a command that runs again in this
		// process would get another chance; the next process reads the cache
		// the probe writes.
	}
}
