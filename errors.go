package main

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

func SyntaxError(input string) *TaroError {
	return GenericSyntaxError("ParseHexColorFast", input, "invalid syntax")
}

func GenericError(fn, action, err string) *TaroError {
	return &TaroError{fn, action, err}
}
