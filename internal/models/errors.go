// Package models holds the pure models, functions and constants of the bot:
// chat access rules, text splitting, token redaction, command parsing and
// every named constant of the project. It depends on the standard library
// alone — no HTTP, no Telegram, no config and no logger.
package models

// The models layer contains only pure logic and returns no errors,
// so no errors are defined here. The file exists to satisfy the
// project rule that every layer has an errors.go at its package root.
