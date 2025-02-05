package httpwrapper

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/hpgood/boomer"
	jsoniter "github.com/json-iterator/go"
)

var client *http.Client
var verbose = true

func init() {

	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 2000
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		MaxIdleConnsPerHost: 2000,
		DisableCompression:  false,
		DisableKeepAlives:   false,
	}
	client = &http.Client{
		Transport: tr,
		Timeout:   time.Duration(10) * time.Second,
	}
}

// genReqAction genReqAction
func genReqAction(fs FuncSet) func(*boomer.RunContext) {

	variables := fs.RScript.genVariables(boomer.NewRunContext())
	initUrl := fs.getURL(variables.InitVariables)
	initBody := fs.getBody(variables.InitVariables)
	initHeaders := fs.getHeaders(variables.InitVariables)

	action := func(ctx *boomer.RunContext) {
		// log.Println("@genReqAction run ID=", ctx.ID)

		var loopNum = fs.Loop
		if loopNum <= 0 {
			loopNum = 1
		}

		ctx.TaskLoop = ctx.TaskLoopID < loopNum //是否循环执行下去

		var debug = fs.RScript.Debug || fs.Debug

		var url string
		var _body string
		var headers map[string]string

		fs.RScript.PreParsed = false
		runVariables := fs.RScript.genVariables(ctx)
		runVariables.MergedVariables["ctx"] = ctx

		if !fs.assertConditionTrue(runVariables.MergedVariables) {
			if debug {
				log.Println("assert condition false, ignore request:", fs.Key)
			}
			return
		}

		if !fs.RScript.WithInitVar && !fs.RScript.WithRunningVar {
			url = fs.Parsed.Url.ParsedValue
			_body = fs.Parsed.Body.ParsedValue
		} else {
			if !fs.Parsed.Url.OriWithRunningVar {
				url = initUrl
			} else {
				url = fs.getURL(runVariables.MergedVariables)
			}

			if !fs.Parsed.Body.OriWithRunningVar {
				_body = initBody
			} else {
				_body = fs.getBody(runVariables.MergedVariables)
			}

			if !fs.Parsed.Header.OriWithRunningVar {
				headers = initHeaders
			} else {
				headers = fs.getHeaders(runVariables.MergedVariables)
			}
		}

		domain := fs.RScript.Domain
		if strings.Contains(domain, "{{") && strings.Contains(domain, "}}") {
			domain = fs.getDomain(ctx)
		}

		fullURL := fmt.Sprintf("%s%s", domain, url)

		ctx.RspHead = "{}"   //head
		ctx.RspCookie = "{}" //cookie
		ctx.RspJSON = "{}"   //body json
		ctx.RspStatus = 0    //status
		ctx.RspText = ""     //body

		// if verbose {
		// 	log.Println("body:",body)
		// }
		request, err := http.NewRequest(fs.Method, fullURL, bytes.NewBuffer([]byte(_body)))
		if err != nil {
			log.Fatalf("%v\n", err)
		}

		for k, v := range initHeaders {
			if k != "_" {
				request.Header.Set(k, v)
			}
		}

		for k, v := range headers {
			if k != "_" {
				request.Header.Set(k, v)
			}
		}

		if debug {
			if len(fs.Name) > 0 {
				log.Println(fs.Name)
			}
			log.Println(formatRequest(request))
		}

		startTime := time.Now()
		response, err := client.Do(request)
		elapsed := time.Since(startTime)

		if err != nil {
			if verbose {
				log.Printf("%v\n", err)
			}
			boomer.RecordFailure(fs.Method, fs.Key, 0.0, err.Error())
		} else {
			ctx.RspStatus = response.StatusCode
			retBody, err := ioutil.ReadAll(response.Body)
			if err != nil {
				log.Printf("%v\n", err)
			} else {
				var res map[string]interface{}
				errJSON := jsoniter.Unmarshal(retBody, &res)
				res["http_status_code"] = response.StatusCode

				//保存上个接口的数据
				// ctx.Data["rsp_status_code"] = strconv.Itoa(response.StatusCode)

				retBodyStr := string(retBody)
				ctx.RspText = retBodyStr

				var head = make(map[string]string)
				headJSON := "{}"
				headCount := 0
				for k, v := range response.Header {
					if strings.HasPrefix(strings.ToLower(k), "x-") {
						key := strings.Replace(k, "-", "", -1)
						// ctx.Data[key]= strings.Join(v,",")
						head[key] = strings.Join(v, ",")
						if debug {
							log.Printf(".rspHead.%s, response header %s=%s\n", key, k, v)
						}
						headCount++
					}
				}
				if headCount > 0 {
					headJSONByte, _ := jsoniter.Marshal(&head)
					headJSON = string(headJSONByte)
				}

				ctx.RspHead = headJSON

				if errJSON == nil {
					ctx.RspJSON = retBodyStr
				} else {
					ctx.RspJSON = "{}"
				}
				// cookie
				var cookies = make(map[string]string)
				cookieJSON := "{}"
				cookieCount := 0

				for _, ck := range response.Cookies() {
					cookies[ck.Name] = ck.Value
					cookieCount++
				}

				if headCount > 0 {
					cookieJSONByte, _ := jsoniter.Marshal(&cookies)
					cookieJSON = string(cookieJSONByte)
				}
				ctx.RspCookie = cookieJSON

				merged := make(map[string]interface{})
				for k, v := range runVariables.MergedVariables {
					merged[k] = v
				}
				for k, v := range res {
					merged[k] = v
				}

				if debug {
					log.Printf("Status Code: %d\n", response.StatusCode)
					log.Println(retBodyStr)

				} else {
					io.Copy(ioutil.Discard, response.Body)
				}

				if fs.assertTrue(merged) {
					if debug {
						log.Println("assert true,time=", elapsed.Nanoseconds()/int64(time.Millisecond), "ms")
					}
					boomer.RecordSuccess(fs.Method, fs.Key,
						elapsed.Nanoseconds()/int64(time.Millisecond), response.ContentLength)
				} else {
					msg := fmt.Sprintf("assert failed,id:%d,url:%s,data:%s,response:%s", ctx.ID, url, _body, retBodyStr)
					boomer.RecordFailure(fs.Method, fs.Key, elapsed.Nanoseconds()/int64(time.Millisecond), msg)
				}
				//保存数据
				fs.storeData(ctx, merged)
			}

			response.Body.Close()

		}

	} //
	return action
}

func formatRequest(r *http.Request) string {
	data, err := httputil.DumpRequest(r, true)
	if err != nil {
		log.Fatal("Error")
	}
	return string(data)
}
