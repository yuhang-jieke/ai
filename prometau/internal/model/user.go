package model

// User 用户模型
type User struct {
	ID       uint   `gorm:"primaryKey;autoIncrement;comment:用户ID"`
	Account  string `gorm:"type:varchar(30);uniqueIndex;not null;comment:账号"`
	Password string `gorm:"type:varchar(255);not null;comment:密码"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}
