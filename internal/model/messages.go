package model

type AddBlockMsg struct {
	Type string
}

type RemoveBlockMsg struct {
	Index int
}

type ConfigChangedMsg struct{}
