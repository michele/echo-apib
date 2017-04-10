package apib

import (
	"bytes"
	"io/ioutil"
	"strings"

	"github.com/labstack/echo"
)

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
