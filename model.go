package fisk

import (
	"fmt"
	"strconv"
	"strings"
)

// Data model for Fisk command-line structure.

var (
	ignoreInCount = map[string]bool{
		"help":                   true,
		"help-long":              true,
		"help-man":               true,
		"completion-bash":        true,
		"completion-script-bash": true,
		"completion-script-zsh":  true,
	}
)

type FlagGroupModel struct {
	Flags []*FlagModel `json:"flags"`
}

func (f *FlagGroupModel) FlagSummary() string {
	out := []string{}
	count := 0

	for _, flag := range f.Flags {
		if !ignoreInCount[flag.Name] {
			count++
		}

		if flag.Required {
			if flag.IsBoolFlag() {
				if flag.IsNegatable() {
					out = append(out, fmt.Sprintf("--[no-]%s", flag.Name))
				} else {
					out = append(out, fmt.Sprintf("--%s=%s", flag.Name, flag.FormatPlaceHolder()))
				}
			}
		}
	}

	if count != len(out) {
		out = append(out, "[<flags>]")
	}

	return strings.Join(out, " ")
}

type FlagModel struct {
	Name        string   `json:"name"`
	Help        string   `json:"help"`
	Short       rune     `json:"short"`
	Default     []string `json:"default"`
	Envar       string   `json:"envar"`
	PlaceHolder string   `json:"place_holder"`
	Required    bool     `json:"required"`
	Hidden      bool     `json:"hidden"`

	// used by plugin model
	Boolean   bool `json:"boolean"`
	Negatable bool `json:"negatable"`

	Value Value `json:"-"`
}

func (f *FlagModel) String() string {
	if f.Value == nil {
		return ""
	}
	return f.Value.String()
}

func (f *FlagModel) IsBoolFlag() bool {
	return isBoolFlag(f.Value)
}

func (f *FlagModel) IsNegatable() bool {
	bf, ok := f.Value.(BoolFlag)
	return ok && bf.BoolFlagIsNegatable()
}

func (f *FlagModel) FormatPlaceHolder() string {
	if f.PlaceHolder != "" {
		return f.PlaceHolder
	}
	if len(f.Default) > 0 {
		ellipsis := ""
		if len(f.Default) > 1 {
			ellipsis = "..."
		}
		if _, ok := f.Value.(*stringValue); ok {
			return strconv.Quote(f.Default[0]) + ellipsis
		}
		return f.Default[0] + ellipsis
	}
	return strings.ToUpper(f.Name)
}

func (f *FlagModel) HelpWithEnvar() string {
	if f.Envar == "" {
		return f.Help
	}
	return fmt.Sprintf("%s ($%s)", f.Help, f.Envar)
}

type ArgGroupModel struct {
	Args []*ArgModel `json:"args"`
}

func (a *ArgGroupModel) ArgSummary() string {
	depth := 0
	out := []string{}
	for _, arg := range a.Args {
		var h string
		if arg.PlaceHolder != "" {
			h = arg.PlaceHolder
		} else {
			h = "<" + arg.Name + ">"
		}
		if !arg.Required {
			h = "[" + h
			depth++
		}
		out = append(out, h)
	}
	out[len(out)-1] = out[len(out)-1] + strings.Repeat("]", depth)
	return strings.Join(out, " ")
}

func (a *ArgModel) HelpWithEnvar() string {
	if a.Envar == "" {
		return a.Help
	}
	return fmt.Sprintf("%s ($%s)", a.Help, a.Envar)
}

type ArgModel struct {
	Name        string   `json:"name"`
	Help        string   `json:"help"`
	Default     []string `json:"default"`
	Envar       string   `json:"envar"`
	PlaceHolder string   `json:"place_holder"`
	Required    bool     `json:"required"`
	Hidden      bool     `json:"hidden"`
	Value       Value    `json:"-"`
}

