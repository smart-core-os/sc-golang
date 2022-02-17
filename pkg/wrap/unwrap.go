package wrap

// Unwrapper defines the Unwrap method for exposing underlying implementation types.
type Unwrapper interface {
	Unwrap() interface{}
}

// UnwrapFully repeatedly casts then unwraps obj until obj no longer implements Unwrapper.
func UnwrapFully(obj interface{}) interface{} {
	for t, ok := obj.(Unwrapper); ok; t, ok = obj.(Unwrapper) {
		obj = t.Unwrap()
	}
	return obj
}
