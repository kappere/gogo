package panics

type BizPanic struct {
	Message string
	Code    int
}

func (p BizPanic) String() string {
	return p.Message
}

func NewBizPanic(message string) *BizPanic {
	return &BizPanic{
		Message: message,
		Code:    -1,
	}
}

func NewBizPanicWithCode(message string, code int) *BizPanic {
	return &BizPanic{
		Message: message,
		Code:    code,
	}
}
