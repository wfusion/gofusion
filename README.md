- English README version: you are reading now =。= translated by openai chat gpt4
- Chinese README version: https://github.com/wfusion/gofusion/blob/master/assets/docs/zh/README.md

# Usage Limitations

- golang: 1.18 - 1.21
- os: windows / darwin / linux
- arch: amd64 / arm64 / loong64

# Introduction

- Features: Highly configurable, highly extendable, high dependency replace ability, multi-component combinable, deeply
  integrated with dependency injection, facilitating efficient business development.
- Supported Components: db, http, i18n, lock, cache, log, mongo, redis, mq, routine, cron, async, metrics
- Special Features:
    - Supports YAML, JSON, TOML configuration file formats, highly configurable component parameters, with the ability
      to modify various component log switches at runtime.
    - Multiple db types supported: MySQL, Postgres, OpenGauss, SQLite, SQLServer, TiDB, ClickHouse, with nearly seamless
      dependency replacement.
    - Multiple mq types supported: RabbitMQ, Kafka, Pulsar, MySQL, Postgres, Redis, GoChannel, with seamless dependency
      replacement.
    - All components automatically register for dependency injection, aiding in building a system architecture based on
      dependency injection, utilizing uber/dig.
    - Distributed scheduled tasks, based on asynq.
    - Distributed asynchronous tasks, based on asynq.
    - HTTP services, scheduled tasks, asynchronous tasks, message queues all support router mode, with high freedom in
      registering function signatures.
    - Supports near and remote caching.
    - Multi-instance support for all components.
- Comparison
    - Compared to microservice frameworks like go-zero, go-micro, kratos, jupiter, kitex, dubbo-go, tarsgo, this
      framework is not a microservices framework, but is designed for dependency integration and combination, and can be
      used in conjunction with other frameworks.

# Quick Start

> Gofusion quick start

- Copy `test/config/configs/app.yml` to the `configs` directory in your business repository, or specify another location
  at startup with the `-configPath` parameter.
- Initialize gofusion with the following code:

```go
package main

import "github.com/wfusion/gofusion/config"

func main() {
	appSetting := &struct{}{} // Business configuration object
	defer config.Registry.Init(&appSetting)()
}
```

> fus quick start

- fus currently supports compression encryption encoding, random number generation, gorm-gen, asynq official open-source
  command line, watermill official open-source command line.

