package orm

import (
	"database/sql"
	"errors"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type Orm struct {
	Db           *sql.DB
	FieldParam   string
	TableName    string
	WhereParam   string
	OrWhereParam string
	WhereExec    []interface{}
	GroupParam   string
	HavingParam  string
	OrderParam   string
	LimitParam   string
	Prepare      string
	AllExec      []interface{}
	Sql          string
	UpdateParam  string
	UpdateExec   []interface{}
	Tx           *sql.Tx
	TransStatus  int
}

// 新建Mysql连接
func NewMysql(Username string, Password string, Address string, Dbname string) (*Orm, error) {
	dsn := Username + ":" + Password + "@tcp(" + Address + ")/" + Dbname + "?charset=utf8&timeout=5s&readTimeout=6s"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	//最大连接数等配置，先占个位
	//db.SetMaxOpenConns(3)
	//db.SetMaxIdleConns(3)

	return &Orm{
		Db:         db,
		FieldParam: "*",
	}, nil
}

// 设置表名
func (e *Orm) Table(name string) *Orm {
	e.TableName = name
	// e.resetOrm()
	return e
}

// 获取表名
func (e *Orm) GetTable() string {
	return e.TableName
}

func (e *Orm) doInsert(batchData interface{}, insertType string) (int64, error) {
	//反射解析
	getValue := reflect.ValueOf(batchData)

	//切片大小
	l := getValue.Len()

	//字段名
	var fieldName []string

	//占位符
	var placeholderString []string

	//循环判断
	for i := 0; i < l; i++ {
		value := getValue.Index(i) // Value of item
		typed := value.Type()      // Type of item
		if typed.Kind() != reflect.Struct {
			panic("批量插入的子元素必须是结构体类型")
		}

		num := value.NumField()

		//子元素值
		var placeholder []string
		//循环遍历子元素
		for j := 0; j < num; j++ {

			//小写开头，无法反射，跳过
			if !value.Field(j).CanInterface() {
				continue
			}

			//解析tag，找出真实的sql字段名
			sqlTag := typed.Field(j).Tag.Get("sql")
			if sqlTag != "" {
				//跳过自增字段
				if strings.Contains(strings.ToLower(sqlTag), "auto_increment") {
					continue
				} else {
					//字段名只记录第一个的
					if i == 1 {
						fieldName = append(fieldName, strings.Split(sqlTag, ",")[0])
					}
					placeholder = append(placeholder, "?")
				}
			} else {
				//字段名只记录第一个的
				if i == 1 {
					fieldName = append(fieldName, typed.Field(j).Name)
				}
				placeholder = append(placeholder, "?")
			}

			//字段值
			e.AllExec = append(e.AllExec, value.Field(j).Interface())
		}

		//子元素拼接成多个()括号后的值
		placeholderString = append(placeholderString, "("+strings.Join(placeholder, ",")+")")
	}

	//拼接表，字段名，占位符
	e.Prepare = insertType + " into " + e.GetTable() + " (" + strings.Join(fieldName, ",") + ") values " + strings.Join(placeholderString, ",")

	//prepare
	var stmt *sql.Stmt
	var err error
	stmt, err = e.Db.Prepare(e.Prepare)
	if err != nil {
		return 0, e.setErrorInfo(err)
	}

	//执行exec,注意这是stmt.Exec
	result, err := stmt.Exec(e.AllExec...)
	if err != nil {
		return 0, e.setErrorInfo(err)
	}

	//获取自增ID
	id, _ := result.LastInsertId()
	return id, nil
}

// 自定义错误格式
func (e *Orm) setErrorInfo(err error) error {
	_, file, line, _ := runtime.Caller(1)
	return errors.New("File: " + file + ":" + strconv.Itoa(line) + ", " + err.Error())
}

// 插入
func (e *Orm) Insert(data interface{}) (int64, error) {
	//判断是批量还是单个插入
	getValue := reflect.ValueOf(data).Kind()
	if getValue == reflect.Struct {
		return e.doInsert([]any{data}, "insert")
	} else if getValue == reflect.Slice || getValue == reflect.Array {
		return e.doInsert(data, "insert")
	} else {
		return 0, errors.New("插入的数据格式不正确，单个插入格式为: struct，批量插入格式为: []struct")
	}
}
func (e *Orm) Replace(data interface{}) (int64, error) {
	//判断是批量还是单个插入
	getValue := reflect.ValueOf(data).Kind()
	if getValue == reflect.Struct {
		return e.doInsert([]any{data}, "replace")
	} else if getValue == reflect.Slice || getValue == reflect.Array {
		return e.doInsert(data, "replace")
	} else {
		return 0, errors.New("插入的数据格式不正确，单个插入格式为: struct，批量插入格式为: []struct")
	}
}
func (e *Orm) Where(data ...interface{}) *Orm {
	e.doWhere(data, "and")
}
func (e *Orm) OrWhere(data ...interface{}) *Orm {
	e.doWhere(data, "or")
}
func (e *Orm) doWhere(data ...interface{}, whereType string) *Orm {
	//判断使用顺序
	if whereType == "or" && e.WhereParam == "" {
		panic("WhereOr必须在Where后面调用")
	}

	//判断是结构体还是多个字符串
	var dataType int
	if len(data) == 1 {
		dataType = 1
	} else if len(data) == 2 {
		dataType = 2
	} else if len(data) == 3 {
		dataType = 3
	} else {
		panic("参数个数错误")
	}

	//多次调用判断
	if e.WhereParam != "" {
		e.WhereParam += " " + whereType + " ("
	} else {
		e.WhereParam += "("
	}

	//如果是结构体
	if dataType == 1 {
		t := reflect.TypeOf(data[0])
		v := reflect.ValueOf(data[0])

		//字段名
		var fieldNameArray []string

		//循环解析
		for i := 0; i < t.NumField(); i++ {

			//首字母小写，不可反射
			if !v.Field(i).CanInterface() {
				continue
			}

			//解析tag，找出真实的sql字段名
			sqlTag := t.Field(i).Tag.Get("sql")
			if sqlTag != "" {
				fieldNameArray = append(fieldNameArray, strings.Split(sqlTag, ",")[0]+"=?")
			} else {
				fieldNameArray = append(fieldNameArray, t.Field(i).Name+"=?")
			}

			e.WhereExec = append(e.WhereExec, v.Field(i).Interface())
		}

		//拼接
		e.WhereParam += strings.Join(fieldNameArray, " and ") + ") "

	} else if dataType == 2 {
		//直接=的情况
		e.WhereParam += data[0].(string) + "=?) "
		e.WhereExec = append(e.WhereExec, data[1])
	} else if dataType == 3 {
		//3个参数的情况

		//区分是操作符in的情况
		data2 := strings.Trim(strings.ToLower(data[1].(string)), " ")
		if data2 == "in" || data2 == "not in" {
			//判断传入的是切片
			reType := reflect.TypeOf(data[2]).Kind()
			if reType != reflect.Slice && reType != reflect.Array {
				panic("in/not in 操作传入的数据必须是切片或者数组")
			}

			//反射值
			v := reflect.ValueOf(data[2])
			//数组/切片长度
			dataNum := v.Len()
			//占位符
			ps := make([]string, dataNum)
			for i := 0; i < dataNum; i++ {
				ps[i] = "?"
				e.WhereExec = append(e.WhereExec, v.Index(i).Interface())
			}

			//拼接
			e.WhereParam += data[0].(string) + " " + data2 + " (" + strings.Join(ps, ",") + ")) "

		} else {
			e.WhereParam += data[0].(string) + " " + data[1].(string) + " ?) "
			e.WhereExec = append(e.WhereExec, data[2])
		}
	}

	return e
}

// 删除
func (e *Orm) Delete() (int64, error) {

	//拼接delete sql
	e.Prepare = "delete from " + e.GetTable()

	//如果where不为空
	if e.WhereParam != "" || e.OrWhereParam != "" {
		e.Prepare += " where " + e.WhereParam + e.OrWhereParam
	}

	//limit不为空
	if e.LimitParam != "" {
		e.Prepare += "limit " + e.LimitParam
	}

	//第一步：Prepare
	var stmt *sql.Stmt
	var err error
	stmt, err = e.Db.Prepare(e.Prepare)
	if err != nil {
		return 0, err
	}

	e.AllExec = e.WhereExec

	//第二步：执行exec,注意这是stmt.Exec
	result, err := stmt.Exec(e.AllExec...)
	if err != nil {
		return 0, e.setErrorInfo(err)
	}

	//影响的行数
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, e.setErrorInfo(err)
	}

	return rowsAffected, nil
}

