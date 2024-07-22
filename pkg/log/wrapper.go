package log

import "go.uber.org/zap"

/**
* Basically, redefine [zap]'s methods to require an argument
* of the [logObject] type, then inline it.
 */

func (s *Logger) Debug(msg string, logObject logObject) {
	s.internal.Debugw(msg, zap.Inline(logObject))
}

func (s *Logger) Info(msg string, logObject logObject) {
	s.internal.Infow(msg, zap.Inline(logObject))
}

func (s *Logger) Warn(msg string, logObject logObject) {
	s.internal.Warnw(msg, zap.Inline(logObject))
}

func (s *Logger) Error(msg string, logObject logObject) {
	s.internal.Errorw(msg, zap.Inline(logObject))
}

// In development, the logger panics after sending the message.
func (s *Logger) DPanic(msg string, logObject logObject) {
	s.internal.DPanicw(msg, zap.Inline(logObject))
}

func (s *Logger) Panic(msg string, logObject logObject) {
	s.internal.Panicw(msg, zap.Inline(logObject))
}

// The logger calls [os.Exit] after sending the message.
func (s *Logger) Fatal(msg string, logObject logObject) {
	s.internal.Fatalw(msg, zap.Inline(logObject))
}
