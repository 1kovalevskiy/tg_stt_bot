package logger

import "time"

// Close stops accepting records and waits for the queue to drain. It is safe
// to call more than once: the second call finds the sink already closed.
func (s *ServiceChatSink) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()

		return nil
	}

	s.closed = true
	close(s.queue)
	s.mu.Unlock()

	select {
	case <-s.done:
		return nil
	case <-time.After(s.drainTimeout):
		return ErrSinkDrainTimeout
	}
}