// 更新
func (e *Orm) Update(data ...interface{}) (int64, error) {

	//判断是结构体还是多个字符串
	var dataType int
	if len(data) == 1 {
		dataType = 1
	} else if len(data) == 2 {
		dataType = 2
	} else {
		return 0, errors.New("参数个数错误")
	}

	//如果是结构体
	if dataType == 1 {
		t := reflect.TypeOf(data[0])
		v := reflect.ValueOf(data[0])

		var fieldNameArray []string
		for i := 0; i < t.NumField(); i++ {

			//首字母小写，不可反射
			if !v.Field(i).CanInterface() {
				continue
			}

			//解析tag，找出真实的sql字段名
			sqlTag := t.Field(i).Tag.Get("sql")
			if sqlTag != "" {
				fieldNameArray = append(fieldNameArray, strings.Split(sqlTag, ",")[0]+"=?")
			} else {
				fieldNameArray = append(fieldNameArray, t.Field(i).Name+"=?")
			}

			e.UpdateExec = append(e.UpdateExec, v.Field(i).Interface())
		}
		e.UpdateParam += strings.Join(fieldNameArray, ",")

	} else if dataType == 2 {
		//直接=的情况
		e.UpdateParam += data[0].(string) + "=?"
		e.UpdateExec = append(e.UpdateExec, data[1])
	}

	//拼接sql
	e.Prepare = "update " + e.GetTable() + " set " + e.UpdateParam

	//如果where不为空
	if e.WhereParam != "" || e.OrWhereParam != "" {
		e.Prepare += " where " + e.WhereParam + e.OrWhereParam
	}

	//limit不为空
	if e.LimitParam != "" {
		e.Prepare += "limit " + e.LimitParam
	}

	//prepare
	var stmt *sql.Stmt
	var err error
	stmt, err = e.Db.Prepare(e.Prepare)
	if err != nil {
		return 0, e.setErrorInfo(err)
	}

	//合并UpdateExec和WhereExec
	if e.WhereExec != nil {
		e.AllExec = append(e.UpdateExec, e.WhereExec...)
	}

	//执行exec,注意这是stmt.Exec
	result, err := stmt.Exec(e.AllExec...)
	if err != nil {
		return 0, e.setErrorInfo(err)
	}

	//影响的行数
	id, _ := result.RowsAffected()
	return id, nil
}

