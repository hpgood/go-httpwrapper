package httpwrapper

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"text/template"
	"time"

	"github.com/hpgood/boomer"
	jsoniter "github.com/json-iterator/go"
)

const (
	NoValue = "<no value>"
)

type Variable map[string]interface{}

type Variables struct {
	Declare          []string               `json:"declare"`
	InitVariables    map[string]interface{} `json:"init_variables"`
	RunningVariables map[string]interface{} `json:"running_variables"`
	MergedVariables  map[string]interface{}
}

type RunScript struct {
	Debug  bool              `json:"debug"`
	Domain string            `json:"domain"`
	Header map[string]string `json:"header"`
	Variables
	FuncSet        []FuncSet `json:"func_set"`
	WithInitVar    bool
	WithRunningVar bool
}

type StoreKV struct{
	Name  string  `json:"name"`
	Value string 	`json:"value"`
}

type FuncSet struct {
	Key         string            `json:"key"`
	Method      string            `json:"method"`
	Body        string            `json:"body"`
	Url         string            `json:"url"`
	Header      map[string]string `json:"header"`
	Probability int               `json:"probability"`
	Validator   string            `json:"validator"`
	Condition   string            `json:"condition"` //运行条件
	Store       map[string]string `json:"store"`     //保存的内容
	Parsed      struct {
		Body   StrComponent
		Url    StrComponent
		Header SMapComponent
	}
	RScript *RunScript
}

type Component struct {
	OriWithInitVar    bool
	OriWithRunningVar bool
}

type StrComponent struct {
	Component
	ParsedValue string
}

