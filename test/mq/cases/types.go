package cases

import "time"

const (
	nameDefault = "default"

	nameRawRabbitmq  = "raw_rabbitmq"
	nameRawKafka     = "raw_kafka"
	nameRawPulsar    = "raw_pulsar"
	nameRawRedis     = "raw_redis"
	nameRawMysql     = "raw_mysql"
	nameRawPostgres  = "raw_postgres"
	nameRawGoChannel = "raw_gochannel"

	nameEventRabbitmq  = "event_rabbitmq"
	nameEventKafka     = "event_kafka"
	nameEventPulsar    = "event_pulsar"
	nameEventRedis     = "event_redis"
	nameEventMysql     = "event_mysql"
	nameEventPostgres  = "event_postgres"
	nameEventGoChannel = "event_gochannel"

	ackTimeout = 2 * time.Second
	timeout    = 20 * time.Second
)

type cs struct {
	name    string
	subTest func()
}

type structCreated struct {
	ID string `json:"id"`
}

func (s *structCreated) EventType() string {
	return "struct_created"
}
