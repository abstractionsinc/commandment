package shelltest

import "strings"

const (
	windowsOS                   = "windows"
	shelltestFailureDetailLimit = 4096
)

type failureError struct {
	subject string
	detail  string
	cause   error
}

func (err failureError) Error() string {
	var message strings.Builder
	message.WriteString("shelltest: ")
	message.WriteString(err.subject)
	if err.detail != "" {
		message.WriteString(": ")
		message.WriteString(err.detail)
	}
	if err.cause != nil {
		message.WriteString(": ")
		message.WriteString(err.cause.Error())
	}
	return message.String()
}

func (err failureError) Unwrap() error { return err.cause }

func boundedFailureDetail(detail string) string {
	if len(detail) <= shelltestFailureDetailLimit {
		return detail
	}
	const marker = "…[truncated]"
	limit := shelltestFailureDetailLimit - len(marker)
	if limit < 0 {
		return marker
	}
	return detail[:limit] + marker
}
