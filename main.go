package main

import (
	"fmt"
	"sync"
	"time"
)

type Message struct {
	Topic   string
	Payload interface{}
}

type Subscriber struct {
	Channel     chan interface{}
	Unsubscribe chan bool
}

type Broker struct {
	subscribers map[string][]*Subscriber
	mutex       sync.Mutex
}

func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[string][]*Subscriber),
	}
}

func (b *Broker) Subscribe(topic string) *Subscriber {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	subscriber := &Subscriber{
		Channel:     make(chan interface{}, 1),
		Unsubscribe: make(chan bool),
	}

	b.subscribers[topic] = append(b.subscribers[topic], subscriber)

	return subscriber
}

func (b *Broker) Unsubscribe(topic string, subscriber *Subscriber) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if subscribers, found := b.subscribers[topic]; found {
		for i, sub := range subscribers {
			if sub == subscriber {
				close(sub.Channel)
				b.subscribers[topic] = append(subscribers[:i], subscribers[i+1:]...)
				return
			}
		}
	}
}

func (b *Broker) Publish(topic string, payload interface{}) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if subscribers, found := b.subscribers[topic]; found {
		for _, sub := range subscribers {
			select {
			case sub.Channel <- payload:
			case <-time.After(time.Second):
				fmt.Println("timed out, Unsubscribing from the topic : %s\n", topic)
				b.Unsubscribe(topic, sub)
			}
		}
	}
}

func main() {
	broker := NewBroker()

	subscriber := broker.Subscribe("new_topic")

	go func() {
		for {
			select {
			case msg, ok := <-subscriber.Channel:
				if !ok {
					fmt.Println("subcriber channel closed")
					return
				}
				fmt.Println("Received message : %v\n", msg)
			case <-subscriber.Unsubscribe:
				fmt.Println("Unsubscribed.")
				return
			}
		}
	}()

	broker.Publish("new_topic", "hello world")
	broker.Publish("new_topic", "This is a test message")

	time.Sleep(2 * time.Second)
	broker.Unsubscribe("new_topic", subscriber)

	broker.Publish("new_topic", "This message is after unsubscribe")

	time.Sleep(time.Second)
}
