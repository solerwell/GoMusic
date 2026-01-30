package config

import (
	"fmt"
	"sync"

	"github.com/spf13/viper"

	"GoMusic/misc/log"
)

// Config 全局配置结构
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port int `mapstructure:"port"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Type   string       `mapstructure:"type"`
	MySQL  MySQLConfig  `mapstructure:"mysql"`
	SQLite SQLiteConfig `mapstructure:"sqlite"`
}

// MySQLConfig MySQL数据库配置
type MySQLConfig struct {
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	Username  string `mapstructure:"username"`
	Password  string `mapstructure:"password"`
	DBName    string `mapstructure:"dbname"`
	Charset   string `mapstructure:"charset"`
	ParseTime bool   `mapstructure:"parseTime"`
	Loc       string `mapstructure:"loc"`
}

// SQLiteConfig SQLite数据库配置
type SQLiteConfig struct {
	Path string `mapstructure:"path"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

var (
	instance *Config
	once     sync.Once
)

// GetConfig 获取全局配置实例(单例模式)
func GetConfig() *Config {
	once.Do(func() {
		instance = loadConfig()
	})
	return instance
}

// loadConfig 加载配置文件
func loadConfig() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")      // 当前目录
	viper.AddConfigPath("./")     // 当前目录
	viper.AddConfigPath("../")    // 上级目录
	viper.AddConfigPath("../../") // 上上级目录

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		log.Errorf("读取配置文件失败: %v", err)
		panic(fmt.Sprintf("读取配置文件失败: %v", err))
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		log.Errorf("解析配置文件失败: %v", err)
		panic(fmt.Sprintf("解析配置文件失败: %v", err))
	}

	log.Infof("配置加载成功,数据库类型: %s", config.Database.Type)
	return &config
}

// IsMySQL 判断是否使用MySQL数据库
func (c *Config) IsMySQL() bool {
	return c.Database.Type == "mysql"
}

// IsSQLite 判断是否使用SQLite数据库
func (c *Config) IsSQLite() bool {
	return c.Database.Type == "sqlite"
}

// GetMySQLDSN 获取MySQL连接字符串
func (c *Config) GetMySQLDSN() string {
	mysql := c.Database.MySQL
	parseTime := "True"
	if !mysql.ParseTime {
		parseTime = "False"
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%s&loc=%s",
		mysql.Username,
		mysql.Password,
		mysql.Host,
		mysql.Port,
		mysql.DBName,
		mysql.Charset,
		parseTime,
		mysql.Loc,
	)
}

// GetRedisAddr 获取Redis地址
func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}

// GetServerAddr 获取服务器监听地址
func (c *Config) GetServerAddr() string {
	return fmt.Sprintf(":%d", c.Server.Port)
}
