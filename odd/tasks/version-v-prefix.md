# version-v-prefix

Restore the `v` prefix on the version marker and drop the revision for clean
release builds.

Issue: [#40](https://github.com/yepizrene-devoost/dflow/issues/40)

## Work Units

- [x] **WU1**: Fix the GoReleaser ldflags to inject the tag form (`{{.Tag}}`)
  instead of the bare version (`{{.Version}}`), preserving `snapshot-<short>`
  for snapshot builds.
- [x] **WU2**: Add test cases for clean-release and dirty-release markers in
  `cmd/utils/version_test.go`.
- [x] **WU3**: Verify snapshot build still reports `snapshot-<short>` and
  `HasReleaseProvenance` classifies it correctly.
