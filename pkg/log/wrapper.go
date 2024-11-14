package log

import "go.uber.org/zap"

/**
* Basically, redefine [zap]'s methods to require an argument
* of the [logObject] type, then inline it.
 */

func (s *Logger) Debug(msg string, logObject LogObject) {
	defer s.internal.Sync()
	s.internal.Debugw(msg, zap.Inline(logObject))
}

func (s *Logger) Info(msg string, logObject LogObject) {
	defer s.internal.Sync()
	s.internal.Infow(msg, zap.Inline(logObject))
}

func (s *Logger) Warn(msg string, logObject LogObject) {
	s.internal.Warnw(msg, zap.Inline(logObject))
}

func (s *Logger) Error(msg string, logObject LogObject) {
	s.internal.Errorw(msg, zap.Inline(logObject))
}

// In development, the logger panics after sending the message.
func (s *Logger) DPanic(msg string, logObject LogObject) {
	s.internal.DPanicw(msg, zap.Inline(logObject))
}

func (s *Logger) Panic(msg string, logObject LogObject) {
	s.internal.Panicw(msg, zap.Inline(logObject))
}

// The logger calls [os.Exit] after sending the message.
func (s *Logger) Fatal(msg string, logObject LogObject) {
	s.internal.Fatalw(msg, zap.Inline(logObject))
}
