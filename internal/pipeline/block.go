package pipeline

import "github.com/bcross2/pipeforge-tui/internal/commands"

type Block struct {
	ID     int
	Type   string
	Config map[string]any
}

func NewBlock(id int, cmdType string) Block {
	def := commands.Registry[cmdType]
	config := make(map[string]any)
	for k, v := range def.Defaults {
		config[k] = v
	}
	return Block{ID: id, Type: cmdType, Config: config}
}

func (b Block) GetString(key string) string {
	if v, ok := b.Config[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (b Block) GetBool(key string) bool {
	if v, ok := b.Config[key]; ok {
		if bv, ok := v.(bool); ok {
			return bv
		}
	}
	return false
}

func (b Block) GetInt(key string) int {
	if v, ok := b.Config[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case float64:
			return int(n)
		}
	}
	return 0
}
