package db

import (
	serverconfig "Dapp/server_config"
	"fmt"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	GSqlInstance *gorm.DB
)

const (
	globalSqlDNS = "%s:%s@tcp(%s)/%s?charset" +
		"=utf8mb4&loc=Local&parseTime=true" +
		"&interpolateParams=false" // globalSqlDNS 数据源sql的dns
)

func initDataBase(cfg *serverconfig.MySqlConfig) (*gorm.DB, error) {
	dbSourceName := fmt.Sprintf(globalSqlDNS,
		cfg.UserName, cfg.Password, cfg.DbAddress, cfg.Database)
	gormDB, err := gorm.Open(mysql.Open(dbSourceName), &gorm.Config{})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "init database(%s) got error : %s  ",
			cfg.Database, err.Error())
		return nil, err
	}
	db, dbErr := gormDB.DB()
	if dbErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "get database(%s) got error : %s  ",
			cfg.Database, dbErr.Error())
		return nil, err
	}
	// 设置一下最大空连接
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	// 设置一下最大连接
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	// 设置一下最大空连接时长
	db.SetConnMaxIdleTime(time.Duration(cfg.MaxIdleSeconds) * time.Second)
	// 设置一下最大连接时长
	db.SetConnMaxLifetime(time.Duration(cfg.MaxLifeSeconds) * time.Second)
	// ping一下数据库
	pErr := db.Ping()
	if pErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "ping database(%s) got error : %s  ",
			cfg.Database, pErr.Error())
		return nil, pErr
	}
	GSqlInstance = gormDB
	return GSqlInstance, nil
}

func CloseDB() error {
	db, dbErr := GSqlInstance.DB()
	if dbErr != nil {
		return dbErr
	}
	return db.Close()
}

func InitDB() {
	systemCFG := serverconfig.GlobalCFG.DBCFG
	var retError error
	GSqlInstance, retError = initDataBase(systemCFG)
	if retError != nil {
		panic(fmt.Sprintf("server connect sql error(%s)", retError.Error()))
	}
	fmt.Fprintln(os.Stdout, "server connect mysql success")
}
