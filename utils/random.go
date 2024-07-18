package utils

import "time"

func RandomOne[T any](list []*T) (int64, *T) {
	if len(list) == 0 {
		return 0, nil
	}

	index := time.Now().UnixMicro() % int64(len(list))

	return index, list[index]
}
