package sync

import (
	"context"
	"reflect"
)

// TODO: Support cancellation?
type Channel interface {
	Send(ctx context.Context, v interface{})

	SendNonblocking(ctx context.Context, v interface{}) (ok bool)

	Receive(ctx context.Context, vptr interface{}) (more bool)

	ReceiveNonblocking(ctx context.Context, vptr interface{}) (ok bool)

	Close()
}

func NewChannel() Channel {
	return &channel{
		c: make([]interface{}, 0),
	}
}

func NewBufferedChannel(size int) Channel {
	return &channel{
		c:    make([]interface{}, size),
		size: size,
	}
}

type channel struct {
	c         []interface{}
	receivers []func(interface{})
	closed    bool
	size      int
}

func (c *channel) Close() {
	c.closed = true

	// TODO: Wake up all blocked sends
	// TODO: Wake up all blocked receives
	// for len(c.receivers) > 0 {
	// 	r := c.receivers[0]
	// 	c.receivers[0] = nil
	// 	c.receivers = c.receivers[1:]

	// 	r(nil) // TODO: Send closed
	// }
}

func (c *channel) Send(ctx context.Context, v interface{}) {
	cr := getCoState(ctx)

	for {
		if c.send(v) {
			return
		}

		cr.Yield()
	}
}

func (c *channel) SendNonblocking(ctx context.Context, v interface{}) bool {
	return c.send(v)
}

func (c *channel) Receive(ctx context.Context, vptr interface{}) (more bool) {
	cr := getCoState(ctx)

	addedListener := false
	receivedValue := false

	for {
		// Try to receive from buffered channel
		if c.receive(vptr) {
			return !c.closed
		}

		// Register handler to receive value once
		if !addedListener {
			cb := func(v interface{}) {
				receivedValue = true
				reflect.ValueOf(vptr).Elem().Set(reflect.ValueOf(v))
			}
			c.receivers = append(c.receivers, cb)
			addedListener = true
		}

		// Yield and wait for value
		cr.Yield()

		if receivedValue {
			return !c.closed
		}
	}
}

func (c *channel) ReceiveNonblocking(ctx context.Context, vptr interface{}) (ok bool) {
	return c.receive(vptr)
}

func (c *channel) hasValue() bool {
	return len(c.c) > 0
}

func (c *channel) send(v interface{}) bool {
	if c.closed {
		panic("channel closed")
	}

	if len(c.receivers) > 0 {
		r := c.receivers[0]
		c.receivers[0] = nil
		c.receivers = c.receivers[1:]
		r(v)
		return true
	}

	// No waiting receiver, if we have capacity try to add
	if c.hasCapacity() {
		c.c = append(c.c, v)
		return true
	}

	return false
}

func (c *channel) receive(vptr interface{}) bool {
	if c.closed {
		panic("channel closed")
	}

	if c.hasValue() {
		v := c.c[0]
		c.c = c.c[1:]

		// TODO: Assert pointer
		reflect.ValueOf(vptr).Elem().Set(reflect.ValueOf(v))
		return true
	}

	return false
}

func (c *channel) hasCapacity() bool {
	return len(c.c) < c.size
}
