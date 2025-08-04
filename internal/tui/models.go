package tui

var (
	mainModelOptions = []ConfigItem{
		{Name: "Edit config"},
		{Name: "Preview and Run"},
	}
	configMenuOptions = []ConfigItem{
		{Name: "Manual change"},
		{Name: "Load file"},
		{Name: "Save file"},
	}
	configPreviewOptions = []ConfigItem{
		{Name: "Back to main menu"},
		{Name: "Run application"},
	}
)

type Action string

const (
	ActionLoad Action = "load"
	ActionSave Action = "save"
)

type FileSelectedMsg struct {
	Path   string
	Action Action
}

type ConfigItem struct {
	Name     string
	Value    any
	IsStruct bool
}
