package main

// AppVersion is embedded at compile time as a plain constant.
// Keep in sync with apps/desktop/package.json "version".
// [LOG] Choice: plain constant over ldflags — no build-script changes needed
// for Phase 1; ldflags can be wired later when release automation is added.
const AppVersion = "0.1.0"
