package clock

import "time"

type AdvanceTimeCommand struct {
	By time.Duration
}

type AdvanceTimeResult struct {
	Current time.Time
}

type TimeAdvancer struct {
	clock interface {
		Advance(time.Duration) error
		Now() time.Time
	}
}

func NewTimeAdvancer(clock interface {
	Advance(time.Duration) error
	Now() time.Time
}) (*TimeAdvancer, error) {
	if clock == nil {
		return nil, ErrInvalidClock
	}

	return &TimeAdvancer{clock: clock}, nil
}

func (a *TimeAdvancer) Execute(command AdvanceTimeCommand) (AdvanceTimeResult, error) {
	if command.By <= 0 {
		return AdvanceTimeResult{}, ErrInvalidAdvance
	}

	if err := a.clock.Advance(command.By); err != nil {
		return AdvanceTimeResult{}, err
	}

	return AdvanceTimeResult{Current: a.clock.Now()}, nil
}
