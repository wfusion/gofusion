- English README version: [README.md](../../../README.md)
- 中文 README: 您当前正在阅读此文档

# 框架使用限制：

- golang: 1.18 - 1.22
- os: windows / darwin / linux
- arch: amd64 / arm64 / loong64

# 框架简介

- 框架特性: 高可配置化, 高可拓展性, 高依赖可替换性, 多组件可组合, 深度结合依赖注入, 助力业务高效率建设
- 框架支持组件: db, http, i18n, lock, cache, log, mongo, redis, mq, routine, cron, async
- 框架特色功能:
    - 支持 yaml, json, toml 格式配置文件, 组件参数高可配置化, 可在运行时修改各组件日志开关
    - 多种 db 类型支持: mysql, postgres, opengauss, sqlite, sqlserver, tidb, clickhouse, 依赖替换基本无业务感知
    - 多种 mq 类型支持: rabbitmq, kafka, pulsar, mysql, postgres, redis, gochannel, 依赖替换无业务感知
    - 各组件均自动注册依赖注入, 助力业务构建依赖注入的系统架构, 基于 uber/dig
    - 分布式定时任务, 基于 asynq
    - 分布式异步任务, 基于 asynq
    - http 服务, 定时任务, 异步任务, 消息队列均支持 router 模式, 高自由度注册函数签名
    - 支持近端和远端缓存
    - 各组件支持多实例
- 框架对比
    - 相比 go-zero, go-micro, kratos, jupiter, kitex, dubbo-go, tarsgo 等微服务框架, 本框架非微服务框架, 是为依赖之整合与组合,
      可以和其他框架组合使用

# 快速开始

> Gofusion quick start

- 拷贝 [test/config/configs/app_zh.yml](../../../test/config/configs/app_zh.yml) 到业务仓库 configs 目录中并重命名为 app.yml, 或者其他位置在启动时通过 --config-file 参数指定
- 通过如下代码初始化 gofusion

```go
package main

import "github.com/wfusion/gofusion/config"

func main() {
	appSetting := &struct{}{} // 业务配置对象
	defer config.Registry.Init(&appSetting)()
}
```

> fus quick start

- fus 目前支持压缩加密编码, 随机数生成, gorm-gen, asynq 官方开源命令行, watermill 官方开源命令行

```bash
~ > go install github.com/wfusion/gofusion/config/common/fus@master
~ > fus -h
Gofusion CLI (v0.0.1 built with go1.18.10 darwin/arm64 from v1.1.4 on Sun Nov 12 18:05:03 CST 2023)

Capability:
  asynq client integrated
  watermill client with pubsub kafka, ampq, and io enabled integerate
  gorm gentool integerate
  encoder&decoder with cipher, compress, and print encoding
  random bytes generater

Usage:
  fus [command]

Available Commands:
  asynq       A CLI for asynq
  completion  Generate the autocompletion script for the specified shell
  dec         Decode data from stdin or filename
  enc         Encode data from stdin or filename
  gorm        A CLI for gorm gen-tool
  help        Help about any command
  mill        A CLI for watermill
  rnd         Generate cryptographically secure random bytes

Flags:
      --debug     print debug info
  -h, --help      help for fus
  -v, --version   version for fus

Use "fus [command] --help" for more information about a command.
```

# 单元测试

- 基于 allure 的单测用例报告, [test result](../../unittest/unittest.html)
- 基于 github.com/nikolaydubina/go-cover-treemap 生成的单测覆盖率报告, 当前覆盖率为 60%

![go-cover-treemap](../../unittest/coverage.svg)

# 功能简述

> 简述 gofusion 能力

## config

> 配置组件, 主要提供 gofusion 初始化、配置解析以及各组件优雅退出功能