// 查询多条，返回值为map切片
func (e *Orm) Select() ([]map[string]string, error) {

	//拼接sql
	e.Prepare = "select * from " + e.GetTable()

	//如果where不为空
	if e.WhereParam != "" || e.OrWhereParam != "" {
		e.Prepare += " where " + e.WhereParam + e.OrWhereParam
	}

	e.AllExec = e.WhereExec

	//query
	rows, err := e.Db.Query(e.Prepare, e.AllExec...)
	if err != nil {
		return nil, e.setErrorInfo(err)
	}

	//读出查询出的列字段名
	column, err := rows.Columns()
	if err != nil {
		return nil, e.setErrorInfo(err)
	}

	//values是每个列的值，这里获取到byte里
	values := make([][]byte, len(column))

	//因为每次查询出来的列是不定长的，用len(column)定住当次查询的长度
	scans := make([]interface{}, len(column))

	for i := range values {
		scans[i] = &values[i]
	}

	results := make([]map[string]string, 0)
	for rows.Next() {
		if err := rows.Scan(scans...); err != nil {
			//query.Scan查询出来的不定长值放到scans[i] = &values[i],也就是每行都放在values里
			return nil, e.setErrorInfo(err)
		}

		//每行数据
		row := make(map[string]string)

		//循环values数据，通过相同的下标，取column里面对应的列名，生成1个新的map
		for k, v := range values {
			key := column[k]
			row[key] = string(v)
		}

		//添加到map切片中
		results = append(results, row)
	}

	return results, nil
}

