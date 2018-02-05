package log

type noopAggregator struct {
	ch chan error
}

func newNoopAggregator() *noopAggregator {
	return &noopAggregator{}
}

func (n *noopAggregator) Listen() error {
	return nil
}

func (n *noopAggregator) Stop() error {
	return nil
}

func (n *noopAggregator) Stopped() <-chan error {
	n.ch = make(chan error, 1)
	return n.ch
}
