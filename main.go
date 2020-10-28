package main

import (
	"fmt"
	"strings"
)

type name struct {
	hellos []string
}

func main() {
	n := &name{}
	hs := make([]string, 0)
	n.hellos = hs
	hs = append(hs, "hello")
	n.hellos = hs
	fmt.Println(n)

	s := "hello/world"
	fmt.Println(s[0:strings.Index(s, "/")])
}