// 查询1条
func (e *Orm) SelectOne() (map[string]string, error) {

	//limit 1 单个查询
	results, err := e.Limit(1).Select()
	if err != nil {
		return nil, e.setErrorInfo(err)
	}

	//判断是否为空
	if len(results) == 0 {
		return nil, nil
	} else {
		return results[0], nil
	}
}

// 查询多条，返回值为struct切片
func (e *Orm) Find(result interface{}) error {

	if reflect.ValueOf(result).Kind() != reflect.Ptr {
		return e.setErrorInfo(errors.New("参数请传指针变量！"))
	}

	if reflect.ValueOf(result).IsNil() {
		return e.setErrorInfo(errors.New("参数不能是空指针！"))
	}

	//拼接sql
	e.Prepare = "select * from " + e.GetTable()

	e.AllExec = e.WhereExec

	//query
	rows, err := e.Db.Query(e.Prepare, e.AllExec...)
	if err != nil {
		return e.setErrorInfo(err)
	}

	//读出查询出的列字段名
	column, err := rows.Columns()
	if err != nil {
		return e.setErrorInfo(err)
	}

	//values是每个列的值，这里获取到byte里
	values := make([][]byte, len(column))

	//因为每次查询出来的列是不定长的，用len(column)定住当次查询的长度
	scans := make([]interface{}, len(column))

	//原始struct的切片值
	destSlice := reflect.ValueOf(result).Elem()

	//原始单个struct的类型
	destType := destSlice.Type().Elem()

	for i := range values {
		scans[i] = &values[i]
	}

	//循环遍历
	for rows.Next() {

		dest := reflect.New(destType).Elem()

		if err := rows.Scan(scans...); err != nil {
			//query.Scan查询出来的不定长值放到scans[i] = &values[i],也就是每行都放在values里
			return e.setErrorInfo(err)
		}

		//遍历一行数据的各个字段
		for k, v := range values {
			//每行数据是放在values里面，现在把它挪到row里
			key := column[k]
			value := string(v)

			//遍历结构体
			for i := 0; i < destType.NumField(); i++ {

				//看下是否有sql别名
				sqlTag := destType.Field(i).Tag.Get("sql")
				var fieldName string
				if sqlTag != "" {
					fieldName = strings.Split(sqlTag, ",")[0]
				} else {
					fieldName = destType.Field(i).Name
				}

				//struct里没这个key
				if key != fieldName {
					continue
				}

				//反射赋值
				if err := e.reflectSet(dest, i, value); err != nil {
					return err
				}
			}
		}
		//赋值
		destSlice.Set(reflect.Append(destSlice, dest))
	}

	return nil
}

// 反射赋值
func (e *Orm) reflectSet(dest reflect.Value, i int, value string) error {
	switch dest.Field(i).Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		res, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return e.setErrorInfo(err)
		}
		dest.Field(i).SetInt(res)
	case reflect.String:
		dest.Field(i).SetString(value)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		res, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return e.setErrorInfo(err)
		}
		dest.Field(i).SetUint(res)
	case reflect.Float32:
		res, err := strconv.ParseFloat(value, 32)
		if err != nil {
			return e.setErrorInfo(err)
		}
		dest.Field(i).SetFloat(res)
	case reflect.Float64:
		res, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return e.setErrorInfo(err)
		}
		dest.Field(i).SetFloat(res)
	case reflect.Bool:
		res, err := strconv.ParseBool(value)
		if err != nil {
			return e.setErrorInfo(err)
		}
		dest.Field(i).SetBool(res)
	}
	return nil
}

// 查询单条，返回值为struct切片
func (e *Orm) FindOne(result interface{}) error {

	//取的原始值
	dest := reflect.Indirect(reflect.ValueOf(result))

	//new一个类型的切片
	destSlice := reflect.New(reflect.SliceOf(dest.Type())).Elem()

	//调用
	if err := e.Limit(1).Find(destSlice.Addr().Interface()); err != nil {
		return err
	}

	//判断返回值长度
	if destSlice.Len() == 0 {
		return e.setErrorInfo(errors.New("NOT FOUND"))
	}

	//取切片里的第0个数据，并复制给原始值结构体指针
	dest.Set(destSlice.Index(0))
	return nil
}
func (e *Orm) Field(field string) *Orm {
	e.FieldParam = field
	return e
}

