package apib

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
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
var currentParams []*Param

type Call struct {
	Group    string
	Name     string
	Request  Request
	Response Response
}

type Resources []Resource
type Resource struct {
	Name        string
	URI         string
	Call        Call
	Params      map[string][]string
	ExtraParams []*Param
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

type Param struct {
	Name        string
	Example     string
	Description string
	Type        string
	Required    bool
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
		for _, rs := range rss {
			c := rs.Call

			f.Write([]byte(fmt.Sprintf("## %s [%s %s]\n\n", c.Name, c.Request.Method, c.Request.URI)))
			if len(rs.Params) > 0 || len(rs.ExtraParams) > 0 {
				f.Write([]byte(fmt.Sprintf("+ Parameters\n\n")))
				if len(rs.Params) > 0 {
					for k, v := range rs.Params {
						for _, p := range v {
							f.Write([]byte(fmt.Sprintf("    + %s: `%s` (string)\n", url.QueryEscape(k), p)))
						}
					}
				}
				if len(rs.ExtraParams) > 0 {
					for _, p := range rs.ExtraParams {
						f.Write([]byte(fmt.Sprintf("    + %s: `%s` (%s, optional)\n", url.QueryEscape(p.Name), p.Example, p.Type)))
						if p.Description != "" {
							desc := stringToLines(p.Description)
							for i, s := range desc {
								desc[i] = "      " + s + "  \n"
							}
							f.Write([]byte(fmt.Sprintf("%s\n", strings.Join(desc, ""))))
						}
					}
				}
			}
			f.Write([]byte("\n"))

			// f.Write([]byte(fmt.Sprintf("### %s [%s]\n\n\n", c.Name, c.Request.Method)))
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

func AddParam(name string, typ string, ex string, desc string, req bool) {
	if currentParams == nil {
		currentParams = []*Param{}
	}
	currentParams = append(currentParams, &Param{
		Name:        name,
		Type:        typ,
		Example:     ex,
		Description: desc,
		Required:    req,
	})
}

func Flush() {
	currentGroup = ""
	currentName = ""
	currentParams = []*Param{}
}

func params(c echo.Context) map[string]string {
	params := make(map[string]string)

	for _, p := range c.ParamNames() {
		params[p] = c.Param(p)
	}

	return params
}
