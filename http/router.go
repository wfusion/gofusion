package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"reflect"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/spf13/cast"

	"github.com/wfusion/gofusion/common/constant"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/clone"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/http/gracefully"
	"github.com/wfusion/gofusion/http/parser"
	"github.com/wfusion/gofusion/routine"
)

type parseFrom int

const (
	parseFromBody parseFrom = 1 + iota
	parseFromQuery
)

type dispatch int

const (
	dispatchIRouter dispatch = iota
	dispatchGroup
	dispatchRoutes
)

type routerHandler any
type routerRequestParser func(*gin.Context, reflect.Type) (reflect.Value, error)

var (
	methodWithBody = map[string]bool{
		http.MethodPut:   true,
		http.MethodPost:  true,
		http.MethodPatch: true,
	}
)

type router struct {
	gin.IRouter

	open         chan struct{}
	close        chan struct{}
	ctx          context.Context
	appName      string
	successCode  int
	errorCode    Errcode
	shutdownFunc func()
	metricsConf  metricsConf

	routes gin.IRoutes      `optional:"true"`
	group  *gin.RouterGroup `optional:"true"`
	ptr    dispatch         `optional:"true"`
}

func newRouter(ctx context.Context, r gin.IRouter, appName string, successCode, errorCode int) IRouter {
	return &router{
		IRouter:     r,
		ctx:         ctx,
		open:        make(chan struct{}),
		close:       make(chan struct{}),
		appName:     appName,
		successCode: successCode,
		errorCode:   Errcode(errorCode),
	}
}

func (r *router) Use(middlewares ...gin.HandlerFunc) IRouter {
	return &router{
		IRouter:      r.IRouter,
		open:         r.open,
		close:        r.close,
		ctx:          r.ctx,
		appName:      r.appName,
		successCode:  r.successCode,
		errorCode:    r.errorCode,
		shutdownFunc: r.shutdownFunc,
		routes:       r.use().Use(middlewares...),
		group:        r.group,
		ptr:          dispatchRoutes,
	}
}

func (r *router) Handle(uri string, fn routerHandler, opts ...utils.OptionExtender) IRouter {
	opt := utils.ApplyOptions[routerOption](opts...)
	r.use().HEAD(uri, r.convertMulti("Handle", uri, fn, opt)...)
	return r
}
func (r *router) Any(uri string, fn routerHandler, opts ...utils.OptionExtender) IRouter {
	opt := utils.ApplyOptions[routerOption](opts...)
	r.use().Any(uri, r.convertMulti("Any", uri, fn, opt)...)
	return r
}
func (r *router) GET(uri string, fn routerHandler, opts ...utils.OptionExtender) IRouter {
	opt := utils.ApplyOptions[routerOption](opts...)
	r.use().GET(uri, r.convertMulti(http.MethodGet, uri, fn, opt)...)
	return r
}
func (r *router) POST(uri string, fn routerHandler, opts ...utils.OptionExtender) IRouter {
	opt := utils.ApplyOptions[routerOption](opts...)
	r.use().POST(uri, r.convertMulti(http.MethodPost, uri, fn, opt)...)
	return r
}
func (r *router) DELETE(uri string, fn routerHandler, opts ...utils.OptionExtender) IRouter {
	opt := utils.ApplyOptions[routerOption](opts...)
	r.use().DELETE(uri, r.convertMulti(http.MethodDelete, uri, fn, opt)...)
	return r
}
func (r *router) PATCH(uri string, fn routerHandler, opts ...utils.OptionExtender) IRouter {
	opt := utils.ApplyOptions[routerOption](opts...)
	r.use().PATCH(uri, r.convertMulti(http.MethodPatch, uri, fn, opt)...)
	return r
}
func (r *router) PUT(uri string, fn routerHandler, opts ...utils.OptionExtender) IRouter {
	opt := utils.ApplyOptions[routerOption](opts...)
	r.use().PUT(uri, r.convertMulti(http.MethodPut, uri, fn, opt)...)
	return r
}
func (r *router) OPTIONS(uri string, fn routerHandler, opts ...utils.OptionExtender) IRouter {
	opt := utils.ApplyOptions[routerOption](opts...)
	r.use().OPTIONS(uri, r.convertMulti(http.MethodOptions, uri, fn, opt)...)
	return r
}
func (r *router) HEAD(uri string, fn routerHandler, opts ...utils.OptionExtender) IRouter {
	opt := utils.ApplyOptions[routerOption](opts...)
	r.use().HEAD(uri, r.convertMulti(http.MethodHead, uri, fn, opt)...)
	return r
}
func (r *router) Group(relativePath string, handlers ...gin.HandlerFunc) IRouter {
	return &router{
		IRouter:      r.IRouter,
		open:         r.open,
		close:        r.close,
		ctx:          r.ctx,
		appName:      r.appName,
		successCode:  r.successCode,
		errorCode:    r.errorCode,
		shutdownFunc: r.shutdownFunc,
		routes:       r.routes,
		group:        r.useIRouter().Group(relativePath, handlers...),
		ptr:          dispatchGroup,
	}
}

