package metrics

import "log"

// Logger supports logging at various log levels.
type Logger interface {
	// Debug logs a message at Debug level.
	Debug(args ...any)

	// Info logs a message at Info level.
	Info(args ...any)

	// Warn logs a message at Warning level.
	Warn(args ...any)

	// Error logs a message at Error level.
	Error(args ...any)

	// Fatal logs a message at Fatal level
	// and process will exit with status set to 1.
	Fatal(args ...any)
}

type defaultLogger struct{}

func (d *defaultLogger) Debug(args ...any) {
	log.Println(args...)
}
func (d *defaultLogger) Info(args ...any) {
	log.Println(args...)
}
func (d *defaultLogger) Warn(args ...any) {
	log.Println(args...)
}
func (d *defaultLogger) Error(args ...any) {
	log.Println(args...)
}
func (d *defaultLogger) Fatal(args ...any) {
	log.Fatalln(args...)
}
