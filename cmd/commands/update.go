package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/yepizrene-devoost/dflow/cmd/selfupdate"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

// updateReport is the machine-readable report of `dflow update`.
//
// Field order is the document order encoding/json produces, and it matches the
// keys the human-readable mode prints. `updated` is true only when the running
// binary was actually replaced, and `path` names that binary only in that case,
// so a caller can never read an untouched path as a successful update.
type updateReport struct {
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version"`
	UpdateAvailable bool   `json:"update_available"`
	Updated         bool   `json:"updated"`
	Path            string `json:"path"`
	ReleaseURL      string `json:"release_url"`
}

// updateAPIURLEnv names the environment variable that overrides the GitHub API
// base URL the updater queries.
//
// It exists so tests can point the command at an `httptest` server and never
// touch the real network, and so a user behind a mirror can redirect the
// lookup. The default is the public GitHub API the installer already documents.
const updateAPIURLEnv = "DFLOW_UPDATE_API_URL"

// UpdateCmd replaces the running dflow binary with the latest published release.
//
// It answers three questions in one command: whether a newer release exists,
// what its changelog says, and how to install it while leaving the current
// binary untouched unless the whole download verifies.
var UpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update dflow to the latest published release",
	Long: `Download the latest published dflow release and replace the running binary.

The command asks GitHub for the newest release, compares its version with the
one this binary reports, and, when it is newer, downloads the archive for the
host platform, verifies it against the release's published SHA-256 checksums,
extracts the new binary and swaps it into place next to the running one. A
failure at any step leaves the current binary untouched.

The release notes are summarized before anything is downloaded, so the decision
to install is made with the changelog in view, and the same summary is shown
again after the swap next to the full release page. When the command runs on a
terminal it asks for confirmation before downloading; the default answer is yes
and --yes skips the question. A run whose input or output is not a terminal (a
script or a pipe) is never prompted and installs as it always has.

Pass --check to report whether an update is available without downloading or
replacing anything. Pass --force to reinstall the latest release even when it is
not newer than the installed one (for example, after a corrupted install).
--check and --force cannot be combined. An explicit check also refreshes the
update-check cache, which keeps the background startup notice quiet for the next
24 hours. Pass --json for one machine-readable document instead; in that mode
stdout carries a single six-key document and nothing else, with no prompt and no
notes summary.

The update is driven by the release the repository publishes, not by this
checkout: a binary built or installed with tools you manage yourself (a plain
'go build', or 'go install') carries the "dev" version marker and has no release
provenance, so its version cannot be compared. The command warns about that and
continues instead of refusing, but a binary managed by 'go install' is usually
best updated by running that command again.

Set DFLOW_UPDATE_API_URL to query a different GitHub API base URL, for example a
mirror or a local test server. It defaults to https://api.github.com.

The updater needs no authentication: dflow's releases are public. The trade-off
is GitHub's unauthenticated rate limit (60 requests per hour per IP), which the
command reports as an actionable error when it is hit.`,
	Example: `  dflow update
  dflow update --yes
  dflow update --check
  dflow update --force
  dflow update --json`,
	Args: cobra.NoArgs,
	// The format must be decided before RunE runs, so a failure such as the
	// --check/--force conflict is reported in the requested format instead of as
	// styled text. Cobra runs PersistentPreRun, then PreRunE, then RunE: this
	// mirrors the existing --json contract in status.go.
	PreRunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, err := cmd.Flags().GetBool("json")
		if err != nil {
			return err
		}
		if jsonOutput {
			utils.SetFormat(utils.FormatJSON)
		}
		return nil
	},
	RunE: runUpdate,
}