func (r *router) StaticFile(uri, file string) IRouter { r.use().StaticFile(uri, file); return r }
func (r *router) StaticFileFS(uri, file string, fs http.FileSystem) IRouter {
	r.use().StaticFileFS(uri, file, fs)
	return r
}
func (r *router) Static(uri, file string) IRouter { r.use().Static(uri, file); return r }
func (r *router) StaticFS(uri string, fs http.FileSystem) IRouter {
	r.use().StaticFS(uri, fs)
	return r
}

func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.IRouter.(*gin.Engine).ServeHTTP(w, req)
}
func (r *router) ListenAndServe() (err error) {
	if _, ok := utils.IsChannelClosed(r.open); ok {
		<-r.Closing()
		return
	}

	conf := r.Config()
	gracefully.DefaultReadTimeOut = conf.ReadTimeout
	gracefully.DefaultWriteTimeOut = conf.WriteTimeout
	gracefully.DefaultMaxHeaderBytes = 1 << 20 // use http.DefaultMaxHeaderBytes - which currently is 1 << 20 (1MB)

	port := fmt.Sprintf(":%v", conf.Port)
	srv := gracefully.NewServer(r.appName, r.IRouter.(*gin.Engine), port, conf.NextProtos)
	r.shutdownFunc = srv.Shutdown

	r.close = make(chan struct{})
	close(r.open)
	defer func() {
		close(r.close)
		r.open = make(chan struct{})
	}()

	if conf.TLS {
		return srv.ListenAndServeTLS(conf.Cert, conf.Key)
	} else {
		return srv.ListenAndServe()
	}
}
func (r *router) Start() {
	if _, ok := utils.IsChannelClosed(r.open); ok {
		return
	}
	conf := r.Config()
	gracefully.DefaultReadTimeOut = conf.ReadTimeout
	gracefully.DefaultWriteTimeOut = conf.WriteTimeout
	gracefully.DefaultMaxHeaderBytes = 1 << 20 // use http.DefaultMaxHeaderBytes - which currently is 1 << 20 (1MB)

	port := fmt.Sprintf(":%v", conf.Port)
	srv := gracefully.NewServer(r.appName, r.IRouter.(*gin.Engine), port, conf.NextProtos)
	r.shutdownFunc = srv.Shutdown
	if conf.TLS {
		routine.Go(srv.ListenAndServeTLS, routine.Args(conf.Cert, conf.Key), routine.AppName(r.appName))
	} else {
		routine.Go(srv.ListenAndServe, routine.AppName(r.appName))
	}

	r.close = make(chan struct{})
	close(r.open)
}
func (r *router) Config() OutputConf {
	cfg := new(Conf)
	_ = config.Use(r.appName).LoadComponentConfig(config.ComponentHttp, cfg)

	return OutputConf{
		Port:         cfg.Port,
		TLS:          cfg.TLS,
		Cert:         cfg.Cert,
		Key:          cfg.Key,
		NextProtos:   cfg.NextProtos,
		SuccessCode:  cfg.SuccessCode,
		ReadTimeout:  utils.Must(time.ParseDuration(cfg.ReadTimeout)),
		WriteTimeout: utils.Must(time.ParseDuration(cfg.WriteTimeout)),
		AsynqConf:    clone.Slowly(cfg.Asynq),
	}
}
func (r *router) Running() <-chan struct{} { return r.open }
func (r *router) Closing() <-chan struct{} { return r.close }

