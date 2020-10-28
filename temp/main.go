package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"reflect"
	"strings"
)

const Delimiter = ","
const ColumnTag = "column"
const PrimaryTag = "primary"
const And = " and "
const SelectTemplate = "select %s from %s where %s %s %s"
const InsertTemplate = "insert into %s (%s) values %s"
const UpdateTemplate = "update %s set %s where %s"
const DeleteTemplate = "delete from %s where %s"

type orm struct {
	db          *sql.DB
	table       string
	order       string
	limit       string
	where       string
	whereParams []interface{}
	err         error
}

func InitOrm(db *sql.DB, table string) *orm {
	o := &orm{
		db:          db,
		table:       table,
		where:       "1=1",
		whereParams: make([]interface{}, 0),
	}
	return o
}

func (o *orm) Order(orderBy string) *orm {
	o.order = orderBy
	return o
}
func (o *orm) Limit(limit uint32) *orm {
	o.limit = fmt.Sprintf("limit %d", limit)
	return o
}
func (o *orm) Page(page uint32, pageSize uint32) *orm {
	o.limit = fmt.Sprintf("limit %d,%d", page*pageSize, page*(pageSize+1))
	return o
}
func (o *orm) Condition(condition interface{}) *orm {
	if condition == nil {
		o.err = errors.New("condition is nil")
		return o
	}
	keys, values, err := GetKV(reflect.ValueOf(condition))
	if err != nil {
		o.err = err
		return o
	}
	for i := 0; i < len(keys); i++ {
		o.where = o.where + keys[i] + "=?,"
		o.whereParams = append(o.whereParams, values[i])
	}
	o.where = o.where[0 : len(o.where)-1]
	return o
}

//Select dest must be a ptr, e.g. *user, *[]user
func (o *orm) Select(ctx context.Context, dest interface{}) (err error) {
	if o.err != nil {
		return o.err
	}
	if dest == nil {
		err = errors.New("dest is nil")
		return err
	}
	t := reflect.TypeOf(dest)
	v := reflect.ValueOf(dest)
	if t.Kind() != reflect.Ptr || !v.Elem().CanAddr() {
		err = errors.New("only support param like *user, *[]user")
		return err
	}
	switch t.Elem().Kind() {
	case reflect.Struct:
		err = o.selectSingle(ctx, v)
	case reflect.Slice:
		err = o.selectMultiple(ctx, v)
	default:
		err = errors.New("only support param like *user, *[]user")
	}
	return err
}

func (o *orm) selectSingle(ctx context.Context, v reflect.Value) (err error) {
	o.limit = "limit 1"
	columns, address := ColumnAndAddress(v)
	if columns == nil || len(columns) == 0 {
		err = errors.New("parse empty columns")
		return err
	}
	if address == nil || len(address) == 0 {
		err = errors.New("parse empty address")
		return err
	}
	sqlStr := fmt.Sprintf(SelectTemplate, strings.Join(columns, Delimiter), o.table, o.where, o.order, o.limit)
	fmt.Println(sqlStr)
	err = o.db.QueryRowContext(ctx, sqlStr, o.whereParams...).Scan(address)
	return err
}
func (o *orm) selectMultiple(ctx context.Context, v reflect.Value) (err error) {
	//每个数组成员的类型
	ct := v.Type().Elem()
	//创建一个空的struct
	cv := reflect.New(ct)
	columns, _ := ColumnAndAddress(cv)
	if columns == nil || len(columns) == 0 {
		err = errors.New("parse empty columns")
		return err
	}
	sqlStr := fmt.Sprintf(SelectTemplate, strings.Join(columns, Delimiter), o.table, o.where, o.order, o.limit)
	fmt.Println(sqlStr)
	rows, err := o.db.QueryContext(ctx, sqlStr, o.whereParams...)
	if err != nil {
		return err
	}
	// 创建一个新数组并把元素的值追加进去
	newArr := make([]reflect.Value, 0)
	for rows.Next() {
		temp := reflect.New(ct)
		_, tempAddress := ColumnAndAddress(temp)
		err = rows.Scan(tempAddress...)
		if err != nil {
			return err
		}
		newArr = append(newArr, temp)
	}
	//和旧的生成一个新的
	resArr := reflect.Append(v, newArr...)
	v.Set(resArr)
	return nil
}

