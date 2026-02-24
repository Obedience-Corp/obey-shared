// Package contract defines the shared types and protocol for the central
// watcher contract file (watchers.yaml). Camp and fest write entries to this
// contract, and the obey daemon reads it to configure its filesystem watchers.
//
// The contract uses owner-scoped merging: each tool (camp, fest) owns its
// entries and can update them independently without affecting the other's
// entries.
package contract
