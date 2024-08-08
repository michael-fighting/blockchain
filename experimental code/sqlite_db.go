package cleisthenes

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

var (
	Db        *sql.DB
	BlockStmt *sql.Stmt
	IndexStmt *sql.Stmt
)

func init() {
	db, err := sql.Open("sqlite3", "./block.db")
	if err != nil {
		log.Fatal().Msgf("open block db failed, err:%s", err.Error())
	}
	Db = db

	_, err = db.Exec("select 1 from block")
	if err != nil {
		// block表不存在
		_, err = db.Exec(`
		CREATE TABLE block (
		    height INTEGER PRIMARY KEY,
		    data VARCHAR(64) NULL
			)
		`)
		if err != nil {
			log.Fatal().Msgf("create block table failed, err:%s", err.Error())
		}
	}

	_, err = db.Exec("select 1 from tx_index")
	if err != nil {
		// tx_index表不存在
		_, err = db.Exec(`
		CREATE TABLE tx_index (
		    txId VARCHAR(64) PRIMARY KEY,
		    height INTEGER
			)
		`)
		if err != nil {
			log.Fatal().Msgf("create block table failed, err:%s", err.Error())
		}
	}
	stmt, err := Db.Prepare("INSERT INTO block(height, data) values(?,?)")
	if err != nil {
		log.Fatal().Msgf("prepare block stmt failed, err:%s", err.Error())
	}
	BlockStmt = stmt

	stmt, err = Db.Prepare("INSERT INTO tx_index(txId, height) values(?, ?)")
	if err != nil {
		log.Fatal().Msgf("prepare tx index stmt failed, err:%s", err.Error())
	}
	IndexStmt = stmt
}

func WriteBlock(height uint64, data []byte) error {
	return nil
	_, err := BlockStmt.Exec(height, data)
	if err != nil {
		log.Error().Msgf("write block failed, height:%v, err:%s", height, err.Error())
	}
	return err
}

func WriteIndex(height uint64, txs []interface{}) error {
	transaction, err := Db.Begin()
	if err != nil {
		log.Fatal().Msgf("begin write index db transaction failed, err:%s", err.Error())
	}
	defer transaction.Rollback()

	stmt, err := transaction.Prepare("INSERT INTO tx_index(txId, height) values(?, ?)")
	if err != nil {
		log.Fatal().Msgf("transaction prepare stmt failed, err:%s", err.Error())
	}
	defer stmt.Close()

	for _, tx := range txs {
		t := tx.(map[string]interface{})
		id := t["id"].(string)
		_, err := stmt.Exec(id, height)
		if err != nil {
			log.Error().Msgf("write index failed, id:%s height:%v, err:%s", id, height, err.Error())
			return err
		}
	}

	err = transaction.Commit()
	if err != nil {
		log.Fatal().Msgf("transaction commit failed, err:%s", err.Error())
	}

	return nil
}
