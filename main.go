package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	_cache "com.cne/ai-tracking-monitor/cache"
	_db "com.cne/ai-tracking-monitor/db"
	_queue "com.cne/ai-tracking-monitor/queue"
	_utils "com.cne/ai-tracking-monitor/utils"
)

const (
	AppName    string = "tracking-monitor" // 表示应用程序名。
	AppVersion string = "0.1.0"            // 表示应用程序版本。

	DefaultConfigFile string = "./" + AppName + ".json" // 表示默认的配置文件名。
	DefaultDebug      bool   = false                    // 表示默认是否开启Debug模式。

	DefaultRedisHost     string = "localhost" // 表示默认的Redis主机地址。
	DefaultRedisPort     int    = 6379        // 表示默认的Redis端口号。
	DefaultRedisPassword string = ""          // 表示默认的Redis口令。
	DefaultRedisDB       int    = 0           // 表示默认的Redis数据库。
)

var (
	flagVersion bool // 是否显示版权信息
	flagHelp    bool // 是否显示帮助信息
	flagVerify  bool // 是否只检查配置文件
	flagDebug   bool // 是否显示调试信息

	configuration *Configuration = &Configuration{
		Redis: RedisConfiguration{
			Host:     DefaultRedisHost,
			Port:     DefaultRedisPort,
			Password: DefaultRedisPassword,
			DB:       DefaultRedisDB,
		},
	}
)

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Llongfile)
	log.SetOutput(&_utils.RollingFileLoggerWriter{Pattern: "log/" + AppName + "-$date.log"})

	flag.BoolVar(&flagVersion, "version", false, "Shows version message")
	flag.BoolVar(&flagHelp, "h", false, "Shows this help message")
	flag.BoolVar(&flagVerify, "verify", false, "Verify configuration and quit")
	flag.BoolVar(&flagDebug, "debug", DefaultDebug, "Show debugging information")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -version\n", AppName)
		fmt.Fprintf(os.Stderr, "Usage: %s -h\n", AppName)
		fmt.Fprintf(os.Stderr, "Usage: %s -verify\n", AppName)
		fmt.Fprintf(os.Stderr, "Usage: %s [-debug] [CONFIG_FILE]\n", AppName)
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	if flagVersion {
		fmt.Printf("%s version %s\n", AppName, AppVersion)
		return
	}

	if flagHelp {
		flag.Usage()
		return
	}

	if !flagDebug {
		log.SetOutput(ioutil.Discard)
	}

	// 加载配置。
	if err := loadConfig(strings.TrimSpace(flag.Arg(0))); err != nil {
		panic(fmt.Errorf("cannot load configuration: %w", err))
	}

	if flagVerify {
		fmt.Printf("configuration:\n%#v\n", configuration)
		return
	}

	// 输出pid文件。
	pidFilename := AppName + "-pid"
	if runtime.GOOS != "windows" {
		pidFilename = "/var/run/" + pidFilename
	}
	if pidFile, err := os.Create(pidFilename); err == nil {
		pidFile.WriteString(fmt.Sprintf("%v", os.Getpid()))
		pidFile.Close()

		defer os.Remove(pidFilename)
	}

	// 初始化数据库。
	if err := _db.InitDB(configuration.DB.DSN); err != nil {
		panic(err)
	}

	// 初始化Redis缓存。
	if err := _cache.InitRedisCache(configuration.Redis.Host, configuration.Redis.Port, configuration.Redis.Password, configuration.Redis.DB); err != nil {
		panic(err)
	}

	// 初始化Redis队列。
	if err := _queue.InitRedisQueue(configuration.Redis.Host, configuration.Redis.Port, configuration.Redis.Password, configuration.Redis.DB); err != nil {
		panic(err)
	}

	// 开始服务。
	err := runForEver()
	if err != nil {
		panic(err)
	}

}

func loadConfig(configFile string) (err error) {
	if configFile == "" {
		configFile = DefaultConfigFile
	}

	configFile, err = filepath.Abs(configFile)
	if err != nil {
		return err
	}

	configFileStat, err := os.Stat(configFile)
	if err != nil {
		return err
	}

	if configFileStat.IsDir() {
		configFile = filepath.Join(configFile, DefaultConfigFile)
	}

	err = loadConfigFromFile(configFile)
	if err != nil {
		return err
	}

	// 检查数据库DSN的格式是否正确。
	configuration.DB.DSN = strings.TrimSpace(configuration.DB.DSN)
	if configuration.DB.DSN == "" || !strings.Contains(configuration.DB.DSN, "@") || !strings.Contains(configuration.DB.DSN, ":") {
		return fmt.Errorf("dsn should contains at(@) and colon(:)")
	}

	return err
}

func loadConfigFromFile(configFile string) (err error) {
	var cf *os.File

	fmt.Printf("Loading configuration from %s ...\n", configFile)

	if cf, err = os.Open(configFile); err != nil {
		return err
	}

	defer cf.Close()

	dec := json.NewDecoder(cf)
	err = dec.Decode(&configuration)
	if err != nil {
		return err
	}

	return nil
}

func runForEver() error {
	go doRun()

	// 启动守护routine。
	sigChannel := make(chan os.Signal, 256)
	signal.Notify(sigChannel, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	for {
		sig := <-sigChannel
		fmt.Fprintf(os.Stderr, "Received sig: %#v\n", sig)
		switch sig {
		case syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM:
			return nil
		}
	}
}

func doRun() {
	// TODO: 通过配置文件设置轮询周期。
	timer := time.NewTimer(5 * time.Second)

	fmt.Printf("Checking... \n")

	for {
		<-timer.C
		timer.Reset(5 * time.Minute)

		go func() {
			defer _utils.RecoverPanic()

			doCheck()
		}()
	}
}
