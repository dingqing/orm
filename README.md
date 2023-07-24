# ORM
> 手动实现ORM（Golang）

目录 |二级目录
---|---
[理解](#理解) |[原生方式](#原生方式)，[API列表](#方法列表)
[实现](#实现) |
[性能测试](#性能测试)

***

## 理解
- 为什么要用ORM？

    原生操作数据库的方式代码复杂、重复，

    ORM作为应用代码与数据库操作的中间层， **保证开发效率、数据库操作性能与安全**，

    即，ORM通过函数封装，底层调用database/sql库、为程序提供简便的数据库操作API。

### 原生方式
"database/sql"库中的操作方式是“SQL语句+占位符、传入实际变量执行”，如：
```go []
//方式一，Exec()：
result, err := db.Exec("INSERT INTO user (username, departname, created) VALUES (?, ?, ?)","lisi","dev","2020-08-04")

//方式二，Prepare()+Exec()，效率更高：
stmt, err := db.Prepare("INSERT INTO user (username, departname, created) VALUES (?, ?, ?)")
result2, err := stmt.Exec("zhangsan", "pro", time.Now().Format("2006-01-02"))
```

### 方法列表
要求支持链式调用。如：
```go []
orm.Where().OrWhere().Order().Limit().Select()
```
方法|说明
---|---
NewMysql(string,string,string,string) |参数：用户名、密码、数据库地址、数据库名
设置查询字段Field(string)
Table(string)
Join() |待实现
Where(),OrWhere() |分别相当于sql中的and和or，均支持两种调用方式（参数可以是字符串或结构体）
Group(...string)
Having(...any) |支持两种调用方式（参数可以是字符串或结构体）
Order(...string) |要求参数个数为偶数，如Order("uid","asc", "status", "desc")
Limit(...int64) |支持一个或两个参数
查询多条Select()，查询单条SelectOne() |返回类型分别为map切片、map
查询多条Find(any)，查询单条FindOne(any) |返回类型分别为引用结构体切片、引用结构体
Count()/Max()/Min()/Avg()/Sum()
Insert(any)/Replace(any) |支持批量或单个插入（参数可以是结构体或结构体切片），后面不允许链式调用其他方法
Delete() |后面不允许链式调用其他方法
Update() |支持两种调用方式（参数可以是字符串或结构体），后面不允许链式调用其他方法
GetLastSql()
Exec(string)/Query(string) |执行原生sql的增删改/查询操作
事务Begin()/Commit()/Rollback() |[使用示例](#事务使用示例)

#### Find(any)，FindOne(any)使用示例
```go []
//定义好结构体
type User struct {
    Uid        int    `sql:"uid,auto_increment"`
    Username   string `sql:"username"`
    Departname string `sql:"departname"`
    Status     int64  `sql:"status"`
}

var users []User
// select * from userinfo where status=1
err := e.Table("userinfo").Where("status", 2).Find(&users)
if err != nil {
    fmt.Println(err.Error())
} else {
    fmt.Printf("%#v", users)
}

var user User
// select * from userinfo where status=1
err := e.Table("userinfo").Where("status", 2).FindOne(&user)
if err != nil {
    fmt.Println(err.Error())
} else {
    fmt.Printf("%#v", user)
}
```

#### 事务使用示例
```go []
err0 := e.Begin()
isCommit := true
if err0 != nil {
    fmt.Println(err0.Error())
    os.Exit(1)
}

result1, err1 := e.Table("user").Where("uid", "=", 10803).Update("departname", 110)
if err1 != nil {
    isCommit = false
    fmt.Println(err1.Error())
}
if result1 <= 0 {
    isCommit = false
    fmt.Println("update 0")
}
fmt.Println("result1 is :", result1)
fmt.Println("sql is :", e.GetLastSql())

result2, err2 := e.Table("user").Where("uid", "=", 10802).Delete()
if err2 != nil {
    isCommit = false
    fmt.Println(err2.Error())
}
if result2 <= 0 {
    isCommit = false
    fmt.Println("delete 0")
}
fmt.Println("result2 is :", result2)
fmt.Println("sql is :", e.GetLastSql())

user1 := User{
    Username:   "EE",
    Departname: "22",
    Created:    "2012-12-12",
    Status:     1,
}
id, err3 := e.Table("user").Insert(user1)
if err3 != nil {
    isCommit = false
    fmt.Println(err3.Error())
}
fmt.Println("id is :", id)
fmt.Println("sql is :", e.GetLastSql())

if isCommit {
    _ = e.Commit()
    fmt.Println("ok")
} else {
    _ = e.Rollback()
    fmt.Println("error")
}
```

## 实现
通过反射获取入参类型，拼接SQL语句调用"database/sql"库中的方法。

***

## 性能测试
```
go test -bench=. -benchmem 
```