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
	tx          *sql.Tx
	table       string
	order       string
	limit       string
	where       string
	whereParams []interface{}
	err         error
}

func InitOrm(db *sql.DB, table string) *orm {
	o := &orm{
		db:    db,
		table: table,
	}
	o.reset()
	return o
}

func (o *orm) Tx(tx *sql.Tx) *orm {
	o.tx = tx
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
func (o *orm) Page(pageNo uint32, pageSize uint32) *orm {
	o.limit = fmt.Sprintf("limit %d,%d", pageNo*pageSize, (pageNo+1)*pageSize)
	return o
}

//Condition support User{} &User{} map[string]interface{}, most field or kv will change to  where a=? and b=? while if field or key is slice will change to where a in (?,?,?)
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
	o.where = ""
	for i := 0; i < len(keys); i++ {
		v := reflect.ValueOf(values[i])
		if v.Kind() == reflect.Slice {
			l := v.Len()
			if l == 0 {
				continue
			}
			placeHolder := strings.Repeat("?,", l)
			placeHolder = placeHolder[0 : len(placeHolder)-1]
			keys[i] = keys[i] + fmt.Sprintf(" in (%s) ", placeHolder)
			for j := 0; j < l; j++ {
				o.whereParams = append(o.whereParams, v.Index(j).Interface())
			}
		} else {
			keys[i] = keys[i] + " = ? "
			o.whereParams = append(o.whereParams, values[i])
		}
	}
	o.where = strings.Join(keys, " and ")
	return o
}
func (o *orm) reset() {
	o.whereParams = make([]interface{}, 0)
	o.where = "1=1"
}

//Select dest must be a ptr, e.g. *user, *[]user
func (o *orm) Select(ctx context.Context, dest interface{}) (err error) {
	defer o.reset()
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
	if v.Kind() != reflect.Struct {
		err = errors.New("selectSingle only support *User")
	}
	if !v.CanSet() {
		err = errors.New(fmt.Sprintf("selectSingle reflect value %v can't set", v))
	}
	columns, address, err := ColumnAndAddress(v)
	if err != nil {
		return err
	}
	if columns == nil || len(columns) == 0 {
		err = errors.New("parse empty columns")
		return err
	}
	if address == nil || len(address) == 0 {
		err = errors.New("parse empty address")
		return err
	}
	query := fmt.Sprintf(SelectTemplate, strings.Join(columns, Delimiter), o.table, o.where, o.order, o.limit)
	if o.tx == nil {
		query = query + "for update"
		fmt.Printf("sql: %s", query)
		err = o.db.QueryRowContext(ctx, query, o.whereParams...).Scan(address...)
	} else {
		fmt.Printf("sql: %s", query)
		err = o.tx.QueryRowContext(ctx, query, o.whereParams...).Scan(address...)
	}
	return err
}
func (o *orm) selectMultiple(ctx context.Context, v reflect.Value) (err error) {
	if !v.CanSet() {
		err = errors.New(fmt.Sprintf("selectMultiple reflect value %v can't set", v))
	}
	//每个数组成员的类型
	ct := v.Type().Elem().Elem()
	if ct.Kind() != reflect.Struct {
		err = errors.New("selectMultiple only support *[]User, not *[]*User")
	}
	//创建一个空的struct
	cv := reflect.New(ct)
	columns, _, err := ColumnAndAddress(cv)
	if err != nil {
		return err
	}
	if columns == nil || len(columns) == 0 {
		err = errors.New("parse empty columns")
		return err
	}
	query := fmt.Sprintf(SelectTemplate, strings.Join(columns, Delimiter), o.table, o.where, o.order, o.limit)

	var rows *sql.Rows
	if o.tx == nil {
		fmt.Printf("sql: %s", query)
		rows, err = o.db.QueryContext(ctx, query, o.whereParams...)
	} else {
		query = query + "for update"
		fmt.Printf("sql: %s", query)
		rows, err = o.tx.QueryContext(ctx, query, o.whereParams...)
	}
	if err != nil {
		return err
	}
	// 创建一个新数组并把元素的值追加进去
	newArr := make([]reflect.Value, 0)
	for rows.Next() {
		temp := reflect.New(ct).Elem()
		_, tempAddress, err := ColumnAndAddress(temp)
		if err != nil {
			return err
		}
		err = rows.Scan(tempAddress...)
		if err != nil {
			return err
		}
		newArr = append(newArr, temp)
	}
	//和旧的生成一个新的
	resArr := reflect.Append(v.Elem(), newArr...)
	v.Elem().Set(resArr)
	return nil
}

//Insert in can be User, *User, []User, []*User, map[string]interface{}, returns lastId
func (o *orm) Insert(ctx context.Context, obj interface{}) (id int64, err error) {
	defer o.reset()
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
	fmt.Printf("sql: %s", query)
	var result sql.Result
	if o.tx == nil {
		result, err = o.db.ExecContext(ctx, query, values...)
	} else {
		result, err = o.tx.ExecContext(ctx, query, values...)
	}

	if err != nil {
		return 0, err
	}
	id, err = result.LastInsertId()
	return
}