func (r *router) shutdown() {
	if r.close != nil {
		if _, ok := utils.IsChannelClosed(r.close); ok {
			return
		}
	}
	if r.shutdownFunc != nil {
		r.shutdownFunc()
	}
	if r.close != nil {
		close(r.close)
	}

	r.open = make(chan struct{})
}

func (r *router) use() gin.IRoutes {
	switch r.ptr {
	case dispatchIRouter:
		return r.IRouter
	case dispatchGroup:
		return r.group
	case dispatchRoutes:
		return r.routes
	default:
		return r.IRouter
	}
}

func (r *router) useIRouter() gin.IRouter {
	switch r.ptr {
	case dispatchIRouter:
		return r.IRouter
	case dispatchGroup:
		return r.group
	case dispatchRoutes:
		panic(errors.New("group method unsupported for gin.Routes interface"))
	default:
		return r.IRouter
	}
}

// convert
// Warning: MultipartFormDataBody only support Struct or *Struct
// support router handler signature as follows:
// - be compatible with native func(c *gin.Context) without any in&out parameters parsed
// - func(c *gin.Context, req Struct FromQuery) error
// - func(c *gin.Context, req Struct FromJsonBody) error
// - func(c *gin.Context, req Struct FromFormUrlDecodedBody) error
// - func(c *gin.Context, req Struct FromMultipartFormDataBody) error
// - func(c *gin.Context, req *Struct FromQuery) error
// - func(c *gin.Context, req *Struct FromParam) error
// - func(c *gin.Context, req *Struct FromJsonBody) error
// - func(c *gin.Context, req *Struct FromFormUrlDecodedBody) error
// - func(c *gin.Context, req *Struct FromMultipartFormDataBody) error
// - func(c *gin.Context, req map[string]any FromQuery) error
// - func(c *gin.Context, req map[string]any FromJsonBody) error
// - func(c *gin.Context, req map[string]any FromFormUrlDecodedBody) error
// - func(c *gin.Context, req []map[string]any FromQuery) error
// - func(c *gin.Context, req []map[string]any FromJsonBody) error
// - func(c *gin.Context, req []map[string]any FromFormUrlDecodedBody) error
// - func(c *gin.Context, req *FromQuery) (rsp *Struct{Data any; Page, Count int; Msg string}, err error)
// - func(c *gin.Context, req *FromQuery) (rsp *Struct{Embed}, err error)
// - func(c *gin.Context, req *FromQuery) (data any, page, count int, msg string, err error)
// - class.public.func(c *gin.Context, req Struct FromQuery) error
// - class.private.func(c *gin.Context, req Struct FromQuery) error
func (r *router) convert(method, uri string, handler routerHandler, opt *routerOption) gin.HandlerFunc {
	// check IRouter handler type
	typ := reflect.TypeOf(handler)
	if err := r.checkHandlerType(method, uri, typ); err != nil {
		panic(err)
	}

	// return raw gin handler
	if typ.NumIn() == 1 && typ.NumOut() == 0 {
		return func(c *gin.Context) {
			reflect.ValueOf(handler).Call([]reflect.Value{reflect.ValueOf(c)})
			c.Next()
		}
	}

	// parse request&response
	if typ.NumIn() == 1 {
		return r.wrapHandlerFunc(handler, nil)
	}

	parseMap := map[parseFrom]routerRequestParser{
		parseFromBody:  r.parseReqFromBody,
		parseFromQuery: r.parseReqFromQuery,
	}

	var reqParse routerRequestParser
	if p, ok := parseMap[opt.parseFrom]; ok {
		reqParse = p
	} else if methodWithBody[method] {
		reqParse = r.parseReqFromBody
	} else {
		reqParse = r.parseReqFromQuery
	}

	return r.wrapHandlerFunc(handler, reqParse)
}

