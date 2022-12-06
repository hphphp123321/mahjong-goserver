package common

func Max[T Comparable](o []T) T {
	if len(o) == 0 {
		var zero T
		return zero
	}
	maxIndex := 0
	for i := len(o) - 1; i > 0; i-- {
		if o[i].CompareTo(o[maxIndex]) > 0 {
			maxIndex = i
		}
	}
	return o[maxIndex]
}

func Min[T Comparable](o []T) T {
	if len(o) == 0 {
		var zero T
		return zero
	}
	maxIndex := 0
	for i := len(o) - 1; i > 0; i-- {
		if o[i].CompareTo(o[maxIndex]) < 0 {
			maxIndex = i
		}
	}
	return o[maxIndex]
}

func IndexOf[T Comparable](o T, t []T) int {
	if len(t) == 0 {
		return -1
	}
	for i := len(t) - 1; i >= 0; i-- {
		if t[i].CompareTo(o) == 0 {
			return i
		}
	}
	return -1
}

func MaxNum[T Num](ns []T) T {
	if len(ns) == 0 {
		var zero T
		return zero
	}
	maxIndex := 0
	for i := len(ns) - 1; i > 0; i-- {
		if ns[i]-ns[maxIndex] > 0 {
			maxIndex = i
		}
	}
	return ns[maxIndex]
}

func MinNum[T Num](ns []T) T {
	if len(ns) == 0 {
		var zero T
		return zero
	}
	minIndex := 0
	for i := len(ns) - 1; i > 0; i-- {
		if ns[i]-ns[minIndex] < 0 {
			minIndex = i
		}
	}
	return ns[minIndex]
}
