package main

import (
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"
)

type Command func(s *discordgo.Session, m *discordgo.MessageCreate, args []string)

type commandInfo struct {
	run        Command
	needsAdmin bool
	help       string
}

type CommandSet struct {
	commands map[string]*commandInfo
}

func NewCommandSet() *CommandSet {
	return &CommandSet{
		commands: make(map[string]*commandInfo),
	}
}

func (c *CommandSet) AddCommand(name string, help string,
	run Command, needsAdmin bool) {
	_, ok := c.commands[name]
	if ok {
		fmt.Printf("Command %s already registered.\n", name)
		return
	}

	c.commands[name] = &commandInfo{
		run:        run,
		needsAdmin: needsAdmin,
		help:       help,
	}
}

func (c *CommandSet) Exec(s *discordgo.Session,
	m *discordgo.MessageCreate, args []string) {

	if len(args) == 0 {
		s.ChannelMessageSend(m.ChannelID, "TODO: add help")
		return
	}

	if args[0] == "help" {
		s.ChannelMessageSend(m.ChannelID, c.HelpMsg(isAdmin(m.Author)))
		return
	}

	cmd, ok := c.commands[args[0]]

	if !ok {
		s.ChannelMessageSend(m.ChannelID,
			fmt.Sprintf("Command \"%s\" not found", args[0]))
		return
	}

	if cmd.needsAdmin && !isAdmin(m.Author) {
		s.ChannelMessageSend(m.ChannelID, "Begone pretender!")
		return
	}

	cmd.run(s, m, args[1:])
}

func (c *CommandSet) HelpMsg(isAdmin bool) string {
	msg := "```commands:\n"

	maxCommndLen := 0

	for name, _ := range c.commands {
		if len(name) > maxCommndLen {
			maxCommndLen = len(name)
		}
	}

	fmtString := "\t%-" + strconv.FormatInt(int64(maxCommndLen), 10) + "s  %s\n"

	for name, cmd := range c.commands {
		if cmd.needsAdmin && !isAdmin {
			continue
		}
		msg += fmt.Sprintf(fmtString, name, cmd.help)
	}

	msg += "```"
	return msg
}
