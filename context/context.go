package context

import (
	"context"
	"encoding/gob"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wfusion/gofusion/common/infra/watermill"

	"github.com/wfusion/gofusion/common/infra/watermill/message"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/serialize/json"
)

// _context is a gofusion union context struct
//nolint: revive // struct tag too long issue
type _context struct {
	Langs            []string `json:"langs" yaml:"langs" toml:"langs" mapstructure:"langs"`
	UserID           *string  `json:"user_id" yaml:"user_id" toml:"user_id" mapstructure:"user_id"`
	TraceID          *string  `json:"trace_id" yaml:"trace_id" toml:"trace_id" mapstructure:"trace_id"`
	CronTaskID       *string  `json:"cron_task_id" yaml:"cron_task_id" toml:"cron_task_id" mapstructure:"cron_task_id"`
	CronTaskName     *string  `json:"cron_task_name" yaml:"cron_task_name" toml:"cron_task_name" mapstructure:"cron_task_name"`
	Deadline         *string  `json:"deadline" yaml:"deadline" toml:"deadline" mapstructure:"deadline"`
	DeadlineLocation *string  `json:"deadline_location" yaml:"deadline_location" toml:"deadline_location" mapstructure:"deadline_location"`
}

func (c *_context) unmarshal() (ctx context.Context) {
	ctx = context.Background()
	if c == nil {
		return
	}

	if c.Langs != nil {
		ctx = SetLangs(ctx, c.Langs)
	}
	if c.UserID != nil {
		ctx = SetUserID(ctx, *c.UserID)
	}
	if c.TraceID != nil {
		ctx = SetTraceID(ctx, *c.TraceID)
	}
	if c.CronTaskID != nil {
		ctx = SetCronTaskID(ctx, *c.CronTaskID)
	}
	if c.CronTaskName != nil {
		ctx = SetCronTaskName(ctx, *c.CronTaskName)
	}
	if c.Deadline != nil {
		location := utils.Must(time.LoadLocation(*c.DeadlineLocation))
		// FIXME: it may result context leak issue
		ctx, _ = context.WithDeadline(ctx, utils.Must(time.ParseInLocation(time.RFC3339Nano, *c.Deadline, location)))
	}

	return
}

func (c *_context) Marshal() (b []byte) {
	bs, cb := utils.BytesBufferPool.Get(nil)
	defer cb()
	utils.MustSuccess(gob.NewEncoder(bs).Encode(c))

	b = make([]byte, bs.Len())
	copy(b, bs.Bytes())
	return
}

func Flatten(ctx context.Context) (c *_context) {
	c = new(_context)
	if langs := GetLangs(ctx); langs != nil {
		c.Langs = langs
	}
	if userID := GetUserID(ctx); utils.IsStrNotBlank(userID) {
		c.UserID = utils.AnyPtr(userID)
	}
	if traceID := GetTraceID(ctx); utils.IsStrNotBlank(traceID) {
		c.TraceID = utils.AnyPtr(traceID)
	}
	if taskID := GetCronTaskID(ctx); utils.IsStrNotBlank(taskID) {
		c.CronTaskID = utils.AnyPtr(taskID)
	}
	if taskName := GetCronTaskName(ctx); utils.IsStrNotBlank(taskName) {
		c.CronTaskName = utils.AnyPtr(taskName)
	}
	if deadline, ok := ctx.Deadline(); ok {
		c.Deadline = utils.AnyPtr(deadline.Format(time.RFC3339Nano))
		c.DeadlineLocation = utils.AnyPtr(deadline.Location().String())
	}
	return
}

func WatermillMetadata(ctx context.Context) (metadata message.Metadata) {
	metadata = make(message.Metadata)
	if langs := GetLangs(ctx); langs != nil {
		marshaled, _ := json.Marshal(langs)
		metadata["langs"] = string(marshaled)
	}
	if userID := GetUserID(ctx); utils.IsStrNotBlank(userID) {
		metadata["user_id"] = userID
	}
	if traceID := GetTraceID(ctx); utils.IsStrNotBlank(traceID) {
		metadata["trace_id"] = traceID
	}
	if taskID := GetCronTaskID(ctx); utils.IsStrNotBlank(taskID) {
		metadata["cron_task_id"] = taskID
	}
	if taskName := GetCronTaskName(ctx); utils.IsStrNotBlank(taskName) {
		metadata["cron_task_name"] = taskName
	}
	if deadline, ok := ctx.Deadline(); ok {
		metadata["deadline"] = deadline.Format(time.RFC3339Nano)
		metadata["deadline_location"] = deadline.Location().String()
	}
	return
}

