package util

type TaroError struct {
	Func   string
	Action string
	Err    string
}

func (e *TaroError) Error() string {
	return "taro." + e.Func + ":\n    error with: " + e.Action + "\n    because: " + e.Err
}

func GenericSyntaxError(fn, input, reason string) *TaroError {
	return &TaroError{fn, "parsing \"" + input + "\"", reason}
}

func SyntaxError(fn, input string) *TaroError {
	return GenericSyntaxError(fn, input, "invalid syntax")
}

func GenericError(fn, action, err string) *TaroError {
	return &TaroError{fn, action, err}
}
