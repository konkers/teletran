package teletran

import (
	"fmt"
	"io"
	"strconv"

	"github.com/bwmarrin/discordgo"
)

type CommandContext struct {
	Session *discordgo.Session
	Message *discordgo.MessageCreate
	Bot     *Bot
	message []byte
}

func (ctx *CommandContext) SendResponse(message string) {
	ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, message)
}

// Implement io.Writer interface
func (ctx *CommandContext) Write(p []byte) (n int, err error) {
	ctx.message = append(ctx.message, p...)
	return len(p), nil
}

var _ io.Writer = (*CommandContext)(nil)

func (ctx *CommandContext) flushMessage() {
	if ctx.message != nil {
		ctx.SendResponse(string(ctx.message))
	}
}

func (ctx *CommandContext) IsAdmin() bool {
	return ctx.Bot.IsAdmin(ctx)
}

type Command func(ctx *CommandContext, args []string)

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

func (c *CommandSet) Exec(ctx *CommandContext, args []string) {

	if len(args) == 0 || args[0] == "help" {
		c.Help(ctx)
		return
	}

	cmd, ok := c.commands[args[0]]

	if !ok {
		ctx.SendResponse(fmt.Sprintf("Command \"%s\" not found", args[0]))
		return
	}

	if cmd.needsAdmin && !ctx.IsAdmin() {
		ctx.SendResponse("Begone pretender!")
		return
	}

	cmd.run(ctx, args[1:])
}

func (c *CommandSet) Help(ctx *CommandContext) {
	ctx.SendResponse(c.HelpMsg(ctx.IsAdmin()))
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
