package tester

import (
	"github.com/JN0V/workline/internal/kit/testmark"
)

// The marks the tester leaves that others read live in the kit
// (internal/kit/testmark); the tester writes them under these names.
var (
	answerMark  = testmark.AnswerMark
	personMark  = testmark.PersonMark
	settleMark  = testmark.SettleMark
	blockedMark = testmark.BlockedMark
	unreadMark  = testmark.UnreadMark
)

const (
	branchPrefix = testmark.TestsBranch
	curateKey    = testmark.CurateKey
	refusedPre   = testmark.RefusedPre
	refusedRun   = testmark.RefusedRun
)