func (r *router) convertMulti(method, uri string, hdr routerHandler, opt *routerOption) (result gin.HandlersChain) {
	result = make(gin.HandlersChain, 0, len(opt.beforeHandlers)+len(opt.aftersHandlers)+1)
	for _, hdr := range opt.beforeHandlers {
		result = append(result, r.convert(method, uri, hdr, opt))
	}
	result = append(result, r.convert(method, uri, hdr, opt))
	for _, hdr := range opt.aftersHandlers {
		result = append(result, r.convert(method, uri, hdr, opt))
	}
	return
}

func (r *router) checkHandlerType(method, uri string, typ reflect.Type) (err error) {
	if typ.Kind() != reflect.Func {
		return errors.Errorf("router handler should be a function [method[%s] uri[%s]]", method, uri)
	}

	// check in
	if typ.NumIn() < 1 {
		return errors.Errorf("router handler should have at least 1 parameter in "+
			"[method[%s] uri[%s]]", method, uri)
	}
	if typ.NumIn() > 2 {
		return errors.Errorf("router handler should not have more than 2 parameters in "+
			"[method[%s] uri[%s]]", method, uri)
	}
	if typ.In(0) != constant.GinContextType {
		return errors.Errorf("router handler first parameter in should be *gin.Context "+
			"[method[%s] uri[%s]]", method, uri)
	}
	if typ.NumIn() == 2 {
		if !r.checkParamType(typ.In(1), supportParamType) {
			return errors.Errorf("router handler second parameter in type not supportted "+
				"[method[%s] uri[%s]]", method, uri)
		}
	}

	// check out

	// check error
	if typ.NumOut() > 0 && !typ.Out(typ.NumOut()-1).AssignableTo(constant.ErrorType) {
		return errors.Errorf("router handler last paramater out should be error type "+
			"[method[%s] uri[%s]]", method, uri)
	}

	// check (data any, page, count int, msg string, err error)
	if numOut := typ.NumOut(); numOut > 1 {
		supportTypes := supportReturnTypeList[numOut-1]
		for i := 0; i < numOut-1; i++ {
			if !r.checkParamType(typ.Out(i), supportTypes[i]) {
				return errors.Errorf("router handler paramater out format is illegal "+
					"[method[%s] uri[%s] index[%v] unsupported[%s] suppoted[%+v]]",
					method, uri, i+1, typ.Out(i).Kind(), supportTypes[i])
			}
		}
	}

	return
}

var (
	supportReturnTypeList = map[int][]map[reflect.Kind]struct{}{
		1: {supportDataType},                                                 // data
		2: {supportDataType, supportMsgType},                                 // data, msg
		3: {supportDataType, supportIntType, supportMsgType},                 // data, count, msg
		4: {supportDataType, supportIntType, supportIntType, supportMsgType}, // data, page, count, msg
	}
	supportIntType = map[reflect.Kind]struct{}{
		reflect.Int:   {},
		reflect.Int8:  {},
		reflect.Int16: {},
		reflect.Int32: {},
		reflect.Int64: {},
	}
	supportDataType = map[reflect.Kind]struct{}{
		reflect.Map:       {},
		reflect.Array:     {},
		reflect.Slice:     {},
		reflect.Struct:    {},
		reflect.Interface: {},
	}
	supportMsgType = map[reflect.Kind]struct{}{
		reflect.String: {},
	}
	supportParamType = map[reflect.Kind]struct{}{
		reflect.Map:    {},
		reflect.Array:  {},
		reflect.Slice:  {},
		reflect.Struct: {},
	}
)

func (r *router) checkParamType(typ reflect.Type, supportType map[reflect.Kind]struct{}) bool {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	_, ok := supportType[typ.Kind()]
	return ok
}