```bash
~ > go install github.com/wfusion/gofusion/common/fus@master
~ > fus -h
Gofusion CLI (v0.0.4 built with go1.18.10 darwin/arm64 from v1.0.0 on Sat Nov  4 01:00:32 CST 2023)

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

# Unit Testing

> Unit test coverage report generated based on github.com/nikolaydubina/go-cover-treemap, current coverage is 57%

![go-cover-treemap](assets/coverage.svg)

# Feature Summary

> Briefly describing gofusion capabilities

## config

> Configuration Component, primarily provides gofusion initialization, configuration parsing, and graceful exit
> functionality for various components

- Business configuration class no longer needs to declare dependent configurations; all configurations are under
  the `base` node in the yml file.
- Components are automatically registered and gracefully exited in order based on business dependencies.
- Supports specifying configuration files based on environmental variables.
- Supports yaml, json, toml format configuration files.
- Supports multiple configuration file priorities -configPath > app.local.yml > app.$env.yml > app.yml.
- Supports global debug, correspondingly enables debug mode for gorm, gin.
- Configurations support defining default values (yaml format content).
- Supports specifying configuration files using -configPath.
- Supports auto reload of configuration content (mod time + sha256).
- Automatically calls flag.Parse, so no need for business to call it manually.
- Supports business objects implementing BeforeLoad and AfterLoad for callbacks before and after configuration loading.
- Business can customize initialization context, and manage the lifecycle and metrics looping of various components'
  internal implementations through this context.
- Supports encrypting passwords for db, mongo, redis through fus, and loading corresponding configurations from the
  configuration file in ciphertext.

## db

> Database Component, provides relational database encapsulation features, some types are still under development
> testing...

- Supports multiple DB configurations.
- Based on gorm v2 supports mysql, postgres, opengauss, sqlite, sqlserver, tidb (untested), clickhouse (untested).
- Supports string-specified gorm log objects, default gorm log can dynamically adjust at runtime, log level, high
  consumption threshold.
- Supports nullable timestamp and bool soft delete fields, supports multiple soft delete field configurations in the
  same model, compatible with gorm.DeletedAt, compatible with sharding plugin.
- Combined with lock component to provide distributed lock capability.
- Encapsulated dal provides DalInterface interface, can use db.NewDAL directly, provides basic query, queryFirst,
  queryLast, count, pluck,
  take, insertOne, insertInBatch, save, update, updates, delete, firstOrCreate. For compatibility with other usages,
  provides ReadDB, WriteDB
  methods to return native -gorm.DB, provides IgnoreErr and CanIgnore encapsulation to help business avoid gorm no data
  return error when using first, last, pluck, task.
- Encapsulated db.Scan function supports full table scan.
- Encapsulated db.WithinTx function supports transactions.
- Encapsulated common class definition for model.
- Performance and latency dotting, currently no platform for reporting, temporarily unavailable.
- Gorm auto increment id issue in associated entities, currently can be configured according to different databases,
  declared in the configuration file through
  auto_increment_increment, if mariaDB or MySQL will query database auto-increment step by default according to sql show
  variables like 'auto_increment_increment'. Gorm issue see [Issue #5814](https://github.com/go-gorm/gorm/issues/5814)

> DB Sharding Plugin Features

mysql, postgres, sqlserver all passed tests.

- Supports multi-table sharding in one database, multi-column sharding in one table.
- Supports raw SQL.
- Supports AutoMigrate, DropTable.
- Multi-columns can use expression aggregation, default large digit shift according to binary stitching.
- Can shard table according to id column, default using snowflake algorithm, machine code is hash(host ip + local ip +
  pid) << 8 | last segment of local ipv4.
- Supports custom suffix name, default is original_table_name_column1_column2_0, suffix is 1, or 01, 001 depending on
  the number of sharded tables.
- Supports default table creation when creating, if the corresponding sharded table is not created, it will be created
  automatically, convenient for data migration.
- Combined with DalInterface interface, its batch insert, Save, Delete all support automatic sharding, that is, entities
  of different tables can be processed.

## HTTP

> HTTP component, provides HTTP component and error code encapsulation features

- Supports function signatures other than gin.HandlerFunc, automatically parses parameters from param, query, body based
  on HTTP request content-type, and analyzes returned data based on returned error.
- Supports various middlewares: cors cross-domain, original logic; logging for recording desensitized request logs; xss
  defense; trace id propagation, enhanced original logic; recover for exception capture, original logic; string-based
  middleware customization supported, yet this feature is temporarily unavailable.
- Supports i18n error codes.
- Error message texts support i18n.
- Error message texts support parameters (based on official golang text/template).
- Supports custom responses, use http.Response directly or inherit http.Embed for customization, mainly for
  compatibility with various business usages.
- Supports zero-copy gin.HandlerFunc based on gin framework, supports static filenames, supports io.ReadSeeker data
  streams.

## I18n

> Internationalization component, provides internationalization text features

- Supports multiple languages.
- Supports defining variables.
- Supports duplicate key detection, i.e., startup will fail if error code definitions are duplicated.

## context

> Context component, provides field passing functionality based on context

- Supports initialization from gin based on HTTP component, compatible with past usages.
- user id, trace id propagation.
- In async component, supports user id, trace id, context.deadline propagation.
- In db component, supports gorm transaction propagation.
- In log component, supports automatic parsing and printing of user id, trace id information carried by context.

## lock

> Distributed Lock component, provides distributed lock functionality

- Supports distributed locks based on redis lua.
- Supports distributed locks managed with timeout based on redis setnx.
- Supports distributed locks based on mysql/mariadb GET_LOCK/RELEASE_LOCK.
- Encapsulated lock.Within for distributed lock invocation.

## Cache

> Cache component, provides data caching functionality

- Supports near and remote caching, near-end uses github.com/bluele/gcache, remote currently only supports redis.
- Near-end cache supports arc, lfu, lru, simple eviction policies, arc used by default.
- Supports json, gob, msgpack, ctor for raw object encoding.
- Supports zstd, zlib, s2, gzip, deflate for compressed cache data post-serialization.
- Supports expiration time settings at the granularity of keys.
- Different encoding and compression algorithm read compatibility, i.e., modifying encoding and compression algorithms
  does not affect the reading and parsing of historical data.

## Log

> Log component, provides log printing and output functionality

- Based on go.uber.org/zap, compatible with past usage.
- Supports log.Fields as parameters directly brought into log printing.
- Supports console and file output, configuration similar to before, mainly for compatibility with the past, console
  color output configurable.
- Based on gopkg.in/natefinch/lumberjack.v2, supports automatic archiving of output files, defines maximum size, split
  cycle, maximum number, gzip compression for archived logs.
- Supports printing real file location of log call records (not a fixed stack depth skipping solution, as gorm, redis,
  mongo custom log printing is not at a fixed stack depth).
- Encapsulates TimeElapsed to record execution duration.

## Mongo

> Mongo component, provides mongo functionality

- Not yet encapsulated as an interface, nor provided dal encapsulation similar to db.
- Supports custom log printing, filters out printable mongo commands.
- Supports multiple mongo configurations.

## Redis

> Redis component, provides redis functionality

- Supports custom log printing, filters out non-printable redis commands.
- Supports multiple redis configurations.
- Supports combination with lock component, provides distributed lock capability.
- Supports combination with cache component, provides caching capability.
- Supports combination with cron component, provides scheduled task capability.
- Supports combination with async component, provides asynchronous task capability.

## Message Queue

> MQ component, provides message queue functionality

- Based on github.com/ThreeDotsLabs/watermill, forked and modified for compatibility with Go1.21, adjustments made for
  each pubsub open-source implementation.
- Supports amqp, rabbitmq, gochannel, kafka, pulsar, redis, mysql, postgres, all types passed all unit tests with
  consumer groups configured.
- Supports both pub/sub and pub/router modes, both modes can be used simultaneously.
    - When using both modes with the same configuration, router and sub will compete for consumption with raw and
      default messages.
    - When using both modes with the same configuration, router and sub will consume event messages repeatedly.
    - When registering multiple routers with the same configuration, router and sub will compete for consumption with
      raw and default messages.
    - When registering multiple routers with the same configuration, using event will trigger panic as registration name
      must be event type name causing duplication.
- Messages support default, raw, event types for publish/subscribe, raw is original message without encapsulation,
  default and event are framework-encoded messages.
    - Publish/subscribe default messages
        - Publisher uses Messages(), Objects() options to call Publish.
        - Subscriber uses Subscribe.
        - Router uses HandlerFunc, watermill.NoPublishHandlerFunc, watermill.HandlerFunc, custom func(ctx
          context.Context, payload *SerializableObj) error function for registration.
    - Publish/subscribe raw messages
        - Publisher uses Messages() option to call PublishRaw.
        - Subscriber uses SubscribeRaw.
        - Router uses HandlerFunc, watermill.NoPublishHandlerFunc, watermill.HandlerFunc, custom func(ctx
          context.Context, payload *SerializableObj) error function for registration.
    - Publish/subscribe event messages
        - Publisher initializes with NewEventPublisher, NewEventPublisherDI, uses Events() option to call PublishEvent.
        - Subscriber initializes with NewEventSubscriber, NewEventSubscriberDI, calls SubscribeEvent.
        - Router registers with EventHandler or EventHandlerWithMsg closure.
- Consumer execution function can be customized to func(ctx context.Context, payload *SerializableObj) error.
- Producers can directly pass message structures, auto-completes serialization and compression based on configuration,
  only valid for default and event messages.
- Supports json, gob, msgpack, ctor for raw object serialization, only valid for default and event messages.
- Supports post-serialization compression using zstd, zlib, s2, gzip, deflate, only valid for default and event
  messages.
- Supports langs, user_id, trace_id, context.deadline downstream propagation.
- Supports printing of business uuid and native id of messages, not propagated downstream.
- Supports defining custom Event for production and consumption of events, or for distinguishing different messages
  within a single Topic for business distribution, TODO: message deduplication and outdated message discarding to be
  developed.

## Routine

> Coroutine component, provides coroutine management and coroutine pool functionality

- Provides global coroutine count management, defines maximum coroutine count and graceful exit for business services.
- Supports debug mode, changes routine.Go, routine.Promise, and Pool.Submit calls to synchronous calls for easier
  testing and unit test writing.
- Encapsulates routine.Go, routine.Goc for go func calls, allows graceful exit; Args option supports arbitrary function
  parameters, WaitGroup option supports Done call, Channel option supports receiving function returns.
- routine.Loop, routine.Loopc compatible for certain business scenarios that don't require graceful exit.
- Based on github.com/fanliao/go-promise, encapsulates routine.Promise, routine.WhenAll, routine.WhenAny for easy
  multi-coroutine calling and waiting, supports arbitrary parameters.
- Based on github.com/panjf2000/ants/v2, encapsulates Pool interface for coroutine pool usage, supports arbitrary
  parameters.

## Cron

> Distributed scheduled task component, provides distributed scheduled task scheduling and management functionality

- Currently only supports asynq-based redis mode.
- Single service instance supports multiple cron instances, distributed environment supports multiple triggers and
  workers, exhibiting stateless service characteristics when both are enabled.
- Supports lock component integration to solve the issue of multiple trigger task re-distribution (without using lock
  component, use native unique option to avoid issue).
- Scheduled task execution function can be customized to func(ctx context.Context, payload *JsonSerializable) error
  signature, payload will be automatically parsed in gofusion.
- Scheduled task execution will carry cron_task_name and trace_id information in context.
- Configuration file supports asynq framework's timeout, payload (json format), retry, deadline feature configurations.
- Besides configuring scheduled tasks in the configuration file, businesses can define custom task loader to append
  scheduled tasks.

## Async

> Distributed asynchronous task component, provides distributed asynchronous task scheduling and management
> functionality

- Currently only supports asynq-based redis mode.
- Supports langs, user_id, trace_id, context.deadline downstream propagation.
- Single service instance supports multiple async instances, distributed environment supports multiple producers and
  consumers, exhibiting stateless service characteristics when both are enabled.
- Asynchronous execution function can have a custom function signature, satisfying func(ctx context.Context, **selected
  serialization algorithm-supported one or more structures**) error, other parameters will be automatically parsed in
  gofusion.

## Metrics

> Monitoring pinpoint

- Currently only supports Prometheus, developed based on github.com/hashicorp/go-metrics, supports Prometheus's
  histogram pinpoint.
- Supports pull and push modes.
- Supports custom label constants.
- Supports configuration of golang program's runtime pinpoint, each component's pinpoint as well as running program
  hostname, ip label appending.
- Business pinpoint call passes golang channel, channel size and concurrency processing efficiency can be configured to
  avoid affecting business performance or duration.
- Businesses can configure pinpoint, if pinpoint task channel is full, strategy is default to discard pinpoint,
  configurable timeout or without timeout options are available.

## Common

> Common tools, providing frequently used functions and wrappers

| Path            | Capability                                                                                                                                                                                                                                                                       |
|:----------------|:---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| configor        | Mainly used by gofusion itself, loads configuration files and fills in default values                                                                                                                                                                                            |
| constant        | Constants, includes: reflection types, common symbols, time format                                                                                                                                                                                                               |
| constraint      | Generic constraints                                                                                                                                                                                                                                                              |
| di              | Dependency Injection based on uber dig                                                                                                                                                                                                                                           |
| env             | Retrieves runtime environment information                                                                                                                                                                                                                                        |
| fus             | Gofusion CLI, usage details in binary, install: go install github.com/wfusion/gofusion/common/fus@master                                                                                                                                                                         |
| infra           | Infrastructure, currently includes: mongo driver, common relational database drivers, redis driver, asynq asynchronous and scheduled task framework, watermill event-driven framework                                                                                            |
| utils/cipher    | Encryption/Decryption, supports des, 3des, aes, sm4, rc4, chacha20poly1305, xchacha20poly1305 encryption algorithms, ebc, cbc, cfb, ctr, ofb, gcm encryption modes, supports streaming processing                                                                                |
| utils/clone     | Supports deep copy of any type, based on github.com/huandu/go-clone                                                                                                                                                                                                              |
| utils/cmp       | Supports comparison of common types                                                                                                                                                                                                                                              |
| utils/compress  | Compression, supports zstd, zlib, s2, gzip, deflate compression algorithms, supports streaming processing, based on github.com/klauspost/compress                                                                                                                                |
| utils/serialize | Serialization, supports gob, json, msgpack, cbor serialization algorithms                                                                                                                                                                                                        |
| utils/sqlparser | SQL statement parsing, based on github.com/longbridgeapp/sqlparser and supports OFFSET FETCH statements                                                                                                                                                                          |
| utils/encode    | Printable encoding, supports hex, base32, base32-hex, base64, base64-url, base64-raw, base64-raw-url encoding algorithms, supports streaming processing, encryption, compression, and printable encoding combinations for []byte input processing and stream processing wrapping |
| utils/gomonkey  | Monkey patch, fork from github.com/agiledragon/gomonkey                                                                                                                                                                                                                          |
| utils/inspect   | Supports private field read/write, supports function and reflect.Type retrieval based on string, inspired by github.com/chenzhuoyu/go-inspect                                                                                                                                    |
| utils/pool.go   | Pooling generic encapsulation, by default provides pooled objects for bytes.Buffer and []byte                                                                                                                                                                                    |
| utils/xxx.go    | Other common functions or wrappers, covering compare, context, conv, enum, func, heap, ip, json, map, number, options, random, reflect, sets, slice, sort, string, time                                                                                                          |

# Thanks for

> The repository we forked 

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

# Todo List

- [ ] metrics each component
- [ ] mq component support event deduplication and outdated message discarding
- [ ] config component support Apollo dynamic config with viper lib
- [ ] http client wrapper for passing through trace id
- [ ] support rpc component
- [ ] support watermill type in cron and async components