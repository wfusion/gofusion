package cqrs

import (
	"github.com/wfusion/gofusion/common/infra/watermill/message"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/serialize/json"
)

type JSONMarshaler struct {
	NewUUID      func() string
	GenerateName func(v any) string
}

func (m JSONMarshaler) Marshal(v any) (*message.Message, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	msg := message.NewMessage(
		m.newUUID(),
		b,
	)
	msg.Metadata.Set("name", m.Name(v))

	return msg, nil
}

func (m JSONMarshaler) newUUID() string {
	if m.NewUUID != nil {
		return m.NewUUID()
	}

	// default
	return utils.UUID()
}

func (JSONMarshaler) Unmarshal(msg *message.Message, v any) (err error) {
	return json.Unmarshal(msg.Payload, v)
}

func (m JSONMarshaler) Name(cmdOrEvent any) string {
	if m.GenerateName != nil {
		return m.GenerateName(cmdOrEvent)
	}

	return FullyQualifiedStructName(cmdOrEvent)
}

func (m JSONMarshaler) NameFromMessage(msg *message.Message) string {
	return msg.Metadata.Get("name")
}
