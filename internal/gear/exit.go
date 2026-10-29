package gear

// ExitError is a docker CLI failure that should become the process exit code.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "exit status " + itoa(e.Code)
}

func (e *ExitError) Unwrap() error { return e.Err }

// ExitCode is read by urfave/cli so the process exits with docker's status.
func (e *ExitError) ExitCode() int { return e.Code }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