// runUpdate is the command body, kept separate from the command definition so
// the flag resolution, the provenance warning and the three outcomes (check,
// already up to date, install) read as one sequence.
func runUpdate(cmd *cobra.Command, args []string) error {
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}
	checkOnly, err := cmd.Flags().GetBool("check")
	if err != nil {
		return err
	}
	yes, err := cmd.Flags().GetBool("yes")
	if err != nil {
		return err
	}

	// --check only reports and --force only matters when something is replaced,
	// so the two together are a contradiction, not a harmless no-op. Naming the
	// conflict beats silently honouring one of them.
	if checkOnly && force {
		return fmt.Errorf("--check and --force cannot be combined: --check only reports and never replaces the binary, so --force has nothing to force")
	}

	target, err := resolveUpdateTarget()
	if err != nil {
		return err
	}

	// The version marker is the half of the version string that identifies the
	// build channel, and the only half that says whether "newer" is even a
	// meaningful question. A marker with no release provenance means the version
	// was never injected, so it cannot be compared: warn with the user-facing
	// explanation and continue, because refusing would strand exactly the users
	// the message is meant to help.
	current := utils.VersionMarker()
	hasProvenance := selfupdate.HasReleaseProvenance(current)
	if !hasProvenance {
		utils.Warn("this dflow binary carries no release provenance (version marker %q): it was built without a published release version, so it cannot be compared with the latest one. A binary installed with 'go install' is usually updated by running 'go install github.com/yepizrene-devoost/dflow@latest' again.", current)
		// The warning is a paragraph of its own, and on a terminal the helpers
		// wrap it into several physical lines. A blank line separates that block
		// from the report that follows, so the alert never runs into the first
		// report line. Plain is suppressed in JSON mode exactly as Warn is, so a
		// machine-readable run still carries only its document.
		utils.Plain("%s", "")
	}

	client := selfupdate.NewReleaseClientWithBaseURL(updateBaseURL(), nil)
	latest, err := client.LatestRelease()
	if err != nil {
		return err
	}

	// This command just learned the latest release from GitHub, so record it:
	// the background startup notice reuses a cache younger than 24 hours, and an
	// explicit check is the freshest answer there is. The refresh is best-effort
	// on purpose — it is bookkeeping for a courtesy notice, so neither a machine
	// with no resolvable cache path nor a failed write may affect the check or
	// the update the user actually asked for.
	if cachePath, pathErr := selfupdate.UpdateCheckCachePath(); pathErr == nil {
		_ = selfupdate.RecordUpdateCheck(cachePath, latest, time.Now())
	}

	// A binary without release provenance has no version to compare, so the
	// comparison is bypassed rather than handed to the comparator's string
	// fallback: "dev" sorts below every digit, which would make a dev build
	// read as "already up to date" and strand the user behind a --force they
	// have no reason to know about. Without provenance there is always a
	// release to move to, so the answer is simply yes.
	report := updateReport{
		CurrentVersion:  current,
		LatestVersion:   latest.Tag,
		UpdateAvailable: !hasProvenance || selfupdate.IsNewer(latest.Tag, current),
		ReleaseURL:      latest.HTMLURL,
	}

	if checkOnly {
		if utils.CurrentFormat() == utils.FormatJSON {
			return utils.EmitJSON(report)
		}
		reportHumanCheck(report, latest)
		return nil
	}

	if !report.UpdateAvailable && !force {
		if utils.CurrentFormat() == utils.FormatJSON {
			return utils.EmitJSON(report)
		}
		utils.Success("dflow %s is already up to date (%s)", utils.GetVersion(), latest.Tag)
		return nil
	}

	proceed, err := confirmUpdateInstall(yes, current, target, latest)
	if err != nil {
		return err
	}
	if !proceed {
		return nil
	}

	if err := installLatest(target, latest); err != nil {
		return err
	}

	// Only reached when the swap succeeded, so the report can claim the update
	// the binary is now running, not the one that was merely downloaded.
	report.Updated = true
	report.Path = target
	if utils.CurrentFormat() == utils.FormatJSON {
		return utils.EmitJSON(report)
	}
	utils.Info("binary: %s", target)
	reportNotesSummary(latest)
	reportHumanReleaseURL(report.ReleaseURL)
	return nil
}

