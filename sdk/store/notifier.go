package store

import "context"

type Notifier interface {
	Close()
	Notify()
	Modified() <-chan struct{}
	Context() context.Context
}

type notifier struct {
	ctx       context.Context
	ctxCancel func()
	modified  chan struct{}
}

func NewNotifier() Notifier {
	n := &notifier{
		modified: make(chan struct{}, 1),
	}
	n.ctx, n.ctxCancel = context.WithCancel(context.Background())
	return n
}

func (n *notifier) Closed() <-chan struct{} {
	return n.ctx.Done()
}

func (n *notifier) Modified() <-chan struct{} {
	return n.modified
}

func (n *notifier) Notify() {
	select {
	case <-n.ctx.Done():
		return
	default:
		n.modified <- struct{}{}
	}
}

func (n *notifier) Close() {
	n.ctxCancel()
}

func (n *notifier) Context() context.Context {
	return n.ctx
}