type newOption struct {
	g *gin.Context
	c *_context
	m message.Metadata
}

func (o *newOption) ginUnmarshal() (ctx context.Context) {
	ctx = context.Background()
	if userID := o.g.GetString(KeyUserID); utils.IsStrNotBlank(userID) {
		ctx = SetUserID(ctx, userID)
	}
	if traceID := o.g.GetString(KeyTraceID); utils.IsStrNotBlank(traceID) {
		ctx = SetTraceID(ctx, traceID)
	}
	langs := o.g.Request.Header.Values("Accept-Language")
	if lang := o.g.GetString("lang"); utils.IsStrNotBlank(lang) {
		langs = append(langs, lang)
	}
	if lang := o.g.GetString(KeyLangs); utils.IsStrNotBlank(lang) {
		langs = append(langs, lang)
	}
	if len(langs) > 0 {
		ctx = SetLangs(ctx, langs)
	}
	return
}

func (o *newOption) messageUnmarshal() (ctx context.Context) {
	ctx = context.Background()
	mapGetFn := func(k string) string { return o.m[k] }
	if userID := utils.LookupByFuzzyKeyword[string](mapGetFn, "user_id"); utils.IsStrNotBlank(userID) {
		ctx = SetUserID(ctx, userID)
	}
	if traceID := utils.LookupByFuzzyKeyword[string](mapGetFn, "trace_id"); utils.IsStrNotBlank(traceID) {
		ctx = SetTraceID(ctx, traceID)
	}
	if langstr := utils.LookupByFuzzyKeyword[string](mapGetFn, "langs"); utils.IsStrNotBlank(langstr) {
		var langs []string
		_ = json.Unmarshal([]byte(langstr), &langs)
		ctx = SetLangs(ctx, langs)
	}
	if taskID := utils.LookupByFuzzyKeyword[string](mapGetFn, "cron_task_id"); utils.IsStrNotBlank(taskID) {
		ctx = SetCronTaskID(ctx, taskID)
	}
	if name := utils.LookupByFuzzyKeyword[string](mapGetFn, "cron_task_name"); utils.IsStrNotBlank(name) {
		ctx = SetCronTaskName(ctx, name)
	}
	if messageUUID := o.m[watermill.ContextKeyMessageUUID]; utils.IsStrNotBlank(messageUUID) {
		ctx = utils.SetCtxAny(ctx, watermill.ContextKeyMessageUUID, messageUUID)
	}
	if messageRawID := o.m[watermill.ContextKeyRawMessageID]; utils.IsStrNotBlank(messageRawID) {
		ctx = utils.SetCtxAny(ctx, watermill.ContextKeyRawMessageID, messageRawID)
	}

	deadline := utils.LookupByFuzzyKeyword[string](mapGetFn, "deadline")
	deadlineLoc := utils.LookupByFuzzyKeyword[string](mapGetFn, "deadline_location")
	if utils.IsStrNotBlank(deadline) {
		location := utils.Must(time.LoadLocation(deadlineLoc))
		// FIXME: it may result context leak issue
		ctx, _ = context.WithDeadline(ctx, utils.Must(time.ParseInLocation(time.RFC3339Nano, deadline, location)))
	}
	return
}

func Context(c []byte) utils.OptionFunc[newOption] {
	return func(o *newOption) {
		o.c = new(_context)
		utils.MustSuccess(utils.Unmarshal(c, o.c, ""))
	}
}

func Gin(c *gin.Context) utils.OptionFunc[newOption] {
	return func(o *newOption) {
		o.g = c
	}
}

func Watermill(m message.Metadata) utils.OptionFunc[newOption] {
	return func(o *newOption) {
		o.m = m
	}
}

func New(opts ...utils.OptionExtender) (ctx context.Context) {
	o := utils.ApplyOptions[newOption](opts...)

	// alternative
	switch {
	case o.g != nil:
		return o.ginUnmarshal()
	case o.c != nil:
		return o.c.unmarshal()
	case o.m != nil:
		return o.messageUnmarshal()
	default:
		panic(ErrUnknownInstantiationMethod)
	}

	return
}
