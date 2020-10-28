package main

import (
	"fmt"
	"reflect"
	"time"
)

type Foo struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Age       int    `json:"age"`
}

type Bar struct {
	Age    int `json:"age"`
	Score  int `json:"score"`
	Gender int `json:"gender"`
}

type ProjectQueryCondition struct {
	Id         int32  `column:"id"`
	Name       string `column:"name"`
	EnvId      string `column:"env_id"`
	VersionNum int32  `column:"version_num"`
}

func (f *Foo) reflect() {
	val := reflect.ValueOf(f).Elem()

	for i := 0; i < val.NumField(); i++ {
		valueField := val.Field(i)
		typeField := val.Type().Field(i)
		tag := typeField.Tag

		fmt.Printf("Field Name: %s,\t Field Value: %v,\t Tag Value: %s\n", typeField.Name, valueField.Interface(), tag.Get("json"))
	}
}

func GetNotEmptyColumns(obj interface{}) (columns []string, values []interface{}) {
	if obj == nil {
		return nil, nil
	}
	val := reflect.ValueOf(obj).Elem()
	length := val.NumField()
	columns = make([]string, 0)
	values = make([]interface{}, 0)
	for i := 0; i < length; i++ {
		valueField := val.Field(i)
		value := valueField.Interface()
		isEmpty := false
		switch v := value.(type) {
		case int:
			isEmpty = v == 0
		case int32:
			isEmpty = v == 0
		case int64:
			isEmpty = v == 0
		case string:
			isEmpty = len(v) == 0
		}
		if isEmpty {
			continue
		}
		values = append(values, value)
		typeField := val.Type().Field(i)
		tag := typeField.Tag
		columns = append(columns, tag.Get("column")+"=?")
	}
	return
}

func SetAll(ins ...interface{}) {
	for i := 0; i < len(ins); i++ {
		ins[i] = 1
	}
}
func main() {
	ticker := time.NewTicker(2 * time.Second)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				fmt.Println("poll extension status timeout")
				return
			case <-ticker.C:
				fmt.Println("start polling extension status")

			}
		}
	}()
	time.Sleep(10 * time.Second)
	ticker.Stop()
}