type SMapComponent struct {
	Component
	ParsedValue map[string]string
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func dumpContext(ctx *boomer.RunContext){
	fmt.Printf("ctx=\n")
	fmt.Printf("  {\n")
	fmt.Printf("    .ID=%d\n",ctx.ID)
	fmt.Printf("    .RunSeq=%d\n",ctx.RunSeq)
	fmt.Printf("    .RspHead=%s\n",ctx.RspHead)
	fmt.Printf("    .RspCookie=%s\n",ctx.RspCookie)
	fmt.Printf("    .RspStatus=%d\n",ctx.RspStatus)
	fmt.Printf("    .RspJSON=%s\n",ctx.RspJSON)
	fmt.Printf("    .RspText=%s\n",ctx.RspText)
	fmt.Printf("    .Store=%s\n",ctx.Store)
	fmt.Printf("  }\n")
}

func dumpVarsData(vars *Variables){

	fmt.Printf("Variables:\n  \"Declare\":%s\n",strings.Join(vars.Declare,","))
	dumpStringArr("InitVariables",vars.InitVariables)
	dumpStringArr("RunningVariables",vars.RunningVariables)
	dumpStringArr("MergedVariables",vars.MergedVariables)
}
func dumpStringArr(key string,arr map[string]interface{}){
	fmt.Printf("  \"%s\":{\n",key)
	for k,v:=range(arr){
		fmt.Printf("      \"%s\":%v \n",k,v)
	}
	fmt.Printf("  }\n")
}
func dumpVars(dumpVars string){
	dumpVars=strings.Join(strings.Split(dumpVars,","),",\n  ")
	dumpVars=strings.Join(strings.Split(dumpVars,":{"),":{\n  ")
	dumpVars=strings.Join(strings.Split(dumpVars,":["),":[\n    ")
	dumpVars=strings.Join(strings.Split(dumpVars,"],"),"\n  ],")
	fmt.Println(dumpVars)
}

func (rs *RunScript) genVariables(ctx *boomer.RunContext) Variables {
	varsBytes, _ := jsoniter.Marshal(rs.Variables)
	vars := string(varsBytes)
	// fmt.Println(vars)
	//template.Must()
	t,tempError := template.New("Variables").Funcs(TemplateFunc).Parse(vars)
	if tempError!=nil{
		log.Println("@genVariables wrong template:")
		fmt.Println(vars)
		log.Fatal(tempError)
	}
	var tmpBytes bytes.Buffer
	errExec := t.Execute(&tmpBytes, ctx)
	if errExec!=nil{
		log.Println("@genVariables vars:")
		dumpContext(ctx)
		fmt.Println("@genVariables template:")
		dumpVars(vars)
		log.Println("@genVariables err:",errExec.Error())
		dumpVars(vars)
	}
	var variables Variables
	var tempStr=tmpBytes.String()
	// log.Println("@genVariables JSON:",vars)
	tempStr=strings.ReplaceAll(tempStr,"\"####\"","{}")
	tempStr=strings.ReplaceAll(tempStr,"\"##","")
	tempStr=strings.ReplaceAll(tempStr,"##\"","")

	

	decoder := jsoniter.NewDecoder(strings.NewReader(tempStr))
	decoder.UseNumber()
	err := decoder.Decode(&variables)
	// fmt.Println("next")
	if err != nil {
		log.Println("@genVariables variables:")
		dumpVarsData(&variables)
		log.Println("@genVariables errJSON:")
		dumpVars(vars)
		log.Fatal(err)
	}
	merged := make(map[string]interface{})
	for k, v := range variables.InitVariables {
		merged[k] = v
	}
	for k, v := range variables.RunningVariables {
		merged[k] = v
	}

	variables.MergedVariables = merged

	return variables
}

func (rs *RunScript) init() {
	if nil != rs.RunningVariables && len(rs.RunningVariables) > 0 {
		rs.WithRunningVar = true
	}
	if nil != rs.InitVariables && len(rs.InitVariables) > 0 {
		rs.WithInitVar = true
	}
}

func (fs *FuncSet) parseVars(rs RunScript) {
	fs.RScript = &rs
	// no variables
	if !fs.RScript.WithInitVar && !fs.RScript.WithRunningVar {
		fs.Parsed.Url.ParsedValue = fs.Url
		fs.Parsed.Body.ParsedValue = fs.Body
		fs.Parsed.Header.ParsedValue = fs.Header
		return
	}

	parsedUrl := fs.getURLWithWarn(rs.InitVariables,false)
	if strings.Contains(parsedUrl, NoValue) {
		fs.Parsed.Url.OriWithRunningVar = true
	}
	parsedUrl = fs.getURL(rs.RunningVariables)
	if strings.Contains(parsedUrl, NoValue) {
		fs.Parsed.Url.OriWithInitVar = true
	}

	parsedBody := fs.getBodyWithWarn(rs.InitVariables,false)
	if strings.Contains(parsedBody, NoValue) {
		fs.Parsed.Body.OriWithRunningVar = true
	}
	parsedBody = fs.getBody(rs.RunningVariables)
	if strings.Contains(parsedBody, NoValue) {
		fs.Parsed.Body.OriWithInitVar = true
	}
	parsedHeader := fs.getHeadersWithWarn(rs.InitVariables,false)
	for _, v := range parsedHeader {
		if strings.Contains(v, NoValue) {
			fs.Parsed.Header.OriWithRunningVar = true
		}
	}
	parsedHeader = fs.getHeaders(rs.RunningVariables)
	for _, v := range parsedHeader {
		if strings.Contains(v, NoValue) {
			fs.Parsed.Header.OriWithInitVar = true
		}
	}

}

func (fs *FuncSet) getURL(v Variable) string {
	return fs.getURLWithWarn(v,true)
}
func (fs *FuncSet) getURLWithWarn(v Variable,warn bool) string {
	tmpl, err := template.New("URL").Funcs(TemplateFunc).Parse(fs.Url)
	if err != nil {
		log.Println("@getURL error #1 parse:",fs.Url)
		panic(err)
	}
	var tmplBytes bytes.Buffer
	err = tmpl.Execute(&tmplBytes, v)
	if err != nil {
		if warn{
			if fs.RScript.Debug{
				log.Println("@getURL #2 vars:")
				dumpStringArr("Variable",v)
				log.Println("@getURL parse:")
				fmt.Println(fs.Url)
			}
		}
		// panic(err)
		return NoValue
	}
	return tmplBytes.String()
}

func (fs *FuncSet) getBody(v Variable) string {
	return fs.getBodyWithWarn(v,true)
}
func (fs *FuncSet) getBodyWithWarn(v Variable,warn bool) string {
	//.Option(fmt.Sprintf("missingkey=%s",NoValue))
	tmpl, err := template.New("Body").Funcs(TemplateFunc).Parse(fs.Body)
	if err != nil {
		log.Println("@getBody parse:",fs.Body)
		panic(err)
	}
	var tmplBytes bytes.Buffer
	err = tmpl.Execute(&tmplBytes, v)
	if err != nil {
		if warn{
			if fs.RScript.Debug{
				log.Println("@getBody #2 var:",v)
				log.Println("@getBody parse:",fs.Body)
			}
			log.Println("@getBody err:",err.Error())
		}

		// panic(err)
		return NoValue
	}
	return tmplBytes.String()
}

func (fs *FuncSet) getHeaders(v Variable) (hmap map[string]string) {
	return fs.getHeadersWithWarn(v,true)
}
func (fs *FuncSet) getHeadersWithWarn(v Variable,warn bool) (hmap map[string]string) {
	headerBytes, err := jsoniter.Marshal(fs.Header)
	tmpl, err := template.New("Header").Funcs(TemplateFunc).Parse(string(headerBytes))
	if err != nil {
		log.Println("@getHeaders #0 parse:",string(headerBytes))
		panic(err)
	}
	var tmplBytes bytes.Buffer
	err = tmpl.Execute(&tmplBytes, v)
	if err != nil {
		if warn{
			if fs.RScript.Debug{
				log.Println("@getHeaders vars:",v)
				log.Println("@getHeaders parse:",string(headerBytes))
			}
		}
		// if warn{
		// 	panic(err)
		// }

		hmap=make(map[string]string)
		hmap["_"]=NoValue
		return hmap
	}
	err = jsoniter.Unmarshal(tmplBytes.Bytes(), &hmap)
	if err != nil {
		log.Println("@getHeaders #2 parse:",fs.Header)
		panic(err)
	}
	return hmap
}

func (fs *FuncSet) assertTrue(mapping map[string]interface{}) bool {
	t := template.Must(template.New("Validator").Funcs(TemplateFunc).Parse(fs.Validator))
	var bs bytes.Buffer
	//for _, v := range mapping {
	//	fmt.Println(v, reflect.TypeOf(v))
	//}
	err := t.Execute(&bs, mapping)
	if err != nil {
		log.Println("@assertTrue Validator:",fs.Validator)
		panic(err)
	}
	return "true" == bs.String()
}
// assertConditionTrue 运行条件
func (fs *FuncSet) assertConditionTrue(mapping map[string]interface{}) bool {
	if len(fs.Condition)==0{
		return true
	}
	t := template.Must(template.New("Validator").Funcs(TemplateFunc).Parse(fs.Condition))
	var bs bytes.Buffer
	err := t.Execute(&bs, mapping)
	if err != nil {
		if fs.RScript.Debug{
			log.Println("@assertConditionTrue Condition ",fs.Condition)
			log.Println("@assertConditionTrue error ",err.Error())
		}
		return false
		// panic(err)
	}
	return "true" == bs.String()
}
// storeData 保存数据
func (fs *FuncSet) storeData(ctx *boomer.RunContext,mapping map[string]interface{})  {
	if len(fs.Store)==0{
		return  
	}

	varsBytes, _ := jsoniter.Marshal(fs.Store)
	vars := string(varsBytes)
	// fmt.Println(vars)
	//template.Must()
	t,tempError := template.New("store-"+fs.Key).Funcs(TemplateFunc).Parse(vars)
	if tempError!=nil{
		if fs.RScript.Debug{
			log.Println("@storeData wrong template:")
			fmt.Println(vars)
			log.Println("@storeData error:",tempError.Error())
		}
		return
	}
	//把ctx的转换过去

	mapping["ctx"]=ctx

	var tmpBytes bytes.Buffer
	errTemp := t.Execute(&tmpBytes, mapping)
	if errTemp!=nil{
		if fs.RScript.Debug{
			log.Println("@storeData template:",vars)
			log.Println("@storeData err:",errTemp.Error())
		}
		log.Println("@storeData vars:",mapping)
		log.Fatal(errTemp)
	}

	var variables = make(map[string]string)
	var tempStr=tmpBytes.String()

	decoder := jsoniter.NewDecoder(strings.NewReader(tempStr))
	decoder.UseNumber()
	err := decoder.Decode(&variables)
 
	if err != nil {
		if fs.RScript.Debug{
			log.Println("@storeData variables:",variables)
			log.Println("@storeData err:",err.Error())
		}
		log.Println("@storeData errJSON:",tempStr)
		log.Fatal(err)
	}
	for k, v := range variables {
		ctx.Store[k] = v
		if fs.RScript.Debug{
			log.Println("store save:",k,v)
		}
	}
}

func GetTaskList(baseJson string) []*boomer.Task {
	rs := RunScript{}
	err := jsoniter.Unmarshal([]byte(baseJson), &rs)
	if err != nil {
		log.Println("@GetTaskList baseJson:",baseJson)
		panic(err)
	}
	rs.init()
	var tasks []*boomer.Task
	for _, req := range rs.FuncSet {
		req.parseVars(rs)
		action := genReqAction(req)
		task := boomer.Task{
			Name:   req.Key,
			Weight: req.Probability,
			Fn:     action,
		}
		tasks = append(tasks, &task)
	}
	return tasks
}
