package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/hpgood/boomer"
	httpwrapper "github.com/hpgood/go-httpwrapper"
)
var config string

func init() {
    flag.StringVar(&config,"data","test.json","--data=test.json 测试配置文件")
}
//loadConfig loadConfig
func loadConfig(name string) (string,error) {
	_, err := os.Stat(name)
	if err != nil {
		log.Println(err.Error())
		return "",err
	}
 
	bys, err := ioutil.ReadFile(name)
	if err != nil {
		log.Println(err.Error())
		return "",err
	}
	return string(bys),nil
}

func main() {
  
    flag.Parse()
    
    log.Printf("配置文件 --data=%s\n",config)
	start := time.Now()
    templateJSONStr,err:=loadConfig(config)
    if err!=nil{
        // err.Error()
        log.Fatal(err.Error())
    }
	tasks := httpwrapper.GetTaskList(templateJSONStr)
	boomer.Run(tasks...)
	log.Println("用时:", time.Since(start))
}