// confirmUpdateInstall reports what the update would install and, on a
// terminal, asks before a single byte is downloaded. It returns whether the
// install should proceed; a decline is a successful command, not an error.
//
// The ordering is the point: the summary is printed here, before installLatest
// runs, so a user who declines has spent no bandwidth and the running binary is
// untouched.
func confirmUpdateInstall(yes bool, current, target string, latest *selfupdate.Release) (bool, error) {
	// The notes go through the shared output layer, which suppresses them in
	// JSON mode, so the machine-readable document never gains human chrome.
	reportNotesSummary(latest)

	// A machine-readable run and a non-interactive one both keep the historical
	// piped behaviour: no question is asked, and the install proceeds. A prompt
	// would corrupt the JSON document on stdout, and with no terminal there is
	// nobody to answer one — scripting `dflow update` must keep working exactly
	// as it did before this prompt existed.
	if utils.CurrentFormat() == utils.FormatJSON || !utils.IsInteractive() {
		return true, nil
	}

	if yes {
		return true, nil
	}

	var confirmed bool
	if err := survey.AskOne(&survey.Confirm{
		Message: fmt.Sprintf("Install dflow %s and replace %s?", latest.Tag, target),
		Default: true,
	}, &confirmed); err != nil {
		// A prompt that could not even be answered is a real failure, not a
		// decline: treating it as "no" would silently skip a requested update.
		return false, err
	}
	if !confirmed {
		utils.Info("update aborted: dflow %s stays installed", current)
		return false, nil
	}
	return true, nil
}

// maxDigestWidth caps how wide a release-notes content line may render, in
// runes before the caller's two-space indent. It exists so the digest shares one
// text measure with the icon-prefixed lines above it (cmd/utils caps those at
// the same content width): without it, a bullet on a 200-column terminal would
// run to the terminal edge while the surrounding report wraps much earlier.
const maxDigestWidth = 100

// minDigestWidth is the floor under the measured content width. Below it a
// wrapped bullet becomes a column of two-word fragments, which is harder to read
// than letting the line ride past a narrow terminal's edge — the same rationale
// as cmd/utils.minWrapWidth, applied to the digest's own content width rather
// than to the terminal.
const minDigestWidth = 40

// digestIndentWidth is the two-space indent reportNotesSummary puts in front of
// every content line, so a note is visibly subordinate to the "what's new" header
// above it. The digest's content width is measured after this indent, because the
// indent is part of the columns the terminal shows.
const digestIndentWidth = 2

// releaseNotesContentWidth measures how wide a release-notes content line may
// render for the stream this process writes to.
//
// It returns 0 when the width could not be measured — stdout is a pipe, a file or
// a command substitution — which is the digest's "do not wrap" signal, so piped
// and scripted output keeps the historical one-line-per-source-line shape. A
// measured terminal yields its width minus the caller's indent, capped at
// maxDigestWidth and floored at minDigestWidth.
func releaseNotesContentWidth() int {
	width, ok := utils.TerminalWidth()
	if !ok {
		return 0
	}

	content := width - digestIndentWidth
	if content > maxDigestWidth {
		content = maxDigestWidth
	}
	if content < minDigestWidth {
		content = minDigestWidth
	}
	return content
}

// reportNotesSummary prints the short, terminal-sized list of release-note
// lines for the release being offered. An empty body (GitHub has no changelog
// for this release) prints nothing at all, because an empty header would
// promise a section that has no content.
//
// The width passed to the summarizer is measured once for this stream: a terminal
// gets wrapped lines sized to the indent below, and an unmeasured stream gets 0,
// which the summarizer reads as "wrap nothing".
//
// A separator line from the renderer is printed verbatim, without the two-space
// indent the content lines carry: the indent exists to subordinate a note to its
// header, and applying it to a blank line would put two spaces of trailing
// whitespace on a line that shows nothing.
func reportNotesSummary(latest *selfupdate.Release) {
	lines := selfupdate.SummarizeReleaseNotes(latest.Body, releaseNotesContentWidth())
	if len(lines) == 0 {
		return
	}
	utils.Info("what's new in %s:", latest.Tag)
	for _, line := range lines {
		if line == "" {
			utils.Plain("%s", line)
			continue
		}
		utils.Plain("  %s", line)
	}
}

