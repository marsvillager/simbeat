package beater

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"golang.org/x/text/encoding/simplifiedchinese"

	"github.com/xejwofuxu/simbeat/config"
)

type Charset string

const (
	UTF8    = Charset("UTF-8")
	GB18030 = Charset("GB18030")
)

// 本地配置表
type Checkpoint struct {
	RuleId string   `json:"ruleId"`
	Param  []string `json:"param"`
}

type CheckSlice struct {
	CSlice map[string]Checkpoint `json:"checkpoint"`
}

type Check struct {
	CList map[string]CheckSlice `json:"check"`
}

// 基线配置文件
type Platform struct {
	Win   []string `json:"windows"`
	Linux []string `json:"linux"`
}

// Simbeat configuration.
type Simbeat struct {
	done   chan struct{}
	config config.Config
	client beat.Client
}

// New creates an instance of simbeat.
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	c := config.DefaultConfig
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	bt := &Simbeat{
		done:   make(chan struct{}),
		config: c,
	}
	return bt, nil
}

// Run starts simbeat.
func (bt *Simbeat) Run(b *beat.Beat) error {
	logp.Info("simbeat is running! Hit CTRL-C to stop it.")

	var err error
	bt.client, err = b.Publisher.Connect()
	if err != nil {
		return err
	}

	ticker := time.NewTicker(bt.config.Period)
	for {
		select {
		case <-bt.done:
			return nil
		case <-ticker.C:
		}

		// 打开数据库
		db, err := badger.Open(badger.DefaultOptions("database"))
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		// 将组策略导入到数据库中
		gp(db)

		// 将服务导入到数据库中
		services(db)

		// // 打开和读取本地配置文件
		configFile, err := os.Open(`allConf.conf`)
		if err != nil {
			log.Fatal(err)
		}
		byteValue, _ := ioutil.ReadAll(configFile)
		var checkpoint Check
		json.Unmarshal(byteValue, &checkpoint)

		// 打开和读取基线配置文件
		file, err := os.Open(`testget.conf`)
		if err != nil {
			log.Fatal(err)
		}
		check, _ := ioutil.ReadAll(file)
		var platform Platform
		json.Unmarshal(check, &platform)

		for i := 0; i < len(platform.Win); i++ {
			RuleId := platform.Win[i]
			// fmt.Print("RuleId: " + RuleId + "\n")

			// 判断本地是否有对应的配置文件
			test, err := checkpoint.CList["windows"].CSlice[platform.Win[i]]
			if !err {
				log.Println("没有对应配置项")
				// 传空值
				continue
			}

			// param 的多行处理
			var param []string
			var value []string
			for j := 0; j < len(test.Param); j++ {
				param = append(param, test.Param[j])

				// 只读事务
				gtxn := db.NewTransaction(false)
				defer gtxn.Discard()
				// 检索键值对
				if item, err := gtxn.Get([]byte(test.Param[j])); err == nil {
					if v, err := item.ValueCopy(nil); err == nil {
						value = append(value, string(v))
					}
				}
			}

			event := beat.Event{
				Timestamp: time.Now(),
				Fields: common.MapStr{
					"ostype": b.Info.Name,
					"ruleId": RuleId,
					"param":  param,
					"value":  value,
				},
			}

			bt.client.Publish(event)
			logp.Info("Event sent")

			// fmt.Print(param)
			// fmt.Print("\n")
			// fmt.Print(value)
			// fmt.Print("\n")

			// 遍历整个数据库（只读事务）
			// txn := db.NewTransaction(false)
			// defer txn.Discard()
			// iter := badger.DefaultIteratorOptions
			// it := txn.NewIterator(iter)
			// defer it.Close()
			// for it.Rewind(); it.Valid(); it.Next() {
			// 	item := it.Item()
			// 	fmt.Printf("key: %s\n", item.Key())
			// 	fmt.Print([]byte(item.Key()))
			// 	fmt.Print("\n")
			// 	value, _ := item.ValueCopy(item.Key())
			// 	fmt.Printf("value: %s\n", value)
			// }
		}

		// 关闭数据库和文件
		db.Close()
		configFile.Close()

		// 删除指定目录下特定后缀的文件
		WalkDir("database", "sst")
		WalkDir("database", "vlog")
		os.Remove(`database\MANIFEST`)
	}

}

// Stop stops simbeat.
func (bt *Simbeat) Stop() {
	bt.client.Close()
	close(bt.done)
}