func (r *router) parseReqFromBody(c *gin.Context, typ reflect.Type) (dst reflect.Value, err error) {
	ptrDepth := 0
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		ptrDepth++
	}
	defer func() {
		for ptrDepth > 0 {
			dst = dst.Addr()
			ptrDepth--
		}
	}()

	bodyBytes, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	dst = reflect.Indirect(reflect.New(typ))
	if err = c.ShouldBind(dst.Addr().Interface()); err != nil &&
		!(errors.Is(err, binding.ErrConvertToMapString) || (errors.Is(err, binding.ErrConvertMapStringSlice))) {
		err = parseGinBindingValidatorError(err)
		return
	}
	defer func() { c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) }()

	if dst.IsZero() {
		var (
			p           parser.Parser
			param       map[string]string
			contentType string
		)
		if contentType, param, err = mime.ParseMediaType(c.GetHeader("Content-Type")); err != nil {
			return
		}
		if p, err = parser.GetByContentType(contentType); err != nil {
			return
		}
		if err = p.PreParse(param); err != nil {
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		if err = p.Parse(c.Request.Body, dst); err != nil {
			return
		}
	}

	err = utils.ParseTag(
		dst.Addr().Interface(),
		utils.ParseTagName("default"),
		utils.ParseTagUnmarshalType(utils.UnmarshalTypeYaml),
	)

	return
}

func (r *router) parseReqFromQuery(c *gin.Context, typ reflect.Type) (dst reflect.Value, err error) {
	ptrDepth := 0
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		ptrDepth++
	}
	defer func() {
		for ptrDepth > 0 {
			dst = dst.Addr()
			ptrDepth--
		}
	}()

	dst = reflect.Indirect(reflect.New(typ))
	if err = c.ShouldBindUri(dst.Addr().Interface()); err != nil &&
		!(errors.Is(err, binding.ErrConvertToMapString) || (errors.Is(err, binding.ErrConvertMapStringSlice))) {
		err = parseGinBindingValidatorError(err)
		return
	}
	if err = c.ShouldBindQuery(dst.Addr().Interface()); err != nil &&
		!(errors.Is(err, binding.ErrConvertToMapString) || (errors.Is(err, binding.ErrConvertMapStringSlice))) {
		err = parseGinBindingValidatorError(err)
		return
	}
	// support query with json tag
	if dst.IsZero() {
		m := make(map[string][]string)
		for _, v := range c.Params {
			m[v.Key] = []string{v.Value}
		}
		err = utils.CheckIfAny(
			func() error { return parser.MapFormByTag(dst.Addr().Interface(), m, "json") },
			func() error { return parser.MapFormByTag(dst.Addr().Interface(), c.Request.URL.Query(), "json") },
		)
	}

	// parse default tag
	err = utils.ParseTag(
		dst.Addr().Interface(),
		utils.ParseTagName("default"),
		utils.ParseTagUnmarshalType(utils.UnmarshalTypeYaml),
	)

	return
}

func (r *router) wrapHandlerFunc(handler routerHandler, reqParse routerRequestParser) gin.HandlerFunc {
	typ := reflect.TypeOf(handler)
	numOut := typ.NumOut()
	return func(c *gin.Context) {
		var (
			err     error
			reqVal  reflect.Value
			rspVals []reflect.Value
		)

		// deal with request & call handler
		if reqParse == nil {
			rspVals = reflect.ValueOf(handler).Call([]reflect.Value{reflect.ValueOf(c)})
		} else if reqVal, err = reqParse(c, typ.In(1)); err == nil {
			rspVals = reflect.ValueOf(handler).Call([]reflect.Value{reflect.ValueOf(c), reqVal})
		} else {
			switch e := err.(type) {
			case *json.UnmarshalTypeError:
				msg := fmt.Sprintf(": %s field type should be %s", e.Value, e.Type.String())
				r.rspError(c, nil, Err(c, r.errorCode, Param(map[string]any{"err": msg})))
			default:
				r.rspError(c, nil, Err(c, r.errorCode,
					Param(map[string]any{"err": fmt.Sprintf(": %s", err.Error())})))
			}
			c.Next()
			return
		}

		// deal with response
		errVal := rspVals[numOut-1]
		if !errVal.IsNil() {
			err = errVal.Interface().(error)
		}
		rspVals = rspVals[:numOut-1]

		var rspType reflect.Type
		isEmbed, isResponse := false, false
		if len(rspVals) > 0 {
			rspType = utils.IndirectType(rspVals[0].Type())
			isEmbed = utils.EmbedsType(rspType, embedType)
			isResponse = utils.EmbedsType(rspType, responseType)
		}

		switch {
		case isEmbed, isResponse:
			r.rspEmbed(c, rspVals[0], rspType, err) // directly json marshal embed response
		case err != nil:
			r.rspError(c, rspVals, err) // business error
		default:
			r.rspSuccess(c, rspVals) // success with response
		}

		c.Next()
	}
}

