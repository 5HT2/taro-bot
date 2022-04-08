package bot

type Error struct {
	Func   string
	Action string
	Err    string
}

func (e *Error) Error() string {
	return "taro." + e.Func + ":\n    error with: " + e.Action + "\n    because: " + e.Err
}

func GenericSyntaxError(fn, input, reason string) *Error {
	return &Error{fn, "parsing \"" + input + "\"", reason}
}

func SyntaxError(fn, input string) *Error {
	return GenericSyntaxError(fn, input, "invalid syntax")
}

func GenericError(fn, action, err string) *Error {
	return &Error{fn, action, err}
}
