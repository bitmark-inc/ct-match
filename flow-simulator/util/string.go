package util

import "strconv"

func StringFromNum(num int) string {
	str := strconv.Itoa(num)
	if num < 10 {
		str = "0" + str
	}

	return str
}