func (r *router) rspError(c *gin.Context, rspVals []reflect.Value, err error) {
	code, data, page, count, msg := parseRspError(rspVals, err)
	rspError(c, r.appName, code, data, page, count, msg)

	go metricsCode(r.ctx, r.appName, c.Request.URL.Path, c.Request.Method, r.parseHeaderMetrics(c),
		cast.ToInt(code), c.Writer.Status(), c.Writer.Size(), c.Request.ContentLength)
}

func (r *router) rspSuccess(c *gin.Context, rspVals []reflect.Value) {
	data, page, count, msg := parseRspSuccess(rspVals)
	rspSuccess(c, r.successCode, data, page, count, msg)

	go metricsCode(r.ctx, r.appName, c.Request.URL.Path, c.Request.Method, r.parseHeaderMetrics(c),
		r.successCode, c.Writer.Status(), c.Writer.Size(), c.Request.ContentLength)
}

func (r *router) rspEmbed(c *gin.Context, rspVal reflect.Value, rspType reflect.Type, err error) {
	var data any
	if rspVal.IsValid() {
		data = rspVal.Interface()
	} else {
		data = reflect.New(rspType).Interface()
	}

	embedResponse(c, data, err)

	switch rsp := data.(type) {
	case Response:
		go metricsCode(r.ctx, r.appName, c.Request.URL.Path, c.Request.Method, r.parseHeaderMetrics(c),
			rsp.Code, c.Writer.Status(), c.Writer.Size(), c.Request.ContentLength)
	case *Response:
		go metricsCode(r.ctx, r.appName, c.Request.URL.Path, c.Request.Method, r.parseHeaderMetrics(c),
			rsp.Code, c.Writer.Status(), c.Writer.Size(), c.Request.ContentLength)
	default:
		rspData := make(map[string]any)
		dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			Result:  &rspData,
			TagName: "json",
		})
		if err == nil && dec != nil {
			_ = dec.Decode(data)
		}
		var code any
		utils.IfAny(
			func() (ok bool) { code, ok = rspData["code"]; return ok },
			func() (ok bool) { code, ok = rspData["Code"]; return ok },
		)
		if code == nil {
			code = -2
		}
		go metricsCode(r.ctx, r.appName, c.Request.URL.Path, c.Request.Method, r.parseHeaderMetrics(c),
			cast.ToInt(code), c.Writer.Status(), c.Writer.Size(), c.Request.ContentLength)
	}
}

func (r *router) parseHeaderMetrics(c *gin.Context) (headerLabels map[string]string) {
	headerLabels = make(map[string]string, len(r.metricsConf.HeaderLabels))
	for _, metricsHeader := range r.metricsConf.HeaderLabels {
		headerLabels[metricsHeader] = c.Request.Header.Get(metricsHeader)
	}
	return
}

// 0: return error
// 1: return any, error
//    parse as map/struct -> data, msg, count, page
//    parse as others     -> data -> {"code": 0, "data": []} or {"code": 0, "data": {}}
// 2: return data, msg, error
// 3: return data, count, msg, error
// >3: return data, page, count, msg, unknowns..., error
func parseRspSuccess(rspVals []reflect.Value) (data any, page, count int, msg string) {
	switch {
	case len(rspVals) == 0:
	case len(rspVals) == 2:
		data = transformData(rspVals[0])
		msg = reflect.Indirect(rspVals[1]).String()
	case len(rspVals) == 3:
		data = transformData(rspVals[0])
		count = cast.ToInt(reflect.Indirect(rspVals[1]).Int())
		msg = reflect.Indirect(rspVals[2]).String()
	case len(rspVals) > 3:
		data = transformData(rspVals[0])
		page = cast.ToInt(reflect.Indirect(rspVals[1]).Int())
		count = cast.ToInt(reflect.Indirect(rspVals[2]).Int())
		msg = reflect.Indirect(rspVals[3]).String()
	default:
		data, page, count, msg = lookupFieldByStruct(rspVals[0])
	}

	return
}

