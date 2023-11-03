package gochannel

import (
	"context"
	"math/rand"
	"sync"

	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/infra/watermill"
	"github.com/wfusion/gofusion/common/infra/watermill/message"
	"github.com/wfusion/gofusion/common/utils"
)

// Config holds the GoChannel Pub/Sub's configuration options.
type Config struct {
	// Output channel buffer size.
	OutputChannelBuffer int64

	// If persistent is set to true, when subscriber subscribes to the topic,
	// it will receive all previously produced messages.
	//
	// All messages are persisted to the memory (simple slice),
	// so be aware that with large amount of messages you can go out of the memory.
	Persistent bool

	// When true, Publish will block until subscriber Ack's the message.
	// If there are no subscribers, Publish will not block (also when Persistent is true).
	BlockPublishUntilSubscriberAck bool

	ConsumerGroup string

	AppID string
}

// GoChannel is the simplest Pub/Sub implementation.
// It is based on Golang's channels which are sent within the process.
//
// GoChannel has no global state,
// that means that you need to use the same instance for Publishing and Subscribing!
//
// When GoChannel is persistent, messages order is not guaranteed.
type GoChannel struct {
	config Config
	logger watermill.LoggerAdapter

	subscribersWg          sync.WaitGroup
	subscribers            map[string]map[string][]*subscriber
	subscribersLock        sync.RWMutex
	subscribersByTopicLock sync.Map // map of *sync.Mutex

	closed     bool
	closedLock sync.Mutex
	closing    chan struct{}

	persistedMessages     map[string][]*message.Message
	persistedMessagesLock sync.RWMutex
}

// NewGoChannel creates new GoChannel Pub/Sub.
//
// This GoChannel is not persistent.
// That means if you send a message to a topic to which no subscriber is subscribed, that message will be discarded.
func NewGoChannel(config Config, logger watermill.LoggerAdapter) *GoChannel {
	if logger == nil {
		logger = watermill.NopLogger{}
	}

	return &GoChannel{
		config: config,

		subscribers:            make(map[string]map[string][]*subscriber),
		subscribersByTopicLock: sync.Map{},
		logger: logger.With(watermill.LogFields{
			"pubsub_uuid": utils.ShortUUID(),
		}),

		closing: make(chan struct{}),

		persistedMessages: map[string][]*message.Message{},
	}
}

// Publish in GoChannel is NOT blocking until all consumers consume.
// Messages will be sent in background.
//
// Messages may be persisted or not, depending on persistent attribute.
func (g *GoChannel) Publish(ctx context.Context, topic string, messages ...*message.Message) error {
	if g.isClosed() {
		return errors.New("Pub/Sub closed")
	}

	messagesToPublish := make(message.Messages, len(messages))
	for i, msg := range messages {
		messagesToPublish[i] = msg.Copy()
	}

	g.subscribersLock.RLock()
	defer g.subscribersLock.RUnlock()

	subLock, _ := g.subscribersByTopicLock.LoadOrStore(topic, &sync.Mutex{})
	subLock.(*sync.Mutex).Lock()
	defer subLock.(*sync.Mutex).Unlock()

	if g.config.Persistent {
		g.persistedMessagesLock.Lock()
		if _, ok := g.persistedMessages[topic]; !ok {
			g.persistedMessages[topic] = make([]*message.Message, 0)
		}
		g.persistedMessages[topic] = append(g.persistedMessages[topic], messagesToPublish...)
		g.persistedMessagesLock.Unlock()
	}

	for i := range messagesToPublish {
		msg := messagesToPublish[i]

		ackedBySubscribers, err := g.sendMessage(ctx, topic, msg)
		if err != nil {
			return err
		}

		if g.config.BlockPublishUntilSubscriberAck {
			g.waitForAckFromSubscribers(msg, ackedBySubscribers)
		}
	}

	return nil
}

func (g *GoChannel) waitForAckFromSubscribers(msg *message.Message, ackedByConsumer <-chan struct{}) {
	logFields := watermill.LogFields{"message_uuid": msg.UUID}
	g.logger.Debug("[Common] watermill gochannel waiting for subscribers ack", logFields)

	select {
	case <-ackedByConsumer:
		g.logger.Trace("[Common] watermill gochannel message acked by subscribers", logFields)
	case <-g.closing:
		g.logger.Trace("[Common] watermill gochannel closing pub/sub before ack from subscribers", logFields)
	}
}

