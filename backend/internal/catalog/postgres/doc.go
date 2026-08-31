// Package postgres implements Aether's catalog repository using PostgreSQL.
// It owns SQL persistence, transaction handling, and translation of database
// errors into domain-level repository errors; business rules remain in domain.
package postgres
