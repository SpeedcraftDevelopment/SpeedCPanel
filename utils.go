package main

func contains(arr []string, f func(p string) bool) bool {
	for _, s := range arr {
		if f(s) == true {
			return true
		}
	}
	return false
}

func compareString(str string) func(s string) bool {
	return func(s string) bool {
		return s == str
	}
}

func deleteFromAray(arr []string, f func(p string) bool) (newArray []string) {
	for _, str := range arr {
		if !f(str) {
			newArray = append(newArray, str)
		}
	}
	return newArray
}

func stringArrayToString(arr []string, delimiter string) (result string) {
	for i, str := range arr {
		if i == 0 {
			result += str
		} else {
			result += (delimiter + str)
		}
	}
	return result
}