func (g *GoChannel) sendMessage(ctx context.Context, topic string, message *message.Message) (<-chan struct{}, error) {
	subscribers := g.topicSubscribers(topic)
	ackedBySubscribers := make(chan struct{})

	logFields := watermill.LogFields{"message_uuid": message.UUID, "topic": topic}

	if len(subscribers) == 0 {
		close(ackedBySubscribers)
		g.logger.Info("[Common] watermill gochannel none subscribers to send message", logFields)
		return ackedBySubscribers, nil
	}

	go func(subscribers map[string][]*subscriber) {
		wg := &sync.WaitGroup{}

		if noneGroupSubs, ok := subscribers[""]; ok {
			for i := range noneGroupSubs {
				subscriber := noneGroupSubs[i]

				wg.Add(1)
				go func() {
					subscriber.sendMessageToSubscriber(message, logFields, g.config)
					wg.Done()
				}()
			}
			delete(subscribers, "")
		}
		for _, subs := range subscribers {
			rand.Shuffle(len(subs), func(i, j int) { subs[i], subs[j] = subs[j], subs[i] })
			subscriber := subs[0]
			wg.Add(1)
			go func() {
				subscriber.sendMessageToSubscriber(message, logFields, g.config)
				wg.Done()
			}()
		}

		wg.Wait()
		close(ackedBySubscribers)
	}(subscribers)

	return ackedBySubscribers, nil
}

// Subscribe returns channel to which all published messages are sent.
// Messages are not persisted. If there are no subscribers and message is produced it will be gone.
//
// There are no consumer groups support etc. Every consumer will receive every produced message.
func (g *GoChannel) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	g.closedLock.Lock()

	if g.closed {
		g.closedLock.Unlock()
		return nil, errors.New("pub/sub closed")
	}

	g.subscribersWg.Add(1)
	g.closedLock.Unlock()

	g.subscribersLock.Lock()

	subLock, _ := g.subscribersByTopicLock.LoadOrStore(topic, &sync.Mutex{})
	subLock.(*sync.Mutex).Lock()

	s := &subscriber{
		ctx:           ctx,
		uuid:          utils.UUID(),
		outputChannel: make(chan *message.Message, g.config.OutputChannelBuffer),
		logger:        g.logger,
		closing:       make(chan struct{}),
		g:             g,
	}

	go func(s *subscriber, g *GoChannel) {
		select {
		case <-ctx.Done():
			// unblock
		case <-g.closing:
			// unblock
		}

		s.Close()

		g.subscribersLock.Lock()
		defer g.subscribersLock.Unlock()

		subLock, _ := g.subscribersByTopicLock.Load(topic)
		subLock.(*sync.Mutex).Lock()
		defer subLock.(*sync.Mutex).Unlock()

		g.removeSubscriber(topic, g.config.ConsumerGroup, s)
		g.subscribersWg.Done()
	}(s, g)

	if !g.config.Persistent {
		defer g.subscribersLock.Unlock()
		defer subLock.(*sync.Mutex).Unlock()

		g.addSubscriber(topic, g.config.ConsumerGroup, s)

		return s.outputChannel, nil
	}

	go func(s *subscriber) {
		defer g.subscribersLock.Unlock()
		defer subLock.(*sync.Mutex).Unlock()

		g.persistedMessagesLock.RLock()
		messages, ok := g.persistedMessages[topic]
		g.persistedMessagesLock.RUnlock()

		if ok && g.config.ConsumerGroup == "" {
			for i := 0; i < len(messages); i++ {
				msg := g.persistedMessages[topic][i]
				logFields := watermill.LogFields{"message_uuid": msg.UUID, "topic": topic}

				go s.sendMessageToSubscriber(msg, logFields, g.config)
			}
		}

		g.addSubscriber(topic, g.config.ConsumerGroup, s)
	}(s)

	return s.outputChannel, nil
}

func (g *GoChannel) addSubscriber(topic, group string, s *subscriber) {
	if _, ok := g.subscribers[topic]; !ok {
		g.subscribers[topic] = make(map[string][]*subscriber)
	}
	g.subscribers[topic][group] = append(g.subscribers[topic][group], s)
}

