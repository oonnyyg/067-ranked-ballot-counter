package main

// InvalidBallot 描述一张被排除的选票及其在输入 ballots 数组中的原始序号（从 0 开始）。
type InvalidBallot struct {
	Index  int    `json:"index"`
	Reason string `json:"reason"`
}

// RoundTransfer 描述某轮淘汰者名下选票在淘汰后的去向。
type RoundTransfer struct {
	Round           int              `json:"round"`
	Eliminated      string           `json:"eliminated"`
	EliminatedVotes int64            `json:"eliminated_votes"`
	ToCandidates    map[string]int64 `json:"to_candidates"`
	ToExhausted     int64            `json:"to_exhausted"`
}

// RoundRecord 是一轮计票的完整可复核记录。
type RoundRecord struct {
	Round               int              `json:"round"`
	RemainingCandidates []string         `json:"remaining_candidates"`
	Tallies             map[string]int64 `json:"tallies"`
	NonExhaustedWeight  int64            `json:"non_exhausted_weight"`
	ExhaustedWeight     int64            `json:"exhausted_weight_cumulative"`
	Winner              string           `json:"winner,omitempty"`
	Eliminated          string           `json:"eliminated,omitempty"`
	Transfers           *RoundTransfer   `json:"transfers,omitempty"`
}

// Result 是 result.json 的固定结构。
type Result struct {
	Winner           *string         `json:"winner"`
	RoundsCount      int             `json:"rounds_count"`
	ValidBallots     int             `json:"valid_ballots"`
	InvalidBallots   int             `json:"invalid_ballots"`
	TotalValidWeight int64           `json:"total_valid_weight"`
	InvalidDetails   []InvalidBallot `json:"invalid_ballot_details"`
	Rounds           []RoundRecord   `json:"rounds"`
}
