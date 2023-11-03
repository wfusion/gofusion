package redis

import (
	"github.com/pkg/errors"
	"github.com/vmihailenco/msgpack/v5"
	"github.com/wfusion/gofusion/common/infra/watermill"

	"github.com/wfusion/gofusion/common/infra/watermill/message"
)

const UUIDHeaderKey = "_watermill_message_uuid"

type Marshaller interface {
	Marshal(topic string, msg *message.Message) (map[string]any, error)
}

type Unmarshaller interface {
	Unmarshal(values map[string]any) (msg *message.Message, err error)
}

type MarshallerUnmarshaller interface {
	Marshaller
	Unmarshaller
}

type DefaultMarshallerUnmarshaller struct {
	AppID string
}

func (d DefaultMarshallerUnmarshaller) Marshal(_ string, msg *message.Message) (map[string]any, error) {
	if value := msg.Metadata.Get(UUIDHeaderKey); value != "" {
		return nil, errors.Errorf("metadata %s is reserved by watermill for message UUID", UUIDHeaderKey)
	}

	var (
		md  []byte
		err error
	)

	msg.Metadata[watermill.MessageHeaderAppID] = d.AppID
	if len(msg.Metadata) > 0 {
		if md, err = msgpack.Marshal(msg.Metadata); err != nil {
			return nil, errors.Wrapf(err, "marshal metadata fail")
		}
	}

	return map[string]any{
		UUIDHeaderKey:                msg.UUID,
		watermill.MessageHeaderAppID: d.AppID,
		"metadata":                   md,
		"payload":                    []byte(msg.Payload),
	}, nil
}

func (DefaultMarshallerUnmarshaller) Unmarshal(values map[string]any) (msg *message.Message, err error) {
	msg = message.NewMessage(values[UUIDHeaderKey].(string), []byte(values["payload"].(string)))

	mdv, ok1 := values["metadata"]
	mds, ok2 := mdv.(string)
	if ok1 && ok2 && mds != "" {
		metadata := make(message.Metadata)
		if err := msgpack.Unmarshal([]byte(mds), &metadata); err != nil {
			return nil, errors.Wrapf(err, "unmarshal metadata fail")
		}
		msg.Metadata = metadata
	}

	return msg, nil
}
