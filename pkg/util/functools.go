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

func Values[K comparable, V any](m map[K]V) []V {
	result := make([]V, 0, len(m))
	for _, v := range m {
		result = append(result, v)
	}
	return result
}
