package fields

const InputCode TypeCode = "input"

type Input struct{}

func NewInput() Input {
	return Input{}
}

func (f Input) Code() TypeCode {
	return InputCode
}

var _ FieldType = Input{}
