package main

import (
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/kdar/factorlog"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"time"
)

var (
	logFlag  = flag.String("log", "", "set log path")
	portFlag = flag.Int("port", 5000, "set port")
	urlFlag  = flag.String("url", "", "set proxy target url")

	logger *log.FactorLog
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func SetGlobalLogger(logPath string) *log.FactorLog {
	sfmt := `%{Color "red:white" "CRITICAL"}%{Color "red" "ERROR"}%{Color "yellow" "WARN"}%{Color "green" "INFO"}%{Color "cyan" "DEBUG"}%{Color "blue" "TRACE"}[%{Date} %{Time}] [%{SEVERITY}:%{ShortFile}:%{Line}] %{Message}%{Color "reset"}`
	logger := log.New(os.Stdout, log.NewStdFormatter(sfmt))
	if len(logPath) > 0 {
		logf, err := os.OpenFile(logPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0640)
		if err != nil {
			return logger
		}
		logger = log.New(logf, log.NewStdFormatter(sfmt))
	}
	logger.SetSeverities(log.INFO | log.WARN | log.ERROR | log.FATAL | log.CRITICAL)
	return logger
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Parse()

	logger = SetGlobalLogger(*logFlag)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.GET("/*path", getHander)

	logger.Fatal(router.Run(fmt.Sprintf(":%d", *portFlag)))
}

type Jar struct {
	cookies []*http.Cookie
}

func NewJar() *Jar {
	return &Jar{
		cookies: []*http.Cookie{},
	}
}
func (jar *Jar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	jar.cookies = cookies
}
func (jar *Jar) Cookies(u *url.URL) []*http.Cookie {
	return jar.cookies
}

func getHander(c *gin.Context) {
	path := c.Param("path")
	u, err := url.Parse(fmt.Sprintf("%s%s", *urlFlag, path))
	if err != nil {
		logger.Error(err)
		c.String(http.StatusInternalServerError, fmt.Sprintf("Error: %s", err.Error()))
		return
	}
	q := u.Query()
	for k, v := range c.Request.URL.Query() {
		if len(v) > 0 {
			q.Set(k, v[0])
		}
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(c.Request.Method, u.String(), http.NoBody)
	if err != nil {
		logger.Error(err)
		c.String(http.StatusInternalServerError, fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	for _, v := range c.Request.Cookies() {
		req.AddCookie(v)
	}

	jar := NewJar()
	jar.SetCookies(u, c.Request.Cookies())

	client := &http.Client{
		Jar: jar,
	}
	req.Header = c.Request.Header

	res, err := client.Do(req)
	if err != nil {
		logger.Error(err)
		c.String(http.StatusInternalServerError, fmt.Sprintf("Error: %s", err.Error()))
		return
	}
	result, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		logger.Error(err)
		c.String(http.StatusInternalServerError, fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")             //允许访问所有域
	c.Writer.Header().Add("Access-Control-Allow-Headers", "Content-Type") //header的类型

	c.String(res.StatusCode, "%s", string(result))
}
