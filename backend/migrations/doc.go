// Package migrations embeds Aether's ordered database schema migrations and
// exposes them to database adapters. It contains migration resources only; the
// PostgreSQL adapter controls their transactional application and bookkeeping.
package migrations
