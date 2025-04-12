package validator

import (
	"time"
)

// Proposal represents a governance proposal
type Proposal struct {
	ID          string
	Type        string
	Title       string
	Description string
	Status      string
	CreatedAt   time.Time
	Votes       map[string]bool
}

// Vote represents a vote on a proposal
type Vote struct {
	Voter    string
	Proposal string
	Approve  bool
	Time     time.Time
} 