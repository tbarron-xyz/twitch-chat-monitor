package main

import "fmt"

func EH(err error) { // simple Error Handle
	if err != nil {
		fmt.Println(err)
	}
}

func pr(arg ...interface{}) {
	debug := true
	if debug {
		fmt.Println(arg...)
	}
}

func mapfromarray(arr []string) map[string]struct{} { // so we can do _,contains := map[element]
	ret := map[string]struct{}{}
	for _, e := range arr {
		ret[e] = struct{}{}
	}
	return ret
}

func remove(arr []string, e string) []string {
	i := indexOf(arr, e)
	return append(arr[:i], arr[i+1:]...)
}

func indexOf(arr []string, e string) int {
	for i := 0; i < len(arr); i++ {
		if arr[i] == e {
			return i
		}
	}
	return -1
}
