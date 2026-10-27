# Changelog

All notable changes to this project are documented here. The format is
based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the
project uses [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.1.0] - 2026-10-27

First public release.

### Added
- Full-screen TUI metronome with sample-accurate beat scheduling.
- 60 built-in CC0 click sounds with a previewing sound picker.
- Tap tempo, time signatures (1–16 beats, unit 1/2/4/8/16), accent and volume.
- `-sounds` flag for custom WAV sound sets.
- `-latency` flag to tune display/audio sync.
- `-version` flag.

### Changed
- Out-of-range `-bpm` / `-beats` values and unknown `-sound` names are now
  errors instead of being silently adjusted.

### Fixed
- Malformed custom WAV files (zero or extreme sample rate, empty data) are
  rejected with an error instead of crashing.

[Unreleased]: https://github.com/altf2o/clinome/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/altf2o/clinome/releases/tag/v0.1.0