// limit分页
func (e *Orm) Limit(limit ...int64) *Orm {
	if len(limit) == 1 {
		e.LimitParam = strconv.Itoa(int(limit[0]))
	} else if len(limit) == 2 {
		e.LimitParam = strconv.Itoa(int(limit[0])) + "," + strconv.Itoa(int(limit[1]))
	} else {
		panic("参数个数错误")
	}
	return e
}

// 聚合查询
func (e *Orm) aggregateQuery(name, param string) (interface{}, error) {

	//拼接sql
	e.Prepare = "select " + name + "(" + param + ") as cnt from " + e.GetTable()

	//如果where不为空
	if e.WhereParam != "" || e.OrWhereParam != "" {
		e.Prepare += " where " + e.WhereParam + e.OrWhereParam
	}

	//limit不为空
	if e.LimitParam != "" {
		e.Prepare += " limit " + e.LimitParam
	}

	e.AllExec = e.WhereExec

	//生成sql
	e.generateSql()

	//执行绑定
	var cnt interface{}

	//queryRows
	err := e.Db.QueryRow(e.Prepare, e.AllExec...).Scan(&cnt)
	if err != nil {
		return nil, e.setErrorInfo(err)
	}

	return cnt, err
}

// 总数
func (e *Orm) Count() (int64, error) {
	count, err := e.aggregateQuery("count", "*")
	if err != nil {
		return 0, e.setErrorInfo(err)
	}
	return count.(int64), err
}
func (e *Orm) Max(param string) (string, error) {
	max, err := e.aggregateQuery("max", param)
	if err != nil {
		return "0", e.setErrorInfo(err)
	}
	return string(max.([]byte)), nil
}

// 最小值
func (e *Orm) Min(param string) (string, error) {
	min, err := e.aggregateQuery("min", param)
	if err != nil {
		return "0", e.setErrorInfo(err)
	}

	return string(min.([]byte)), nil
}

// 平均值
func (e *Orm) Avg(param string) (string, error) {
	avg, err := e.aggregateQuery("avg", param)
	if err != nil {
		return "0", e.setErrorInfo(err)
	}

	return string(avg.([]byte)), nil
}

// 总和
func (e *Orm) Sum(param string) (string, error) {
	sum, err := e.aggregateQuery("sum", param)
	if err != nil {
		return "0", e.setErrorInfo(err)
	}
	return string(sum.([]byte)), nil
}

// order排序
func (e *Orm) Order(order ...string) *Orm {
	orderLen := len(order)
	if orderLen%2 != 0 {
		panic("order by参数错误，请保证个数为偶数个")
	}

	//排序的个数
	orderNum := orderLen / 2

	//多次调用的情况
	if e.OrderParam != "" {
		e.OrderParam += ","
	}

	for i := 0; i < orderNum; i++ {
		keyString := strings.ToLower(order[i*2+1])
		if keyString != "desc" && keyString != "asc" {
			panic("排序关键字为：desc和asc")
		}
		if i < orderNum-1 {
			e.OrderParam += order[i*2] + " " + order[i*2+1] + ","
		} else {
			e.OrderParam += order[i*2] + " " + order[i*2+1]
		}
	}

	return e
}

// group分组
func (e *Orm) Group(group ...string) *Orm {
	if len(group) != 0 {
		e.GroupParam = strings.Join(group, ",")
	}
	return e
}