func (g *GoChannel) removeSubscriber(topic, group string, toRemove *subscriber) {
	removed := false
	for _, groupSub := range g.subscribers[topic] {
		for i, sub := range groupSub {
			if sub == toRemove {
				g.subscribers[topic][group] = append(g.subscribers[topic][group][:i],
					g.subscribers[topic][group][i+1:]...)
				removed = true
				break
			}
		}

	}
	if !removed {
		panic("cannot remove subscriber, not found " + toRemove.uuid)
	}
}

func (g *GoChannel) topicSubscribers(topic string) map[string][]*subscriber {
	subscribers, ok := g.subscribers[topic]
	if !ok {
		return nil
	}

	// let's do a copy to avoid race conditions and deadlocks due to lock
	subscribersCopy := make(map[string][]*subscriber, len(subscribers))
	for group, subs := range subscribers {
		subscribersCopy[group] = make([]*subscriber, len(subs))
		copy(subscribersCopy[group], subs)
	}

	return subscribersCopy
}

func (g *GoChannel) isClosed() bool {
	g.closedLock.Lock()
	defer g.closedLock.Unlock()

	return g.closed
}

// Close closes the GoChannel Pub/Sub.
func (g *GoChannel) Close() error {
	g.closedLock.Lock()
	defer g.closedLock.Unlock()

	if g.closed {
		return nil
	}

	g.closed = true
	close(g.closing)

	g.logger.Debug("[Common] watermill gochannel closing pub/sub, waiting for subscribers", nil)
	g.subscribersWg.Wait()

	g.logger.Info("[Common] watermill gochannel pub/sub closed", nil)
	g.persistedMessages = nil

	return nil
}

type subscriber struct {
	ctx context.Context

	uuid string

	sending       sync.Mutex
	outputChannel chan *message.Message

	logger  watermill.LoggerAdapter
	closed  bool
	closing chan struct{}

	g *GoChannel
}

func (s *subscriber) Close() {
	if s.closed {
		return
	}
	close(s.closing)

	s.logger.Debug("[Common] watermill gochannel closing subscriber, waiting for sending lock", nil)

	// ensuring that we are not sending to closed channel
	s.sending.Lock()
	defer s.sending.Unlock()

	s.logger.Debug("[Common] watermill gochannel pub/sub subscriber closed", nil)
	s.closed = true

	close(s.outputChannel)
}

func (s *subscriber) sendMessageToSubscriber(msg *message.Message, logFields watermill.LogFields, conf Config) {
	s.sending.Lock()
	defer s.sending.Unlock()

	rawMessageID := utils.NginxID()
	ctx := context.WithValue(s.ctx, watermill.ContextKeyMessageUUID, msg.UUID)
	ctx = context.WithValue(ctx, watermill.ContextKeyRawMessageID, rawMessageID)
	ctx, cancelCtx := context.WithCancel(ctx)
	defer cancelCtx()
	msg.Metadata[watermill.ContextKeyMessageUUID] = msg.UUID
	msg.Metadata[watermill.ContextKeyRawMessageID] = rawMessageID
	msg.Metadata[watermill.MessageHeaderAppID] = conf.AppID

SendToSubscriber:
	for {
		// copy the message to prevent ack/nack propagation to other consumers
		// also allows to make retries on a fresh copy of the original message
		msgToSend := msg.Copy()
		msgToSend.SetContext(ctx)

		s.logger.Trace("[Common] watermill gochannel sending msg to subscriber", logFields)

		if s.closed {
			s.logger.Info("[Common] watermill gochannel pub/sub closed, discarding msg", logFields)
			return
		}

		select {
		case s.outputChannel <- msgToSend:
			s.logger.Trace("[Common] watermill gochannel sent message to subscriber", logFields)
		case <-s.closing:
			s.logger.Trace("[Common] watermill gochannel closing, message discarded", logFields)
			return
		}

		select {
		case <-msgToSend.Acked():
			s.logger.Trace("[Common] watermill gochannel message acked", logFields)
			return
		case <-msgToSend.Nacked():
			s.logger.Trace("[Common] watermill gochannel nack received, resending message", logFields)
			continue SendToSubscriber
		case <-s.closing:
			s.logger.Trace("[Common] watermill gochannel closing, message discarded", logFields)
			return
		}
	}
}
