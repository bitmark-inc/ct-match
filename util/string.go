package util

import "strconv"

func StringFromNum(num int) string {
	str := strconv.Itoa(num)
	if num < 10 {
		str = "0" + str
	}

	return str
}

func ShortenAccountNumber(accountNumber string) string {
	length := len(accountNumber)
	if length < 8 {
		return ""
	}

	return "[" + accountNumber[:4] + "..." + accountNumber[length-4:] + "]"
}