- 业务配置类无需再声明依赖的配置，配置全在 yml 配置文件的 base 节点下
- 组件根据业务依赖自动注册和按顺序优雅退出的能力
- 支持根据环境变量指定配置文件
- 支持 yaml, json, toml 格式的配置文件 (示例见 [app.yml](../../../test/config/configs/app_zh.yml))
- 支持命令行指定配置参数  (示例见 [args.example](../../../test/config/configs/full_commandline_args.example))
- 多项配置覆盖优先级: 命令行参数 > --config-file > app.local.yml > app."${ENV}".yml > app.yml
- 支持全局 debug，对应会对 gorm, gin 开启 debug 模式
- 配置支持定义默认值（`default: "{yaml 格式的内容}"`）
- 支持 --config-file 指定多个配置文件的方式
- 支持 auto reload 的配置内容的自动重载 (mod time + sha256)
- 自动调用 flag.Parse 所以无需业务自行调用
- 支持业务对象实现 BeforeLoad 和 AfterLoad 在配置加载前后进行回调
- 业务可自定义初始化 context, 业务可通过控制该 context 管理各个组件的内部实现的生命周期和 metrics 打点循环
- 支持对 db, mongo, redis 的 password 通过 fus 加密, 从配置文件中以密文加载对应配置

## db

> 数据库组件, 提供关系型数据库封装功能, 部分 db 类型还在开发自测中...

- 支持多 DB 配置
- 基于 gorm v2 支持 mysql, postgres, opengauss, sqlite, sqlserver, tidb(未测试), clickhouse(未测试)
- 支持字符串指定 gorm 日志对象，默认 gorm 日志可在运行时动态调整是否开启、日志等级、高耗时阈值
- 支持 nullable timestamp 和 bool 软删除字段，支持同一个 model 多软删除字段配置, 兼容 gorm.DeletedAt, 兼容分表插件
- 支持和 lock 组件结合提供分布式锁能力
- 封装 dal 提供 DalInterface 接口，可使用 db.NewDAL 直接使用，提供基本的 query, queryFirst, queryLast, count, pluck,
  take, insertOne, insertInBatch, save, update, updates, delete, firstOrCreate. 为兼容其他使用方式提供了 ReadDB, WriteDB
  方法返回 native 的 -gorm.DB, 提供 IgnoreErr 和 CanIgnore 封装让业务规避 gorm 在使用 first, last, pluck, task 中无数据返回
  error 的问题
