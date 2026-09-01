// Package audit creates privacy-safe, append-only security audit events.
//
// The package accepts only typed actions and a small allowlist of object types.
// It records identifiers and outcomes, never request payloads, query strings,
// or other free-form metadata.
package audit
