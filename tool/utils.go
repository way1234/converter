package tool

import "strings"

// 返回指定数量的"\t"字符串, 例如： count=3，返回\t\t\t
func Tab(count int) string {
	return strings.Repeat("\t", count)
}
