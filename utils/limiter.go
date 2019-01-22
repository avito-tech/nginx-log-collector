package utils

type Limiter chan struct{}

func (l Limiter) Enter() { l <- struct{}{} }
func (l Limiter) Leave() { <-l }

func NewLimiter(l int) Limiter {
	return make(chan struct{}, l)
}
