package orm

import (
	"testing"

	"gorm.io/gorm"
)

func BenchmarkOrmSelect(b *testing.B) {
	e, _ := NewMysql("root", "123456", "127.0.0.1:3306", "test")

	type User struct {
		Username   string `gorm:"username"`
		Departname string `gorm:"departname"`
		Created    string `gorm:"created"`
		Status     int64  `gorm:"status"`
	}
	var users []User

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = e.Table("userinfo").Where("uid", ">=", 50).Limit(100).Find(&users)
	}
	b.StopTimer()
}

func BenchmarkGormSelect(b *testing.B) {
	dsn := "root:123456@tcp(127.0.0.1:3306)/test?charset=utf8mb4&parseTime=True&loc=Local"
	db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	type User struct {
		Username   string `gorm:"username"`
		Departname string `gorm:"departname"`
		Created    string `gorm:"created"`
		Status     int64  `gorm:"status"`
	}
	var users []User

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.Table("userinfo").Where("uid >= ?", "50").Limit(50).Find(&users)
	}
	b.StopTimer()
}

func BenchmarkOrmUpdate(b *testing.B) {
	e, _ := NewMysql("root", "123456", "127.0.0.1:3306", "test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.Table("userinfo").Where("uid", "=", 15).Update("status", 0)
	}
	b.StopTimer()
}

func BenchmarkGormUpdate(b *testing.B) {
	dsn := "root:123456@tcp(127.0.0.1:3306)/test?charset=utf8mb4&parseTime=True&loc=Local"
	db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.Table("userinfo").Where("uid = ?", "15").Update("status", 1)
	}
	b.StopTimer()
}