//Insert in can be User, *User, []User, []*User, map[string]interface{}, returns lastId
func (o *orm) Insert(ctx context.Context, obj interface{}) (id int64, err error) {
	v := reflect.ValueOf(obj)
	//剥离指针
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	var keys []string
	var values []interface{}
	valueLength := 0
	switch v.Kind() {
	case reflect.Struct, reflect.Map:
		keys, values, err = GetKV(v)
		if err != nil {
			return 0, err
		}
	case reflect.Slice:
		valueLength = v.Len()
		for i := 0; i < v.Len(); i++ {
			//Kind是切片时，可以用Index()方法遍历
			sv := v.Index(i)
			//剥离指针
			for sv.Kind() == reflect.Ptr {
				sv = sv.Elem()
			}
			//切片元素不是struct或者map，报错
			if sv.Kind() != reflect.Struct || sv.Kind() != reflect.Map {
				return 0, errors.New("insert slice element must be struct or map")
			}
			//keys只保存一次就行，因为后面的都一样了
			tempKeys, tempValues, err := GetKV(sv)
			if err != nil {
				return 0, err
			}
			if len(keys) == 0 {
				keys = tempKeys
			}
			values = append(values, tempValues...)
		}
	default:
		return 0, errors.New("method Insert error: type error")
	}

	kl := len(keys)
	vl := len(values)
	if kl == 0 || vl == 0 {
		return 0, errors.New("method Insert error: no data")
	}
	var valueStr string
	s1 := strings.Repeat("?,", kl)
	s1 = s1[0 : len(s1)-1]
	if valueLength == 0 {
		valueStr = fmt.Sprintf("(%s)", s1)
	} else {
		valueStr = strings.Repeat(fmt.Sprintf("(%s),", s1), valueLength)
		valueStr = valueStr[0 : len(s1)-1]
	}
	query := fmt.Sprintf(InsertTemplate, o.table, strings.Join(keys, ","), valueStr)
	fmt.Printf("insert query: %s", query)
	result, err := o.db.ExecContext(ctx, query, values...)
	if err != nil {
		return 0, err
	}
	id, err = result.LastInsertId()
	return
}

//Update in can be User, *User, map[string]interface{}, return
func (o *orm) Update(ctx context.Context, obj interface{}) (rowsAffected int64, err error) {
	v := reflect.ValueOf(obj)
	//剥离指针
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	keys, values, err := GetKV(v)
	if err != nil {
		return 0, err
	}
	if len(keys) == 0 {
		err = errors.New("empty values to set")
		return 0, err
	}
	setStr := strings.Join(keys, "=?,")
	setStr = setStr[0 : len(setStr)-1]
	query := fmt.Sprintf(UpdateTemplate, o.table, setStr, o.where)
	fmt.Printf("insert query: %s", query)
	values = append(values, o.whereParams...)
	result, err := o.db.ExecContext(ctx, query, values...)
	if err != nil {
		return 0, err
	}
	rowsAffected, err = result.RowsAffected()
	return
}

//Delete, return rows affected
func (o *orm) Delete(ctx context.Context) (id int64, err error) {
	query := fmt.Sprintf(DeleteTemplate, o.table, o.where)
	fmt.Printf("delete sql: %s \n", query)
	result, err := o.db.ExecContext(ctx, query, o.whereParams...)
	if err != nil {
		return 0, err
	}
	id, err = result.RowsAffected()
	return
}
func ColumnAndAddress(v reflect.Value) (columns []string, address []interface{}) {
	fmt.Println(v)
	dest := v.Elem()
	fmt.Println(dest)
	t := dest.Type()
	columns = make([]string, 0)
	address = make([]interface{}, 0)
	for n := 0; n < t.NumField(); n++ {
		tf := t.Field(n)
		vf := dest.Field(n)
		if tf.Anonymous {
			continue
		}
		fmt.Println(tf)
		fmt.Println(vf)
		for vf.Type().Kind() == reflect.Ptr {
			vf = vf.Elem()
		}
		fmt.Println(v)
		address = append(address, vf.Addr().Interface())
		columns = append(columns, tf.Name)
	}
	return
}

func GetKV(v reflect.Value) (keys []string, values []interface{}, err error) {
	t := v.Type()
	switch t.Kind() {
	case reflect.Struct:
		keys, values = struct2KV(v)
		return
	case reflect.Map:
		mapKeys := v.MapKeys()
		for _, key := range mapKeys {
			value := v.MapIndex(key)
			if !value.IsValid() || reflect.DeepEqual(value.Interface(), reflect.Zero(value.Type()).Interface()) {
				continue
			}
			values = append(values, value)
			keys = append(keys, key.Interface().(string))
		}
		return
	default:
		err = errors.New("unsupported type, only struct and map[string]interface{} is supported")
		return
	}
}

