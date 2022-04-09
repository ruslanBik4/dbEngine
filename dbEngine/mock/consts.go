package mock

import "regexp"

const (
	CONN_MOCK_ENV = "TEST_ENV"
)

var regSQl = regexp.MustCompile(`(?i)select\s+([\s\S]+)+\s+from\s+`)
