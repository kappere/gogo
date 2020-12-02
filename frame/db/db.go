package db

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql" // mysql dialects
	"wataru.com/gogo/config"
	"wataru.com/gogo/logger"
	"wataru.com/gogo/util"
)

// Db 实例
var Db *gorm.DB

func InitDb() (*gorm.DB, func()) {
	dbConf := util.ValueOrDefault((*config.GlobalConfig.Map)["database"], make(map[interface{}]interface{})).(map[interface{}]interface{})
	dbtype := util.ValueOrDefault(dbConf["dbtype"], "").(string)
	url := util.ValueOrDefault(dbConf["url"], "").(string)
	_db, err := gorm.Open(dbtype, url)
	Db = _db
	if err != nil {
		panic(err)
	}
	Db.SingularTable(true)
	// 启用Logger，显示详细日志
	Db.LogMode(true)
	Db.SetLogger(logger.GetGormLogger())
	logger.Info("Initialize datasource")
	return Db, func() {
		if err := Db.Close(); err != nil {
			logger.Error("Datasource close failed: %v", err)
		} else {
			logger.Info("Datasource closed")
		}
	}
}
