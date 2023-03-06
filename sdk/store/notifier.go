package store

type Notifier interface {
	Close()
	Closed() <-chan struct{}
	Notify()
	Modified() <-chan struct{}
}

type notifier struct {
	stop     chan struct{}
	modified chan struct{}
}

func NewNotifier() Notifier {
	return &notifier{
		stop:     make(chan struct{}),
		modified: make(chan struct{}, 1),
	}
}

func (n *notifier) Closed() <-chan struct{} {
	return n.stop
}

func (n *notifier) Modified() <-chan struct{} {
	return n.modified
}

func (n *notifier) Notify() {
	select {
	case <-n.stop:
		return
	default:
		n.modified <- struct{}{}
	}
}

func (n *notifier) Close() {
	close(n.stop)
}
