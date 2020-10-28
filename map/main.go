package main

import "fmt"

func main() {
	s := []string{"name", "age"}
	AddAlias(s, "a")
	fmt.Println(s)
	k := ""
	fmt.Println(len(k))

}
func AddAlias(columns []string, alias string) {
	for i := 0; i < len(columns); i++ {
		columns[i] = alias + "." + columns[i]
	}
	return
}
