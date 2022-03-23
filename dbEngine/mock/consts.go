package mock

import "regexp"

const (
	CONN_MOCK_ENV = "TEST_ENV"
)

var regSQl = regexp.MustCompile(`select(\s+\S+)+from\s+`)
