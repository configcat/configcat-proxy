package pubsub

type SubscriptionHandler[T any] interface {
	Subscribe(chan<- T)
	Unsubscribe(chan<- T)
	Close()
}

type Publisher[T any] interface {
	SubscriptionHandler[T]
	Publish(data T)
}

type pubSub[T any] struct {
	subscriptions map[chan<- T]struct{}
	sub           chan chan<- T
	unsub         chan chan<- T
	pub           chan T
	stop          chan struct{}
}

func NewPublisher[T any]() Publisher[T] {
	p := &pubSub[T]{
		subscriptions: make(map[chan<- T]struct{}),
		sub:           make(chan chan<- T),
		unsub:         make(chan chan<- T),
		pub:           make(chan T, 64),
		stop:          make(chan struct{}),
	}
	go p.run()
	return p
}

func (p *pubSub[T]) run() {
	for {
		select {
		case data := <-p.pub:
			for sub := range p.subscriptions {
				sub <- data
			}
		case ch := <-p.sub:
			p.subscriptions[ch] = struct{}{}
		case ch := <-p.unsub:
			delete(p.subscriptions, ch)
		case <-p.stop:
			return
		}
	}
}

func (p *pubSub[T]) Publish(data T) {
	select {
	case <-p.stop:
		return
	default:
		p.pub <- data
	}
}

func (p *pubSub[T]) Subscribe(ch chan<- T) {
	select {
	case <-p.stop:
		return
	default:
		p.sub <- ch
	}
}

func (p *pubSub[T]) Unsubscribe(ch chan<- T) {
	select {
	case <-p.stop:
		return
	default:
		p.unsub <- ch
	}
}

func (p *pubSub[T]) Close() {
	close(p.stop)
}
