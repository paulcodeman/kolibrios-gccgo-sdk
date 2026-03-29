//go:build kolibrios && gccgo
// +build kolibrios,gccgo

// Package sqlite3 provides a thin SQLite driver and a KolibriOS VFS.
//
// The package keeps the core SQLite source as upstream amalgamation and limits
// platform-specific changes to the surrounding VFS/driver glue.
//
// The current bootstrap runtime is single-threaded. Prefer sqlite3.Open, or set
// database/sql pools to a single open connection when using sql.Open directly.
package sqlite3
