package commands

type FieldType int

const (
	FieldText FieldType = iota
	FieldCheck
	FieldNumber
	FieldSelect
)

type SelectOption struct {
	Value string
	Label string
}

type ConfigField struct {
	Key         string
	Type        FieldType
	Label       string
	Placeholder string
	Hint        string
	Options     []SelectOption
}

type CommandDef struct {
	Label    string
	Excel    string
	Group    string
	Icon     string
	Defaults map[string]any
	Config   []ConfigField
}

type Group struct {
	ID    string
	Label string
}