//Update in can be User, *User, map[string]interface{}, return
func (o *orm) Update(ctx context.Context, obj interface{}) (rowsAffected int64, err error) {
	defer o.reset()
	v := reflect.ValueOf(obj)
	keys, values, err := GetKV(v)
	if err != nil {
		return 0, err
	}
	if len(keys) == 0 {
		err = errors.New("empty values to set")
		return 0, err
	}
	setStr := strings.Join(keys, "=?,")
	setStr = setStr + "=?"
	query := fmt.Sprintf(UpdateTemplate, o.table, setStr, o.where)
	fmt.Printf("sql: %s", query)
	values = append(values, o.whereParams...)
	var result sql.Result
	if o.tx == nil {
		result, err = o.db.ExecContext(ctx, query, values...)
	} else {
		result, err = o.tx.ExecContext(ctx, query, values...)
	}
	if err != nil {
		return 0, err
	}
	rowsAffected, err = result.RowsAffected()
	return
}

//Delete, return rows affected
func (o *orm) Delete(ctx context.Context) (id int64, err error) {
	defer o.reset()
	query := fmt.Sprintf(DeleteTemplate, o.table, o.where)
	fmt.Printf("sql: %s", query)
	var result sql.Result
	if o.tx == nil {
		result, err = o.db.ExecContext(ctx, query, o.whereParams...)
	} else {
		result, err = o.tx.ExecContext(ctx, query, o.whereParams...)
	}
	if err != nil {
		return 0, err
	}
	id, err = result.RowsAffected()
	return
}

//ColumnAndAddress 给定一个*User, 返回他的所有可访问的Field和其成员的指针,
func ColumnAndAddress(v reflect.Value) (columns []string, address []interface{}, err error) {
	//去掉指针
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()
	columns = make([]string, 0)
	address = make([]interface{}, 0)
	for n := 0; n < t.NumField(); n++ {
		tf := t.Field(n)
		vf := v.Field(n)
		//忽略非导出字段
		if tf.Anonymous || !vf.CanInterface() {
			continue
		}
		for vf.Type().Kind() == reflect.Ptr {
			vf = vf.Elem()
		}
		if vf.CanAddr() && vf.Addr().CanInterface() {
			address = append(address, vf.Addr().Interface())
			//优先使用tag
			key := tf.Tag.Get(ColumnTag)
			//没有根据变量名转小写snake
			if key == "" {
				key = strings.ToLower(ToSnake(tf.Name))
			}
			columns = append(columns, key)
		}
	}
	return
}

//GetKV 可以输入User,*User, Map, *Map, 会拿取里面的值不为空的Field/Key和Value, 并且把Key从驼峰转下划线即: AngelaBaby转为angela_baby
func GetKV(v reflect.Value) (keys []string, values []interface{}, err error) {
	//剥离指针
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()
	switch t.Kind() {
	case reflect.Struct:
		keys, values = struct2KV(v)
		return
	case reflect.Map:
		m := v.Interface().(map[string]interface{})
		for key, value := range m {
			key = strings.ToLower(ToSnake(key))
			keys = append(keys, key)
			values = append(values, value)
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
		if tf.Anonymous || !vf.CanInterface() {
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

//运行事务
func RunTx(ctx context.Context, db *sql.DB, dbFunc func(*sql.Tx) error) error {
	tx, errTx := db.BeginTx(ctx, nil)
	if errTx != nil {
		return fmt.Errorf("db transaction begin fail")
	}

	err := dbFunc(tx)
	if err != nil {
		errTx = tx.Rollback()
		if errTx != nil {
			return fmt.Errorf("db transaction rollback fail")
		}
		return err
	} else {
		err = tx.Commit()
	}
	return err
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
	Status     uint32
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
	o := InitOrm(db, "qbase_env")
	err = o.Select(ctx, env)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Printf("query1 env success, %v\n", *env)
	err = o.Condition(map[string]interface{}{
		"Id":     1874,
		"Region": "ap-shanghai",
	}).Select(ctx, env)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Printf("query2 env success, %v\n", *env)

	err = o.Condition(struct{ Id int }{1874}).Select(ctx, env)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Printf("query3 env success, %v\n", *env)
	envs := make([]Env, 0)
	err = o.Condition(map[string]interface{}{
		"Region": "ap-shanghai",
	}).Page(0, 10).Order("order by id desc").Select(ctx, &envs)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Printf("query4 env success, %v\n", envs)

	envs = make([]Env, 0)
	err = o.Condition(map[string]interface{}{
		"Id":     []int{1874, 1878},
		"Region": "ap-shanghai",
	}).Page(0, 10).Order("order by id desc").Select(ctx, &envs)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Printf("query5 env success, %v\n", envs)

	id, err := o.Insert(ctx, Env{
		EnvId:  "helloworld",
		Region: "huoxin",
	})
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Printf("insert env success,id is %v\n", id)

	affectRows, err := o.Condition(Env{
		EnvId:  "helloworld",
		Region: "huoxin",
	}).Update(ctx, Env{
		Status: 1,
		Region: "huoxin",
	})
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Printf("update env success,affectRows is %v\n", affectRows)

	affectRows, err = o.Condition(Env{
		EnvId:  "helloworld",
		Region: "huoxin",
	}).Delete(ctx)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Printf("delete env success,affectRows is %v\n", affectRows)

}
