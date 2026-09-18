package branch

import "errors"

var (
	ErrDuplicateBranch = errors.New("branch already exists in this organization")
)
