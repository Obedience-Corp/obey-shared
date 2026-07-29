// Package festivalbundle implements the portable Festival Bundle (.festival) format.
//
// A Festival Bundle is a ZIP (or directory package) containing:
//
//	info.json   — kind, identity, timestamps, origin/subject metadata
//	payload/    — work-unit tree; optional payload/.artifacts/ for vendored files
//
// Spec (normative design): workflow/design/festival-portable-text-bundle/SPEC.md
// in the obedience-growth-rd campaign (work item WI-93268b). Behavioral oracle:
// that design workitem's Python prototype under prototype/.
//
// Format rules (v1):
//   - kind is required; extension is always .festival
//   - in-root file links are not moved or rewritten
//   - out-of-root / absolute file links are copied into .artifacts/ and rewritten
//   - network URLs are left unchanged
//   - bundle.id is a content hash of payload/ only (sha256:…)
//   - open/unbundle never executes a festival run
//
// Consumers: camp and fest CLIs (and later other Obedience tools). Depend only
// on published obey-shared module versions in product go.mod files.
package festivalbundle