// having过滤
func (e *Orm) Having(having ...interface{}) *Orm {

	//判断是结构体还是多个字符串
	var dataType int
	if len(having) == 1 {
		dataType = 1
	} else if len(having) == 2 {
		dataType = 2
	} else if len(having) == 3 {
		dataType = 3
	} else {
		panic("having个数错误")
	}

	//多次调用判断
	if e.HavingParam != "" {
		e.HavingParam += "and ("
	} else {
		e.HavingParam += "("
	}

	//如果是结构体
	if dataType == 1 {
		t := reflect.TypeOf(having[0])
		v := reflect.ValueOf(having[0])

		var fieldNameArray []string
		for i := 0; i < t.NumField(); i++ {

			//小写开头，无法反射，跳过
			if !v.Field(i).CanInterface() {
				continue
			}

			//解析tag，找出真实的sql字段名
			sqlTag := t.Field(i).Tag.Get("sql")
			if sqlTag != "" {
				fieldNameArray = append(fieldNameArray, strings.Split(sqlTag, ",")[0]+"=?")
			} else {
				fieldNameArray = append(fieldNameArray, t.Field(i).Name+"=?")
			}

			e.WhereExec = append(e.WhereExec, v.Field(i).Interface())
		}
		e.HavingParam += strings.Join(fieldNameArray, " and ") + ") "

	} else if dataType == 2 {
		//直接=的情况
		e.HavingParam += having[0].(string) + "=?) "
		e.WhereExec = append(e.WhereExec, having[1])
	} else if dataType == 3 {
		//3个参数的情况
		e.HavingParam += having[0].(string) + " " + having[1].(string) + " ?) "
		e.WhereExec = append(e.WhereExec, having[2])
	}

	return e
}

// 生成完成的sql语句
func (e *Orm) generateSql() {
	e.Sql = e.Prepare
	for _, i2 := range e.AllExec {
		switch i2.(type) {
		case int:
			e.Sql = strings.Replace(e.Sql, "?", strconv.Itoa(i2.(int)), 1)
		case int64:
			e.Sql = strings.Replace(e.Sql, "?", strconv.FormatInt(i2.(int64), 10), 1)
		case bool:
			e.Sql = strings.Replace(e.Sql, "?", strconv.FormatBool(i2.(bool)), 1)
		default:
			e.Sql = strings.Replace(e.Sql, "?", "'"+i2.(string)+"'", 1)
		}
	}
}

// 获取最后执行生成的sql
func (e *Orm) GetLastSql() string {
	return e.Sql
}

// 直接执行增删改sql
func (e *Orm) Exec(sql string) (id int64, err error) {
	result, err := e.Db.Exec(sql)
	e.Sql = sql
	if err != nil {
		return 0, e.setErrorInfo(err)
	}

	//区分是insert还是其他(update,delete)
	if strings.Contains(sql, "insert") {
		lastInsertId, _ := result.LastInsertId()
		return lastInsertId, nil
	} else {
		rowsAffected, _ := result.RowsAffected()
		return rowsAffected, nil
	}
}

// 直接执行查sql
func (e *Orm) Query(sql string) ([]map[string]string, error) {
	rows, err := e.Db.Query(sql)
	e.Sql = sql
	if err != nil {
		return nil, e.setErrorInfo(err)
	}

	//读出查询出的列字段名
	column, err := rows.Columns()
	if err != nil {
		return nil, e.setErrorInfo(err)
	}

	//values是每个列的值，这里获取到byte里
	values := make([][]byte, len(column))

	//因为每次查询出来的列是不定长的，用len(column)定住当次查询的长度
	scans := make([]interface{}, len(column))

	for i := range values {
		scans[i] = &values[i]
	}

	//最后得到的map
	results := make([]map[string]string, 0)
	for rows.Next() {
		if err := rows.Scan(scans...); err != nil {
			//query.Scan查询出来的不定长值放到scans[i] = &values[i],也就是每行都放在values里
			return nil, e.setErrorInfo(err)
		}

		row := make(map[string]string) //每行数据
		for k, v := range values {
			//每行数据是放在values里面，现在把它挪到row里
			key := column[k]
			row[key] = string(v)
		}
		results = append(results, row)
	}

	return results, nil
}

// 开启事务
func (e *Orm) Begin() error {

	//调用原生的开启事务方法
	tx, err := e.Db.Begin()
	if err != nil {
		return e.setErrorInfo(err)
	}
	e.TransStatus = 1
	e.Tx = tx
	return nil
}

// 事务回滚
func (e *Orm) Rollback() error {
	e.TransStatus = 0
	return e.Tx.Rollback()
}

// 事务提交
func (e *Orm) Commit() error {
	e.TransStatus = 0
	return e.Tx.Commit()
}