// reportHumanCheck renders the read-only report. It names both versions, the
// verdict, what the newer release changes, and the release page, so a user can
// decide with the changelog in view.
func reportHumanCheck(report updateReport, latest *selfupdate.Release) {
	utils.Info("current version: %s", report.CurrentVersion)
	utils.Info("latest release: %s", report.LatestVersion)
	if report.UpdateAvailable {
		utils.Info("an update is available: run `dflow update` to install %s", report.LatestVersion)
		// Only an available update has "what's new" to offer: for the release the
		// binary already runs, its own notes would be noise.
		reportNotesSummary(latest)
	} else {
		utils.Info("no update available: %s is already the latest release", report.CurrentVersion)
	}
	reportHumanReleaseURL(report.ReleaseURL)
}

// reportHumanReleaseURL prints the release page only when the API supplied one:
// an empty "release notes: " line is worse than no line at all. The line is
// opened by a blank line so it does not run together with the digest above it.
//
// Both of its callers are human-only: the JSON paths return before reaching this
// function, and the helpers are suppressed in JSON mode anyway, so the
// machine-readable document keeps exactly its six keys.
func reportHumanReleaseURL(url string) {
	if url != "" {
		utils.Plain("%s", "")
		utils.Info("release notes: %s", url)
	}
}

// resolveTargetBinary returns the path of the running dflow executable with
// symlinks resolved, so a binary invoked through a symlinked entry is updated
// at its real location rather than replacing the link itself.
//
// os.Executable reports the path the process was started from, which on some
// platforms is a symlink (and on macOS may traverse the /var to /private/var
// link). When EvalSymlinks cannot resolve it, the raw path is the best answer
// available; the caller notes the failure and updates there instead of aborting.
func resolveTargetBinary() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("could not locate the running dflow binary: %w", err)
	}

	resolved, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return executable, err
	}
	return resolved, nil
}

// resolveUpdateTarget owns the whole policy for turning the raw executable
// lookup into the target the update installs into, so runUpdate reads one line
// instead of interleaving a fatal branch with a warning branch:
//
//   - no path at all is fatal: there is nothing to update and nothing to
//     install into, and continuing would fail later inside EnsureWritable with
//     a message that names an empty path;
//   - a path that could not be symlink-resolved is still the binary the process
//     was started from, so the update proceeds there and only the failed
//     resolution is noted.
func resolveUpdateTarget() (string, error) {
	target, resolveErr := resolveTargetBinary()
	if target == "" {
		return "", fmt.Errorf("could not locate the running dflow binary: %w", resolveErr)
	}
	if resolveErr != nil {
		utils.Warn("could not resolve the full path of the running dflow binary (%v); continuing with %s", resolveErr, target)
	}
	return target, nil
}

// updateBaseURL returns the GitHub API base URL to query, honouring the
// DFLOW_UPDATE_API_URL override and falling back to the public API. Blank and
// whitespace-only overrides are ignored so an empty environment variable cannot
// silently turn every lookup into a request to the empty string.
func updateBaseURL() string {
	if override := strings.TrimSpace(os.Getenv(updateAPIURLEnv)); override != "" {
		return override
	}
	return selfupdate.DefaultBaseURL
}