func (a *ArgModel) String() string {
	if a.Value == nil {
		return ""
	}

	return a.Value.String()
}

type CmdGroupModel struct {
	Commands []*CmdModel `json:"commands"`
}

func (c *CmdGroupModel) FlattenedCommands() (out []*CmdModel) {
	for _, cmd := range c.Commands {
		if len(cmd.Commands) == 0 {
			out = append(out, cmd)
		}
		out = append(out, cmd.FlattenedCommands()...)
	}
	return
}

type CmdModel struct {
	Name        string   `json:"name"`
	Aliases     []string `json:"aliases"`
	Help        string   `json:"help"`
	HelpLong    string   `json:"help_long"`
	FullCommand string   `json:"-"`
	Depth       int      `json:"-"`
	Hidden      bool     `json:"hidden"`
	Default     bool     `json:"default"`

	*FlagGroupModel
	*ArgGroupModel
	*CmdGroupModel
}

func (c *CmdModel) String() string {
	return c.FullCommand
}

type ApplicationModel struct {
	Name      string            `json:"name"`
	Help      string            `json:"help"`
	Cheat     string            `json:"cheat"`
	Version   string            `json:"version"`
	Author    string            `json:"author"`
	Cheats    map[string]string `json:"cheats"`
	CheatTags []string          `json:"cheat_tags"`

	*ArgGroupModel
	*CmdGroupModel
	*FlagGroupModel
}

func (a *Application) Model() *ApplicationModel {
	return &ApplicationModel{
		Name:           a.Name,
		Help:           a.Help,
		Version:        a.version,
		Author:         a.author,
		Cheats:         a.cheats,
		CheatTags:      a.cheatTags,
		FlagGroupModel: a.flagGroup.Model(),
		ArgGroupModel:  a.argGroup.Model(),
		CmdGroupModel:  a.cmdGroup.Model(),
	}
}

func (a *argGroup) Model() *ArgGroupModel {
	m := &ArgGroupModel{}
	for _, arg := range a.args {
		m.Args = append(m.Args, arg.Model())
	}
	return m
}

func (a *ArgClause) Model() *ArgModel {
	return &ArgModel{
		Name:        a.name,
		Help:        a.help,
		Default:     a.defaultValues,
		Envar:       a.envar,
		PlaceHolder: a.placeholder,
		Required:    a.required,
		Hidden:      a.hidden,
		Value:       a.value,
	}
}

func (f *flagGroup) Model() *FlagGroupModel {
	m := &FlagGroupModel{}
	for _, fl := range f.flagOrder {
		m.Flags = append(m.Flags, fl.Model())
	}
	return m
}

func (f *FlagClause) Model() *FlagModel {
	m := &FlagModel{
		Name:        f.name,
		Help:        f.help,
		Short:       f.shorthand,
		Default:     f.defaultValues,
		Envar:       f.envar,
		PlaceHolder: f.placeholder,
		Required:    f.required,
		Hidden:      f.hidden,
		Value:       f.value,
	}

	m.Boolean = m.IsBoolFlag()
	m.Negatable = m.IsNegatable()

	return m
}

func (c *cmdGroup) Model() *CmdGroupModel {
	m := &CmdGroupModel{}
	for _, cm := range c.commandOrder {
		m.Commands = append(m.Commands, cm.Model())
	}
	return m
}

func (c *CmdClause) Model() *CmdModel {
	depth := 0
	for i := c; i != nil; i = i.parent {
		depth++
	}
	return &CmdModel{
		Name:           c.name,
		Aliases:        c.aliases,
		Help:           c.help,
		HelpLong:       c.helpLong,
		Depth:          depth,
		Hidden:         c.hidden,
		Default:        c.isDefault,
		FullCommand:    c.FullCommand(),
		FlagGroupModel: c.flagGroup.Model(),
		ArgGroupModel:  c.argGroup.Model(),
		CmdGroupModel:  c.cmdGroup.Model(),
	}
}
