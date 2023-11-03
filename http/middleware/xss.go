package middleware

import (
	"html"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/utils/serialize/json"
)

func XSS(whitelistURLs []string) gin.HandlerFunc {
	// Do this once for each unique policy, and use the policy for the life of the
	// program Policy creation/editing is not safe to use in multiple goroutines.
	p := bluemonday.UGCPolicy()

	return func(c *gin.Context) {
		for _, u := range whitelistURLs {
			if strings.HasPrefix(c.Request.URL.String(), u) {
				c.Next()
				return
			}
		}

		sanitizedQuery, err := xssFilterQuery(p, c.Request.URL.RawQuery)
		if err != nil {
			err = errors.Wrap(err, "filter query")
			_ = c.Error(err)
			c.Abort()
			return
		}
		c.Request.URL.RawQuery = sanitizedQuery

		var sanitizedBody string
		body, err := c.GetRawData()
		if err != nil {
			err = errors.Wrap(err, "read body")
			_ = c.Error(err)
			c.Abort()
			return
		}

		// xssFilterJSON() will return error when body is empty.
		if len(body) == 0 {
			c.Next()
			return
		}

		switch binding.Default(c.Request.Method, c.ContentType()) {
		case binding.JSON:
			if sanitizedBody, err = xssFilterJSON(p, string(body)); err != nil {
				err = errors.Wrap(err, "filter json")
			}
		case binding.FormMultipart:
			sanitizedBody = xssFilterPlain(p, string(body))
		case binding.Form:
			if sanitizedBody, err = xssFilterQuery(p, string(body)); err != nil {
				err = errors.Wrap(err, "filter form")
			}
		}
		if err != nil {
			_ = c.Error(err)
			c.Abort()
			return
		}

		c.Request.Body = ioutil.NopCloser(strings.NewReader(sanitizedBody))
		c.Next()
	}
}

func xssFilterQuery(p *bluemonday.Policy, s string) (string, error) {
	values, err := url.ParseQuery(s)
	if err != nil {
		return "", err
	}

	for k, v := range values {
		values.Del(k)
		for _, vv := range v {
			values.Add(k, xssFilterPlain(p, vv))
		}
	}

	return values.Encode(), nil
}

func xssFilterJSON(p *bluemonday.Policy, s string) (string, error) {
	var data any
	if err := json.Unmarshal([]byte(s), &data); err != nil {
		return "", err
	}

	b := strings.Builder{}
	e := json.NewEncoder(&b)
	e.SetEscapeHTML(false)
	if err := e.Encode(xssFilterJSONData(p, data)); err != nil {
		return "", err
	}
	// use `TrimSpace` to trim newline char add by `Encode`.
	return strings.TrimSpace(b.String()), nil
}

func xssFilterJSONData(p *bluemonday.Policy, d any) any {
	switch data := d.(type) {
	case []any:
		for i, v := range data {
			data[i] = xssFilterJSONData(p, v)
		}
		return data
	case map[string]any:
		for k, v := range data {
			data[k] = xssFilterJSONData(p, v)
		}
		return data
	case string:
		return xssFilterPlain(p, data)
	default:
		return data
	}
}

func xssFilterPlain(p *bluemonday.Policy, s string) string {
	sanitized := p.Sanitize(s)
	return html.UnescapeString(sanitized)
}