func parseRspError(rspVals []reflect.Value, err error) (code int, data any, page, count int, msg string) {
	switch e := err.(type) {
	case Errcode:
		code, msg = int(e), e.Error()
	case *bizErr:
		if e.err != nil {
			code, msg = int(e.err.Code), e.Error()
		} else {
			code, msg = int(e.code), e.Error()
		}
	default:
		code, msg = int(errParam), e.Error()
	}

	data, page, count, retMsg := parseRspSuccess(rspVals)
	if retMsg != "" {
		msg = retMsg
	}

	return
}

func lookupFieldByStruct(rspStruct reflect.Value) (data any, page, count int, msg string) {
	var (
		routerResponsePageName = map[string]bool{
			"page": true,
			"Page": true,
		}
		routerResponseCountName = map[string]bool{
			"count": true,
			"Count": true,
		}
		routerResponseDataName = map[string]bool{
			"data": true,
			"Data": true,
		}
		routerResponseMsgName = map[string]bool{
			"msg":     true,
			"Msg":     true,
			"message": true,
			"Message": true,
		}
	)

	type lookupFunc func(v reflect.Value, keywords map[string]bool) (reflect.Value, bool)
	lookupFuncMap := map[reflect.Kind]lookupFunc{
		reflect.Map:    lookupFieldByMap,
		reflect.Struct: lookupFieldByValue,
	}

	rsp := utils.IndirectValue(rspStruct)
	lookup, ok := lookupFuncMap[rsp.Kind()]
	if !ok {
		data = rspStruct.Interface()
		return
	}

	// cannot parse response.Data, resolve all rspStruct as data
	if dataValue, ok := lookup(rsp, routerResponseDataName); ok {
		if dataValue.IsValid() {
			data = transformData(reflect.ValueOf(valueInterface(dataValue, false)))
		}
	} else {
		data = rspStruct.Interface()
		return
	}
	if pageValue, ok := lookup(rsp, routerResponsePageName); ok && pageValue.IsValid() {
		page = cast.ToInt(pageValue.Int())
	}
	if countValue, ok := lookup(rsp, routerResponseCountName); ok && countValue.IsValid() {
		count = cast.ToInt(countValue.Int())
	}
	if msgValue, ok := lookup(rsp, routerResponseMsgName); ok && msgValue.IsValid() {
		msg = msgValue.String()
	}

	return
}

func lookupFieldByValue(v reflect.Value, keywords map[string]bool) (reflect.Value, bool) {
	f := reflect.Indirect(v.FieldByNameFunc(func(s string) bool { return keywords[s] }))
	if !f.IsValid() || f.IsZero() {
		return reflect.Value{}, false
	}
	return reflect.Indirect(f), true
}

func lookupFieldByMap(v reflect.Value, keywords map[string]bool) (reflect.Value, bool) {
	vMap, ok := v.Interface().(map[string]any)
	if !ok {
		return reflect.Value{}, false
	}

	for key := range keywords {
		if vv, ok := vMap[key]; ok {
			return reflect.ValueOf(vv), true
		}
	}
	return reflect.Value{}, false
}

func transformData(data reflect.Value) (transformed any) {
	if !data.IsValid() {
		return
	}

	// some structs return as an interface
	if data.Kind() == reflect.Interface {
		data = reflect.ValueOf(data.Interface())
	}
	if data = utils.IndirectValue(data); !data.IsValid() {
		return
	}

	return data.Interface()
}

type routerOption struct {
	parseFrom      parseFrom
	beforeHandlers []routerHandler
	aftersHandlers []routerHandler
}

func ParseFromBody() utils.OptionFunc[routerOption] {
	return func(r *routerOption) {
		r.parseFrom = parseFromBody
	}
}

func ParseFromQuery() utils.OptionFunc[routerOption] {
	return func(r *routerOption) {
		r.parseFrom = parseFromQuery
	}
}

func HandleBefore(beforeHandlers ...routerHandler) utils.OptionFunc[routerOption] {
	return func(o *routerOption) {
		o.beforeHandlers = beforeHandlers
	}
}

func HandleAfter(aftersHandlers ...routerHandler) utils.OptionFunc[routerOption] {
	return func(o *routerOption) {
		o.aftersHandlers = aftersHandlers
	}
}
