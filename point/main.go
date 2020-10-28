package main

import (
	"encoding/json"
	"fmt"
)

type Student struct {
	Name string
	Age  int
}

func main() {
	s := &Student{
		Name: "humin",
		Age:  36,
	}
	bs, _ := json.Marshal(&s)
	fmt.Println(string(bs))
}
