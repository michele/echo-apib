package echo_apib

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/acsellers/inflections"
	"github.com/labstack/echo"
)

var calls = make(map[string]Resources)
var recording bool
var currentGroup string
var currentName string

type Call struct {
	Group    string
	Name     string
	Request  Request
	Response Response
}

type Resources map[string]Resource
type Resource struct {
	URI    string
	Calls  []Call
	Params map[string][]string
}
type Request struct {
	URI        string
	Method     string
	Headers    map[string]string
	Params     map[string][]string
	Body       string
	PathParams map[string]string
}

type Response struct {
	Headers    map[string]string
	Body       string
	StatusCode int
}

func Record() {
	recording = true
}

func Store() {
	spaces := regexp.MustCompile(" ")
	recording = false

	for group, rss := range calls {
		fileName := inflections.Underscore(group)
		fileName = spaces.ReplaceAllString(fileName, "_")
		f, err := os.Create(fileName + ".apib")
		if err != nil {
			log.Fatal("Can't create file.", err)
		}
		f.Write([]byte(fmt.Sprintf("# Group %s\n\n", group)))
		for uri, rs := range rss {

			f.Write([]byte(fmt.Sprintf("## %s\n\n", uri)))
			if len(rs.Params) > 0 {
				f.Write([]byte(fmt.Sprintf("+ Parameters\n\n")))
				for k, v := range rs.Params {
					for _, p := range v {
						f.Write([]byte(fmt.Sprintf("    + %s: `%s` (string)\n", k, p)))
					}
				}
				f.Write([]byte("\n"))
			}
			for _, c := range rs.Calls {

				f.Write([]byte(fmt.Sprintf("### %s [%s]\n\n\n", c.Name, c.Request.Method)))
				f.Write([]byte(fmt.Sprintf("+ Request (%s)\n\n", c.Request.Headers["Content-Type"])))
				if len(c.Request.Headers) > 0 {
					f.Write([]byte(fmt.Sprintf("    + Headers\n\n")))
					for k, v := range c.Request.Headers {
						f.Write([]byte(fmt.Sprintf("            %s: %s\n", k, v)))
					}
					f.Write([]byte("\n"))
				}

				if len(c.Request.Body) > 0 {
					f.Write([]byte(fmt.Sprintf("    + Body\n\n")))
					body := stringToLines(c.Request.Body)
					for i, s := range body {
						body[i] = "            " + s
					}
					f.Write([]byte(fmt.Sprintf("            \n%s\n            \n\n", strings.Join(body, "\n"))))
				}
				f.Write([]byte("\n"))
				f.Write([]byte(fmt.Sprintf("+ Response %d (%s)\n\n", c.Response.StatusCode, c.Response.Headers["Content-Type"])))
				if len(c.Response.Headers) > 0 {
					f.Write([]byte(fmt.Sprintf("    + Headers\n\n")))
					for k, v := range c.Response.Headers {
						f.Write([]byte(fmt.Sprintf("            %s: %s\n", k, v)))
					}
					f.Write([]byte("\n"))
				}
				if len(c.Response.Body) > 0 {
					f.Write([]byte(fmt.Sprintf("    + Body\n\n")))
					body := stringToLines(c.Response.Body)
					for i, s := range body {
						body[i] = "            " + s
					}
					f.Write([]byte(fmt.Sprintf("            \n%s\n            \n\n", strings.Join(body, "\n"))))
				}
				f.Write([]byte("\n"))
			}
		}
		f.Close()
	}
}

func stringToLines(s string) []string {
	var lines []string

	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

	return lines
}

func Group(group string) {
	currentGroup = group
}

func Name(name string) {
	currentName = name
}

func ApibGenerator(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if recording && currentGroup != "" && currentName != "" {
			bodyBytes, _ := ioutil.ReadAll(c.Request().Body)
			c.Request().Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
			request := Request{
				URI:     c.Request().URL.RequestURI(),
				Method:  c.Request().Method,
				Body:    string(bodyBytes),
				Headers: make(map[string]string),
				Params:  make(map[string][]string),
			}
			for k, v := range c.Request().Header {
				request.Headers[k] = v[0]
			}

			for k, v := range c.QueryParams() {
				request.Params[k] = v
			}

			if len(params(c)) > 0 {
				for k, v := range params(c) {
					request.Params[k] = []string{v}
					request.URI = strings.Replace(request.URI, "/"+v, "/{"+k+"}", 1)
				}
			}
			call := Call{
				Group:   currentGroup,
				Name:    currentName,
				Request: request,
			}
			res := c.Response()
			rw := res.Writer
			w := NewWriter(rw)
			c.Response().Writer = w
			err := next(c)
			var response Response
			if err == nil {
				response = Response{
					Body:       w.Body(),
					StatusCode: c.Response().Status,
					Headers:    make(map[string]string),
				}
			}
			for k, v := range c.Response().Header() {
				response.Headers[k] = v[0]
			}
			call.Response = response
			if calls[currentGroup] == nil {
				calls[currentGroup] = Resources{}
			}
			rss := calls[currentGroup][request.URI]
			if rss.Calls == nil {
				rss.Calls = []Call{}
				rss.Params = make(map[string][]string)
			}
			rss.Params = request.Params
			rss.Calls = append(rss.Calls, call)
			calls[currentGroup][request.URI] = rss
			currentName = ""
			currentGroup = ""
			return nil
		}
		return next(c)
	}
}

type Writer struct {
	http.ResponseWriter
	text        string
	wroteHeader bool
}

func NewWriter(res http.ResponseWriter) *Writer {
	return &Writer{
		ResponseWriter: res,
	}
}

func (aw *Writer) Write(b []byte) (int, error) {
	if !aw.wroteHeader {
		aw.WriteHeader(http.StatusOK)
	}
	aw.text = aw.text + string(b)
	return aw.ResponseWriter.Write(b)
}

func (aw *Writer) WriteHeader(code int) {
	aw.ResponseWriter.WriteHeader(code)
	if aw.wroteHeader {
		return
	}
	aw.wroteHeader = true
}

func (aw *Writer) Body() string {
	return aw.text
}

func params(c echo.Context) map[string]string {
	params := make(map[string]string)

	for _, p := range c.ParamNames() {
		params[p] = c.Param(p)
	}

	return params
}
