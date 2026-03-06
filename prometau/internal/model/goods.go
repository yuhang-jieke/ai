package model

import "gorm.io/gorm"

type Goods struct {
	gorm.Model
	Name  string  `gorm:"type:varchar(30);comment:名称"`
	Price float64 `gorm:"type:decimal(10,2);comment:价格"`
}
