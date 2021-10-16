package main

type TaroError struct {
	Func   string
	Action string
	Err    string
}

func (e *TaroError) Error() string {
	return "taro." + e.Func + ": " + e.Action + " " + e.Err
}

func SyntaxError(input string) *TaroError {
	return &TaroError{"ParseHexColorFast", "parsing \"" + input + "\"", "invalid syntax"}
}
