package db

import (
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"GoMusic/misc/config"
	"GoMusic/misc/log"
	"GoMusic/misc/models"
)

var db *gorm.DB

func init() {
	cfg := config.GetConfig()
	var err error

	// 根据配置类型初始化数据库
	if cfg.IsMySQL() {
		// MySQL数据库
		dsn := cfg.GetMySQLDSN()
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Errorf("MySQL数据库连接失败：%v", err)
			panic(err)
		}
		log.Infof("MySQL数据库连接成功")
	} else if cfg.IsSQLite() {
		// SQLite数据库
		dbPath := cfg.Database.SQLite.Path

		// 确保数据库文件所在目录存在
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Errorf("创建SQLite数据库目录失败：%v", err)
			panic(err)
		}

		db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			log.Errorf("SQLite数据库连接失败：%v", err)
			panic(err)
		}
		log.Infof("SQLite数据库连接成功: %s", dbPath)
	} else {
		err := fmt.Errorf("不支持的数据库类型: %s", cfg.Database.Type)
		log.Errorf("%v", err)
		panic(err)
	}

	// 自动创建表
	db.AutoMigrate(&models.NetEasySong{})

	// 仅对MySQL执行字段迁移
	if cfg.IsMySQL() {
		if err := MigrateNameField(db); err != nil {
			log.Errorf("failed to migrate database: %v", err)
		}
	}
}

func MigrateNameField(db *gorm.DB) error {
	// 使用原生 SQL 来修改字段长度
	return db.Exec("ALTER TABLE net_easy_songs MODIFY name VARCHAR(512);").Error
}

func BatchGetSongById(ids []uint) (map[uint]string, error) {
	var netEasySongs []*models.NetEasySong
	// 仅选择 id 和 name 列
	err := db.Select("id, name").Where("id in ?", ids).Find(&netEasySongs).Error
	if err != nil {
		log.Errorf("查询数据库失败：%v", err)
		return nil, err
	}

	// 歌曲id:歌曲信息
	netEasySongMap := make(map[uint]string, len(netEasySongs))
	for _, v := range netEasySongs {
		netEasySongMap[v.Id] = v.Name
	}
	return netEasySongMap, nil
}

func BatchInsertSong(netEasySongs []*models.NetEasySong) error {
	// 如果 Duplicate primary key 则执行 update 操作
	err := db.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).CreateInBatches(netEasySongs, 500).Error
	if err != nil {
		log.Errorf("数据库插入失败：%v", err)
	}
	return err
}

func BatchDelSong(ids []int) error {
	err := db.Delete(&models.NetEasySong{}, ids).Error
	if err != nil {
		log.Errorf("数据库删除数据失败：%v", err)
	}
	return err
}