- 封装 db.Scan 函数支持全表扫描
- 封装 db.WithinTx 函数支持事务
- 封装 model 定义公共类
- 性能和延迟打点，目前暂无平台可上报，暂时不可用
- gorm 自增 id 在关联实体中的问题，目前可根据不同数据库做配置，可在配置文件中通过
  auto_increment_increment 声明，如果是 mariaDB 或 MySQL 默认会根据 sql show variables like 'auto_increment_increment'
  查询数据库自增步长。gorm 问题详见 [Issue #5814 · go-gorm/gorm · GitHub](https://github.com/go-gorm/gorm/issues/5814)

> DB 分表插件特性如下

- mysql, postgres, sqlserver 均通过测试
- 支持一个数据库多表分表，一个表多列 sharding
- 支持裸 SQL
- 支持 AutoMigrate, DropTable
- 多列可使用表达式聚合，默认大数位移按照二进制拼一起
- 可以根据 id 列分表，默认使用雪花算法，机器码为 hash(宿主机 ip + 本地 ip + pid) << 8 | 本地 ipv4 最后一段
- 支持自定义后缀名，默认就是 原表名_列名1_列名2_0, 尾缀是 1, 还是 01, 001 由分表数量决定
- 创建时支持默认建表，即发现对应分表未创建则自动建表，方便做数据迁移
- 结合 DalInterface 接口，其批量插入，Save，Delete 均支持自动分表，即传入不同表的实体也可以处理

## http

> http 组件, 提供 http 框架和错误码的封装功能

- 支持 gin.HandlerFunc 以外的其他函数签名，自动根据 http 请求 content-type 从 param, query, body 中解析入参，根据返回
  error 和数据解析返回数据
- 支持多种中间件: cors 跨域，原逻辑; logging 记录请求的脱敏日志; xss 防御; trace id 透传，原有逻辑增强; recover 异常捕获，原有逻辑;
  可支持基于字符串的中间件自定义，暂未开放此功能
- 错误码文案支持 i18n
- 错误码文案支持带参数（基于 golang 官方 text/template）
- 支持自定义 response，直接使用 http.Response 或者继承 http.Embed 即可使用自定义，主要是为了兼容业务的各种使用
- 支持基于 gin 框架的零拷贝的 gin.HandlerFunc, 支持静态文件名, 支持 io.ReadSeeker 数据流

## i18n

> 国际化组件, 提供国际化文案功能

- 支持多语言
- 支持定义变量
- 支持重复 key 检测，即错误码定义重复会启动失败

## context

> 上下文组件, 提供基于 context 的字段透传功能

- 基于 http 组件支持从 gin 初始化，兼容过去的使用方法
- user id, trace id 透传
- async 组件中, 支持 user id, trace id, context.deadline 透传
- db 组件中, 支持 gorm 事务透传
- log 组件中, 支持自动解析打印 context 携带的 user id, trace id 信息

## lock

> 分布式锁组件, 提供分布式锁功能

- 支持基于 redis lua 的分布式锁, 可重入
- 支持基于 redis setnx 用 timeout 管理的分布式锁
- 支持基于 mysql/mariadb GET_LOCK/RELEASE_LOCK 的分布式锁
- 支持基于 mongo collection 和唯一键的分布式锁, 可重入
- 封装 lock.Within 分布式锁调用

## cache

> 缓存组件, 提供数据缓存功能

- 支持近端和远端缓存，近端采用 github.com/bluele/gcache, 远端目前仅支持 redis
- 近端缓存支持 arc, lfu, lru, simple 逐出策略, 默认使用 arc
- 支持 json, gob, msgpack, ctor 对原始对象进行编码
- 支持序列化后使用 zstd, zlib, s2, gzip, deflate 压缩缓存数据
- 支持粒度到 key 的过期时间设置
- 不同编码和压缩算法读取兼容，即修改编码和压缩算法不影响历史数据的读取和解析

## log

> 日志组件, 提供日志打印和输出功能

- 基于 go.uber.org/zap, 兼容过去的使用
- 支持 log.Fields 作为参数直接带入日志打印
- 支持 console 和 file 的输出，配置方式同以前类似，主要是和以前保持兼容, console 彩色输出支持配置
- 基于 gopkg.in/natefinch/lumberjack.v2 支持对输出的文件进行自动归档, 可定义归档日志的最大大小, 切割周期, 最大数量, 是否
  gzip 压缩
- 支持打印调用日志记录的真实文件位置（非跳过多少个固定深度栈的解决方案，因为 gorm，redis，mongo 等自定义日志的打印就不是固定的栈深度）
- 封装 TimeElapsed 记录执行耗时

## mongo

> mongo 组件, 提供 mongo 功能

- 暂未封装为接口，暂未提供类似 db 一样的 dal 封装
- 支持自定义日志打印，支持过滤出可打印的 mongo 指令
- 支持多 mongo 配置

## redis

> redis 组件, 提供 redis 功能

- 支持自定义日志打印, 支持过滤出不可打印的 redis 指令
- 支持多 redis 配置
- 支持和 lock 组件结合, 提供分布式锁能力
- 支持和 cache 组件组合, 提供缓存能力
- 支持和 cron 组件组合, 提供定时任务能力
- 支持和 async 组件组合, 提供异步任务能力

## message queue

> mq 组件, 提供消息队列功能

- 基于 github.com/ThreeDotsLabs/watermill, 因原始项目要求 go1.21 故 fork 到本仓库进行修改, 并针对各 pubsub 开源实现进行调整
- 支持 amqp, rabbitmq, gochannel, kafka, pulsar, redis, mysql, postgres, 全类型在配置有消费者组的情况下已通过所有单测
- 框架支持 pub/sub 和 pub/router 两种模式, 且两种模式可同时使用
    - 若是同一个配置同时使用两种模式, 使用 raw 和 default 消息时 router 和 sub 会争抢消费
    - 若是同一个配置同时使用两种模式, 使用 event 消息时 router 和 sub 会重复消费
    - 若是同一个配置注册多次 router, 使用 raw 和 default 消息时 router 和 sub 会争抢消费
    - 若是同一个配置注册多次 router, 使用 event 时会触发 panic, 因为注册名必须是 event type 名称会导致重名
- 消息支持 default, raw, event 两种发布订阅消息, raw 为原始消息不做框架封装, default 和 event 为框架编码后的消息
    - 发布订阅 default 消息, 即
        - publisher 使用 Messages(), Objects() 选项调用 Publish
        - subscriber 使用 Subscribe
        - router 使用 HandlerFunc, watermill.NoPublishHandlerFunc, watermill.HandlerFunc, 自定义 func(ctx
          context.Context, payload *SerializableObj) error 函数注册时
    - 发布订阅 raw 消息, 即
        - publisher 使用 Messages() 选项调用 PublishRaw
        - subscriber 使用 SubscribeRaw
        - router 使用 HandlerFunc, watermill.NoPublishHandlerFunc, watermill.HandlerFunc, 自定义 func(ctx
          context.Context, payload *SerializableObj) error 函数注册时
    - 发布订阅 event 消息, 即
        - publisher 使用 NewEventPublisher, NewEventPublisherDI 初始化, 使用 Events() 选项调用 PublishEvent
        - subscriber 使用 NewEventSubscriber, NewEventSubscriberDI 初始化, 调用 SubscribeEvent
        - router 使用 EventHandler 或 EventHandlerWithMsg 闭包注册时
- 消费者执行函数可自定义为 func(ctx context.Context, payload *SerializableObj) error
- 生产者可直接传递消息结构体, 框架根据配置自动完成序列化和压缩, 仅对 default 和 event 消息有效
- 支持 json, gob, msgpack, ctor 对原始对象进行序列化, 仅对 default 和 event 消息有效
- 支持序列化后使用 zstd, zlib, s2, gzip, deflate 压缩消息, 仅对 default 和 event 消息有效
- 支持 langs, user_id, trace_id, context.deadline 上下游透传
- 支持打印消息的业务 uuid 和消息的原生 id, 上下游不透传
- 支持定义自定义 Event 进行事件的生产和消费, 也可只用于在单个 Topic 中区分不同消息做业务分发, TODO: 消息去重和过时消息丢弃待开发

## routine

> 协程组件, 提供协程管理和协程池功能

- 提供全局协程数量管理，可定义业务服务中的最大协程数和优雅退出
- 支持 debug 模式, 对于 routine.Go, routine.Promise 以及 Pool.Submit 的调用改为同步调用, 方便进行测试和单测编写
- 封装 routine.Go, routine.Goc 提供 go func 调用，可优雅退出; Args 选项支持函数任意入参, WaitGroup 选项支持 Done 调用,
  Channel 选项支持接收函数返回
- routine.Loop, routine.Loopc 兼容部分无需优雅退出的业务场景
- 基于 github.com/fanliao/go-promise 封装 routine.Promise, routine.WhenAll, routine.WhenAny 便于调用和等待多协程, 支持任意入参
- 基于 github.com/panjf2000/ants/v2 封装 Pool 接口提供协程池的使用, 支持任意入参

## cron

> 分布式定时任务组件, 提供分布式定时任务调度和管理功能

- 目前仅支持基于 asynq 的 redis 的模式
- 单个服务实例支持多 cron 实例，分布式环境下支持多 trigger 和多 worker，即同时开启时服务呈现无状态服务特性
- 支持引入 lock 组件解决多 trigger 任务重复分发的问题（不使用 lock 组件即使用原生 unique 选项规避问题）
- 定时任务执行函数可自定义为 func(ctx context.Context, payload *JsonSerializable) error 签名, payload 会在框架中自动解析
- 定时任务执行前会给 context 中携带 cron_task_name 和 trace_id 信息
- 配置文件中支持 asynq 框架中 timeout, payload(json 格式), retry, deadline 特性配置
- 除可以在配置文件中配置定时任务外，业务可自定义 task loader 追加定时任务

## async

> 分布式异步任务组件, 提供分布式异步任务调度和管理功能

- 目前仅支持基于 asynq 的 redis 的模式
- 支持 langs, user_id, trace_id, context.deadline 上下游透传
- 单个服务实例支持多个 async 实例，分布式环境下支持多 producer 和多 consumer，即同时开启时服务呈现无状态服务特性
- 异步执行函数可自定义函数签名, 满足 func(ctx context.Context, **选择的序列化算法支持的一个或多个结构体**) error 即可,
  其他参数会在框架中自动解析

## metrics

> 监控埋点(打点)

- 目前仅支持 prometheus, 基于 github.com/hashicorp/go-metrics 开发, 支持 prometheus 的 histogram 埋点
- 支持 pull 和 push 模式
- 支持自定义 label 常量
- 支持配置 golang 程序的 runtime 埋点, 各个组件的埋点以及运行程序 hostname, ip 的 label 追加
- 业务调用埋点时为传入 golang channel, 可配置 channel 的大小和并发处理效率避免影响业务性能或耗时
- 业务可配置埋点时, 若埋点任务 channel 满时的策略, 默认为丢弃埋点, 可配置 timeout 或 without timeout 可选项

## common

> 通用工具, 提供常用函数和封装

| 路径              | 能力                                                                                                                                         |
|:----------------|:-------------------------------------------------------------------------------------------------------------------------------------------|
| constant        | 常量, 包含: 反射类型, 常用符号, 时间 format                                                                                                              |
| constraint      | 泛型约束                                                                                                                                       |
| di              | 基于 uber dig 的依赖注入                                                                                                                          |
| env             | 获取运行环境信息                                                                                                                                   |
| fus             | Gofusion CLI, 使用详见二进制, 安装: go install github.com/wfusion/gofusion/config/common/fus@master                                                 |
| infra           | 基础设施, 目前包含: mongo 驱动, 常见关系型数据库驱动, redis 驱动, asynq 异步和定时任务框架, watermill 事件驱动框架                                                              |
| utils/cipher    | 加解密, 支持 des, 3des, aes, sm4, rc4, chacha20poly1305, xchacha20poly1305 加密算法, ebc, cbc, cfb, ctr, ofb, gcm 加密模式, 支持流式处理                      |
| utils/clone     | 支持任意类型的深拷贝, 基于 github.com/huandu/go-clone                                                                                                  |
| utils/cmp       | 支持常见类型的比较                                                                                                                                  |
| utils/compress  | 压缩, 支持 zstd, zlib, s2, gzip, deflate 压缩算法, 支持流式处理, 基于 github.com/klauspost/compress                                                        |
| utils/serialize | 序列化, 支持 gob, json, msgpack, cbor 序列化算法                                                                                                     |
| utils/sqlparser | sql 语句解析, 基于 github.com/longbridgeapp/sqlparser 并支持了 OFFSET FETCH 语句                                                                       |
| utils/encode    | 可打印编码, 支持 hex, base32, base32-hex, base64, base64-url, base64-raw, base64-raw-url 编码算法, 支持流式处理, 支持加解密, 压缩以及可打印编码任意排列组合提供 []byte 输入处理和流处理封装 |
| utils/gomonkey  | monkey patch, fork from github.com/agiledragon/gomonkey                                                                                    |
| utils/inspect   | 支持私有字段读写, 支持根据字符串获取函数和 reflect.Type, inspired by github.com/chenzhuoyu/go-inspect                                                          |
| utils/pool.go   | 池化泛型封装, 默认提供了 bytes.Buffer 和 []byte 的池化对象                                                                                                  |
| utils/xxx.go    | 其他常用函数或封装, 涵盖 compare, context, conv, enum, func, heap, ip, json, map, number, options, random, reflect, sets, slice, sort, string, time   |

# Thanks for

- [hibiken/asynq](https://github.com/hibiken/asynq)
- [ThreeDotsLabs/watermill](https://github.com/ThreeDotsLabs/watermill)
- [hashicorp/go-metrics](https://github.com/hashicorp/go-metrics)
- [huandu/go-clone](https://github.com/huandu/go-clone)
- [chenzhuoyu/go-inspect](https://github.com/chenzhuoyu/go-inspect)
- [agiledragon/gomonkey](https://github.com/agiledragon/gomonkey)
- [fvbock/endless](https://github.com/fvbock/endless)
- [longbridgeapp/sqlparser](https://github.com/longbridgeapp/sqlparser)
- [jinzhu/configor](https://github.com/jinzhu/configor)
- [go-gorm/postgres](https://github.com/go-gorm/postgres)
- [natefinch/lumberjack](https://github.com/natefinch/lumberjack)

# Todo List

- [ ] metrics each component
- [ ] mq component support event deduplication and outdated message discarding
- [ ] config component support Apollo dynamic config with viper lib
- [ ] http client wrapper for passing through trace id
- [ ] support rpc component
- [ ] support watermill type in cron and async components