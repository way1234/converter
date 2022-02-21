package tool

import (
	"strings"
	"unicode"
)

// 下划线风格转小驼峰命名法
func ToSmallCamelCase(s string) string {

	arr := strings.Split(s, "_")
	var result string
	if len(arr) > 0 {
		for i, v := range arr {
			if i == 0 {
				result += LcFirst(v)
			} else {
				result += UcFirst(v)
			}
		}
	}

	return result
}

// 下划线风格转大驼峰命名法
func ToBigCamelCase(s string) string {

	arr := strings.Split(s, "_")
	var result string
	if len(arr) > 0 {
		for _, v := range arr {
			result += UcFirst(v)
		}
	}

	return result
}

// 第一个单词首字母变大写
func UcFirst(str string) string {
	for i, v := range str {
		return string(unicode.ToUpper(v)) + str[i+1:]
	}
	return ""
}

// 第一个单词首字母变小写
func LcFirst(str string) string {
	for i, v := range str {
		return string(unicode.ToLower(v)) + str[i+1:]
	}
	return ""
}
