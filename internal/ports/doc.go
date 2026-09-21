// Package ports declares the interfaces internal/service needs from the
// adapter layer (ADR 0002, ADR 0013): one port per file. Like
// internal/model, it depends on nothing else in this module — adapters
// implement these interfaces structurally, without importing this package,
// though internal/adapter does import it for compile-time satisfaction
// assertions.
package ports
