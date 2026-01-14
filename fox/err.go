package main

type ParseError struct {
	File   string
	Line   int
	Column int
	Msg    string
}

/*
func TrackError() {
	if r := recover(); r != nil {
		if e, ok := r.(ParseError); ok {
			fmt.Printf("%s:%d:%d: %s\n", e.File, e.Line, e.Column, e.Msg)
		} else {
			panic(r)
		}
	}
}()
*/
