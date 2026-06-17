package fields

type Option struct {
	Value string
	Label string
}

type Config struct {
	Placeholder string
	Help        string
	Mask        string

	Min  *float64
	Max  *float64
	Step *float64

	Options  []Option
	Multiple bool

	Accept []string
}

type Definition struct {
	Code     string
	Label    string
	Type     TypeCode
	Required bool
	Config   Config
}
