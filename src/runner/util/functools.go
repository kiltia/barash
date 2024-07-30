package util

func Map[T, V any](ts []T, fn func(T) V) []V {
	result := make([]V, len(ts))
	for i, t := range ts {
		result[i] = fn(t)
	}
	return result
}

func Reduce[T, V any](ts []T, initial V, reducer func(V, T) V) V {
	result := initial
	for _, t := range ts {
		result = reducer(result, t)
	}
	return result
}
