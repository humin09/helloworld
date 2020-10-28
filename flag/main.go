package main

import (
	"fmt"
	"os"
	"reflect"
)

type User struct {
	Id   int
	Name string
}

func ColumnAndAddress(v reflect.Value) (columns []string, address []interface{}) {
	dest := v.Elem()
	t := dest.Type()
	columns = make([]string, 0)
	address = make([]interface{}, 0)
	for n := 0; n < t.NumField(); n++ {
		tf := t.Field(n)
		vf := dest.Field(n)
		if tf.Anonymous {
			continue
		}
		for vf.Type().Kind() == reflect.Ptr {
			vf = vf.Elem()
		}
		address = append(address, vf.Addr().Interface())
		columns = append(columns, tf.Name)
	}
	return
}
func ChangeSlice(s interface{}) {
	sT := reflect.TypeOf(s)
	if sT.Kind() != reflect.Ptr {
		fmt.Println("参数必须是ptr类型")
		os.Exit(-1)
	}
	sV := reflect.ValueOf(s)
	fmt.Println(sT.Elem())
	// 取得数组中元素的类型
	sEE := sT.Elem().Elem()
	fmt.Println(sEE)
	// 数组的值
	sVE := sV.Elem()

	// new一个数组中的元素对象
	sON := reflect.New(sEE)
	fmt.Println(sON)
	// 对象的值
	sONE := sON.Elem()
	fmt.Println(sONE)
	cols, adress := ColumnAndAddress(sON)
	fmt.Println(cols)
	fmt.Println(adress)
	// 给对象复制
	sONEId := sONE.FieldByName("Id")
	sONEName := sONE.FieldByName("Name")
	sONEId.SetInt(10)
	sONEName.SetString("李四")

	// 创建一个新数组并把元素的值追加进去
	//newArr := make([]reflect.Value, 0)
	//newArr = append(newArr, sON.Elem())

	// 把原数组的值和新的数组合并
	b := reflect.Append(sVE, sON.Elem())
	a := reflect.Append(b, sON.Elem())

	// 最终结果给原数组
	sVE.Set(a)
}

func main() {
	users := make([]User, 0)
	ChangeSlice(&users)
	// 这里希望让Users指向ChangeSlice函数中的那个新数组
	fmt.Println(users)
}
