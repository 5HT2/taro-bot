package main

type TaroError struct {
	Func   string
	Action string
	Err    string
}

func (e *TaroError) Error() string {
	return "taro." + e.Func + ":\n    `error with: " + e.Action + "\n    because: " + e.Err
}

func SyntaxError(input string) *TaroError {
	return &TaroError{"ParseHexColorFast", "parsing \"" + input + "\"", "invalid syntax"}
}

func GenericError(fn, action, err string) *TaroError {
	return &TaroError{fn, action, err}
}
