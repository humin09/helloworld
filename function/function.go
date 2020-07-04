package function

import (
	"encoding/json"
	"fmt"
)

func NewStudent() (stu Student) {
	stu.Age=18
	stu.Name="humin"
	b,_:=json.Marshal(stu)
	fmt.Println(string(b))
	return stu
}
func print(name string,ver int) string {
	return fmt.Sprintf("%s-%03d", name, ver)
}