// installLatest downloads, verifies, extracts and swaps the release archive for
// the host platform.
//
// The writability precheck runs before the download on purpose: an install
// location the user cannot write fails immediately, without spending bandwidth
// on an archive that could never be installed. Every temporary artifact lives
// in one directory that is removed on return, so no failure path can leave a
// downloaded archive or an extracted binary behind.
func installLatest(target string, latest *selfupdate.Release) error {
	if err := selfupdate.EnsureWritable(target); err != nil {
		return err
	}

	archiveName, err := selfupdate.ArchiveName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}
	archiveURL, ok := latest.AssetURL(archiveName)
	if !ok {
		return fmt.Errorf("the %s release does not publish the %s asset: the release looks incomplete", latest.Tag, archiveName)
	}
	checksumsName := selfupdate.ChecksumsName(latest.Version)
	checksumsURL, ok := latest.AssetURL(checksumsName)
	if !ok {
		return fmt.Errorf("the %s release does not publish the %s asset: the release looks incomplete", latest.Tag, checksumsName)
	}

	tempDir, err := os.MkdirTemp("", "dflow-update-*")
	if err != nil {
		return fmt.Errorf("could not create a temporary directory for the update: %w", err)
	}
	defer os.RemoveAll(tempDir)

	archivePath := filepath.Join(tempDir, archiveName)
	checksumsPath := filepath.Join(tempDir, checksumsName)
	extractedPath := filepath.Join(tempDir, "dflow")

	steps := installSteps{
		archiveURL:    archiveURL,
		checksumsURL:  checksumsURL,
		archiveName:   archiveName,
		archivePath:   archivePath,
		checksumsPath: checksumsPath,
		extractedPath: extractedPath,
		target:        target,
	}

	// The spinner writes to stdout, which JSON mode reserves for the document,
	// so it exists only for the human renderer. A non-interactive stream degrades
	// it to plain single lines, which is still the progress a user wants to see.
	var spinner *utils.Spinner
	if utils.CurrentFormat() != utils.FormatJSON {
		spinner = utils.NewSpinner(fmt.Sprintf("Downloading and installing dflow %s...", latest.Tag))
		spinner.Start()
	}

	if err := steps.run(); err != nil {
		if spinner != nil {
			spinner.Clear()
		}
		return err
	}

	if spinner != nil {
		spinner.Stop(fmt.Sprintf("Updated dflow to %s", latest.Tag))
	}
	return nil
}

// installSteps bundles every path and URL the download-verify-install sequence
// consumes, so the sequence is one method on a named value instead of a call
// with seven positional strings where swapping two same-typed arguments would
// still compile.
type installSteps struct {
	archiveURL    string
	checksumsURL  string
	archiveName   string
	archivePath   string
	checksumsPath string
	extractedPath string
	target        string
}

// run is the ordered download-to-swap sequence. Each step is fatal to the
// update: an unverifiable or unextractable archive must never reach
// InstallBinary, and InstallBinary itself stages the new binary next to the
// target so a failed swap leaves the running one untouched.
func (s installSteps) run() error {
	if err := selfupdate.DownloadFile(s.archiveURL, s.archivePath, nil); err != nil {
		return err
	}
	if err := selfupdate.DownloadFile(s.checksumsURL, s.checksumsPath, nil); err != nil {
		return err
	}
	if err := selfupdate.VerifyChecksum(s.archivePath, s.checksumsPath, s.archiveName); err != nil {
		return err
	}
	if err := selfupdate.ExtractBinary(s.archivePath, s.extractedPath, runtime.GOOS); err != nil {
		return err
	}
	return selfupdate.InstallBinary(s.extractedPath, s.target, runtime.GOOS)
}

func init() {
	UpdateCmd.Flags().Bool("force", false, "Reinstall the latest release even when it is not newer than the installed one")
	UpdateCmd.Flags().Bool("check", false, "Report whether an update is available without downloading or replacing anything")
	UpdateCmd.Flags().BoolP("yes", "y", false, "Skip the installation confirmation prompt")
	UpdateCmd.Flags().Bool("json", false, "Print the report as a single JSON document on stdout")
}
