package log

import (
	"fmt"

	"go.uber.org/zap/zapcore"
)

type logObject struct {
	tag   LogTag
	error error
	data  []any
}

func L() logObject {
	return logObject{}
}

func (l logObject) Tag(t LogTag) logObject {
	l.tag = t
	return l
}

func (l logObject) Error(e error) logObject {
	l.error = e
	return l
}

func (l logObject) Add(k string, v any) logObject {
	// NOTE(evgenymng): no actual array copying is supposed to happen
	// when you call this method. Remember that slices are simply
	// views into the underlying data storage, so they may share it.
	l.data = append(l.data, k, v)
	return l
}

func (l logObject) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("tag", l.tag.String())

	if l.error != nil {
		enc.AddString("error", l.error.Error())
		enc.AddString("error_type", fmt.Sprintf("%T", l.error))
	}

	if len(l.data)%2 == 0 {
		for i := 0; i < len(l.data)-1; i = i + 2 {
			var ok bool
			var strKey string
			if strKey, ok = l.data[i].(string); !ok {
				S.Error(
					"L's Data array has a key that cannot be cast to string",
					L().Tag(LogTagLogParsing),
				)
				continue
			}
			// NOTE(evgenymng): the docs say that this function might be slow
			// and allocation-heavy.
			err := enc.AddReflected(strKey, l.data[i+1])
			if err != nil {
				return err
			}
		}
	} else {
		S.Error(
			"L's Data array isn't of even size, ignoring",
			L().Tag(LogTagLogParsing),
		)
	}
	return nil
}
