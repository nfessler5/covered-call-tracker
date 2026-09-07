package coveredcalltracker

import (
	"context"
	positions "covered-call-tracker/gen/positions"

	"goa.design/clue/log"
)

// positions service example implementation.
// The example methods log the requests and return zero values.
type positionssrvc struct{}

// NewPositions returns the positions service implementation.
func NewPositions() positions.Service {
	return &positionssrvc{}
}

// List active or historical covered call positions.
func (s *positionssrvc) List(ctx context.Context, p *positions.ListPayload) (res []*positions.CoveredCallPosition, err error) {
	log.Printf(ctx, "positions.list")
	return
}

// Open a new covered call position.
func (s *positionssrvc) Create(ctx context.Context, p *positions.CoveredCallPosition) (res *positions.CoveredCallPosition, err error) {
	res = &positions.CoveredCallPosition{}
	log.Printf(ctx, "positions.create")
	return
}

// Close an existing position as ROLLED and create a new open position in one
// atomic transaction.
func (s *positionssrvc) Roll(ctx context.Context, p *positions.RollPayload) (res *positions.CoveredCallPosition, err error) {
	res = &positions.CoveredCallPosition{}
	log.Printf(ctx, "positions.roll")
	return
}
