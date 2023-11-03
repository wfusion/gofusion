package sql

import (
	"database/sql"

	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/infra/watermill/message"
	"github.com/wfusion/gofusion/common/utils/serialize/json"
)

// SchemaAdapter produces the SQL queries and arguments appropriately for a specific schema and dialect
// It also transforms sql.Rows into Watermill messages.
type SchemaAdapter interface {
	// InsertQuery returns the SQL query and arguments that will insert the Watermill message into the SQL storage.
	InsertQuery(topic string, msgs message.Messages) (string, []any, error)

	// SelectQuery returns the SQL query and arguments
	// that returns the next unread message for a given consumer group.
	SelectQuery(topic string, consumerGroup string, offsetsAdapter OffsetsAdapter) (string, []any)

	// DeleteQuery returns the SQL query and arguments
	DeleteQuery(topic string, offset int64) (string, []any)

	// UnmarshalMessage transforms the Row obtained SelectQuery a Watermill message.
	// It also returns the offset of the last read message, for the purpose of acking.
	UnmarshalMessage(row Scanner) (Row, error)

	// SchemaInitializingQueries returns SQL queries which will make sure (CREATE IF NOT EXISTS)
	// that the appropriate tables exist to write messages to the given topic.
	SchemaInitializingQueries(topic string) []string

	// SubscribeIsolationLevel returns the isolation level that will be used when subscribing.
	SubscribeIsolationLevel() sql.IsolationLevel
}

// Deprecated: Use DefaultMySQLSchema instead.
type DefaultSchema = DefaultMySQLSchema

type Row struct {
	Offset   int64
	UUID     []byte
	Payload  []byte
	Metadata []byte

	Msg *message.Message

	ExtraData map[string]any
}

func defaultInsertArgs(msgs message.Messages) ([]any, error) {
	var args []any
	for _, msg := range msgs {
		metadata, err := json.Marshal(msg.Metadata)
		if err != nil {
			return nil, errors.Wrapf(err, "could not marshal metadata into JSON for message %s", msg.UUID)
		}

		args = append(args, msg.UUID, []byte(msg.Payload), metadata)
	}

	return args, nil
}
