package main

import "fmt"

type Student struct {
	Name string
	Age  uint
}

func main() {
	s := make(map[string]Student, 0)
	s["hello"] = Student{
		Name: "hello",
	}
	s["world"] = Student{
		Name: "world",
	}
	for k, v := range s {

		v.Name = v.Name + "===="
		s[k] = v
	}
	fmt.Println(s)
}