func struct2KV(v reflect.Value) (keys []string, values []interface{}) {
	t := v.Type()
	for n := 0; n < t.NumField(); n++ {
		tf := t.Field(n)
		vf := v.Field(n)
		//忽略非导出字段
		if tf.Anonymous {
			continue
		}
		//忽略无效、零值字段
		if !vf.IsValid() || reflect.DeepEqual(vf.Interface(), reflect.Zero(vf.Type()).Interface()) {
			continue
		}
		for vf.Type().Kind() == reflect.Ptr {
			vf = vf.Elem()
		}
		//优先使用tag
		key := tf.Tag.Get(ColumnTag)
		//没有根据变量名转小写snake
		if key == "" {
			key = strings.ToLower(ToSnake(tf.Name))
		}
		keys = append(keys, key)
		values = append(values, vf.Interface())
	}
	return
}

// ToSnake converts a string to snake_case
func ToSnake(s string) string {
	return ToDelimited(s, '_')
}

func ToSnakeWithIgnore(s string, ignore uint8) string {
	return ToScreamingDelimited(s, '_', ignore, false)
}

// ToScreamingSnake converts a string to SCREAMING_SNAKE_CASE
func ToScreamingSnake(s string) string {
	return ToScreamingDelimited(s, '_', 0, true)
}

// ToKebab converts a string to kebab-case
func ToKebab(s string) string {
	return ToDelimited(s, '-')
}

// ToScreamingKebab converts a string to SCREAMING-KEBAB-CASE
func ToScreamingKebab(s string) string {
	return ToScreamingDelimited(s, '-', 0, true)
}

// ToDelimited converts a string to delimited.snake.case
// (in this case `delimiter = '.'`)
func ToDelimited(s string, delimiter uint8) string {
	return ToScreamingDelimited(s, delimiter, 0, false)
}

// ToScreamingDelimited converts a string to SCREAMING.DELIMITED.SNAKE.CASE
// (in this case `delimiter = '.'; screaming = true`)
// or delimited.snake.case
// (in this case `delimiter = '.'; screaming = false`)
func ToScreamingDelimited(s string, delimiter uint8, ignore uint8, screaming bool) string {
	s = strings.TrimSpace(s)
	n := strings.Builder{}
	n.Grow(len(s) + 2) // nominal 2 bytes of extra space for inserted delimiters
	for i, v := range []byte(s) {
		vIsCap := v >= 'A' && v <= 'Z'
		vIsLow := v >= 'a' && v <= 'z'
		if vIsLow && screaming {
			v += 'A'
			v -= 'a'
		} else if vIsCap && !screaming {
			v += 'a'
			v -= 'A'
		}

		// treat acronyms as words, eg for JSONData -> JSON is a whole word
		if i+1 < len(s) {
			next := s[i+1]
			vIsNum := v >= '0' && v <= '9'
			nextIsCap := next >= 'A' && next <= 'Z'
			nextIsLow := next >= 'a' && next <= 'z'
			nextIsNum := next >= '0' && next <= '9'
			// add underscore if next letter case type is changed
			if (vIsCap && (nextIsLow || nextIsNum)) || (vIsLow && (nextIsCap || nextIsNum)) || (vIsNum && (nextIsCap || nextIsLow)) {
				if prevIgnore := ignore > 0 && i > 0 && s[i-1] == ignore; !prevIgnore {
					if vIsCap && nextIsLow {
						if prevIsCap := i > 0 && s[i-1] >= 'A' && s[i-1] <= 'Z'; prevIsCap {
							n.WriteByte(delimiter)
						}
					}
					n.WriteByte(v)
					if vIsLow || vIsNum || nextIsNum {
						n.WriteByte(delimiter)
					}
					continue
				}
			}
		}

		if (v == ' ' || v == '_' || v == '-') && uint8(v) != ignore {
			// replace space/underscore/hyphen with delimiter
			n.WriteByte(delimiter)
		} else {
			n.WriteByte(v)
		}
	}

	return n.String()
}

func AddAlias(columns []string, alias string) []string {
	result := make([]string, len(columns))
	for i := 0; i < len(columns); i++ {
		result[i] = alias + "." + columns[i]
	}
	return result
}
func GetPlaceholders(columns []string) string {
	s := strings.Repeat("?,", len(columns))
	s = s[0 : len(s)-1]
	return s
}

type Env struct {
	Id         uint32
	EnvId      string
	Region     string
	status     uint32
	CreateTime string
	UpdateTime string
}

func main() {
	ctx := context.TODO()
	db, err := sql.Open("mysql", "root:8B8a7PzDv@@tcp(10.236.158.96:19261)/tcb_dev?charset=utf8")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	env := &Env{}
	err = InitOrm(db, "qbase_env").Select(ctx, env)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
}