func gp(db *badger.DB) {
	// 防止 gp.reg 未删除
	os.Remove(`gp.reg`)
	// 保存组策略注册表到文件 gp.reg 中
	gpcmd := exec.Command("reg", "export", `HKEY_CURRENT_USER\SOFTWARE\Microsoft\Windows\CurrentVersion\Group Policy Objects`, "gp.reg")
	gpcmd.Run()

	// 打开文件 gp.reg
	file, err := os.Open(`gp.reg`)
	if err != nil {
		log.Fatal(err)
	}

	// 读取文件
	r := bufio.NewReader(file)

	// 插入数据
	var flag = 1
	var param string
	for {
		// 分行读取
		buf, err := r.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}
		// 由于保存的文件 ascii码带有 0（即空格），需要单独处理
		var gp []byte
		for i := 0; (2*i + 1) < len(buf); i++ {
			gp = append(gp, buf[2*i+1])
		}
		tmp := ConvertByte2String(gp, "UTF8")
		tmp = strings.Replace(tmp, "\n", "", -1)
		tmp = strings.Replace(tmp, "\r", "", -1)
		tmp = strings.Replace(tmp, "]", "", -1)

		if len(tmp) == 0 {
			flag = 1
			continue
		}

		if flag == 1 {
			param = tmp
			// 从随机 uuid 后存入
			param = param[strings.Index(param, "}")+1:]
			flag = 0
		} else if tmp[1:3] != "**" { // 不存 ** 开头的注册表数据
			// 转 int 去掉前置 0，在转 string 存，以便区分空字段和数值 0 (int 类型不行)
			key := tmp[1 : strings.Index(tmp, "=")-1]
			// 分为（1）有冒号的比如："EnableScripts"=dword:00000001，这种是取数值
			// 还有（2）无冒号的比如："ExecutionPolicy"="AllSigned"，这种是 TEXT
			var b uint64
			b, err = strconv.ParseUint(tmp[strings.Index(tmp, ":")+1:], 16, 32)
			if err != nil {
				value := tmp[strings.Index(tmp, "=")+1:]

				// 读写事务（第一个参数是 bool 值，表示事务是否应可写）
				update := db.NewTransaction(true)
				defer update.Discard()
				// 创建键值对
				if err := update.Set([]byte(param+":"+key), []byte(value)); err == nil {
					_ = update.Commit()
				}
			} else {
				value := strconv.Itoa(int(b))
				if err != nil {
					log.Fatal(err)
				}

				// 读写事务（第一个参数是 bool 值，表示事务是否应可写）
				update := db.NewTransaction(true)
				defer update.Discard()
				// 创建键值对
				if err := update.Set([]byte(param+":"+key), []byte(value)); err == nil {
					_ = update.Commit()
				}
			}
		}
	}

	// 关闭和删除文件 gp.reg
	file.Close()
	os.Remove("gp.reg")
}

func services(db *badger.DB) {
	// cmd 查看开启了那些服务
	servicecmd := exec.Command("net", "start")

	// 读取 cmd 输出
	stdout, err := servicecmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	servicecmd.Start()

	// 处理读取数据
	readservice := bufio.NewReader(stdout)
	var services []string
	for {
		buf, err := readservice.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}

		var tmp []byte
		// 去掉开头的空格（32），从第三个 byte 开始
		// 去掉 \n（10）、\r（13）
		for i, v := range buf {
			if i > 2 && v != 10 && v != 13 {
				tmp = append(tmp, buf[i])
			}
		}

		services = append(services, ConvertByte2String(tmp, "GB18030"))
	}

	// 更新 value，1 表示开启，0 表示关闭
	for i := 0; i < len(services); i++ { // 包含头尾的杂字段，但并不影响，另多不少
		// 读写事务（第一个参数是 bool 值，表示事务是否应可写）
		update := db.NewTransaction(true)
		defer update.Discard()
		// 创建键值对
		if err := update.Set([]byte(services[i]), []byte("1")); err == nil {
			_ = update.Commit()
		}
	}
}

func ConvertByte2String(byte []byte, charset Charset) string {
	var str string
	switch charset {
	case GB18030:
		var decodeBytes, _ = simplifiedchinese.GB18030.NewDecoder().Bytes(byte)
		str = string(decodeBytes)
	case UTF8:
		fallthrough
	default:
		str = string(byte)
	}
	return str
}

/* 获取指定路径下以及所有子目录下的所有文件，可匹配后缀过滤（suffix为空则不过滤）*/
func WalkDir(dir, suffix string) {
	rd, _ := ioutil.ReadDir(dir)
	for _, fi := range rd {
		if fi.IsDir() {
			WalkDir(dir+string(filepath.Separator)+fi.Name(), suffix)
		} else {
			if strings.HasSuffix(fi.Name(), suffix) {
				toDelFile := dir + string(filepath.Separator) + fi.Name()
				fmt.Printf("删除文件[%s]\n", toDelFile)
				os.Remove(toDelFile)
			}
		}
	}
}
