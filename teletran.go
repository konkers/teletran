package teletran

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/asdine/storm"
	"github.com/bwmarrin/discordgo"
)

type Config struct {
	Token         string `json:"token"`
	ClientId      string `json:"client_id"`
	CommandPrefix string `json:"command_prefix"`
}

type Bot struct {
	config   *Config
	commands *CommandSet
	db       *storm.DB
}

func NewBot(config *Config) (*Bot, error) {
	db, err := storm.Open("bot.db")
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		config:   config,
		commands: NewCommandSet(),
		db:       db,
	}

	bot.AddCommand("echo", "echo echo echo echo", bot.echoCommand)
	bot.AddCommand("members", "Display server members.", bot.membersCommand)

	bot.AddAdminCommand("admin", "Asserts admin status.", bot.adminCommand)

	return bot, nil
}

func (b *Bot) GetDbBucket(bucketName string) storm.Node {
	return b.db.From(bucketName)
}

func (b *Bot) IsAdmin(ctx *CommandContext) bool {
	user := ctx.Message.Author
	// TODO(konkers): sort out admin
	admin, ok := admins[user.ID]
	return ok && admin
}

func (b *Bot) AddCommand(name string, help string, command Command) {
	b.commands.AddCommand(name, help, command, false)
}

func (b *Bot) AddAdminCommand(name string, help string, command Command) {
	b.commands.AddCommand(name, help, command, true)
}

func (b *Bot) echoCommand(ctx *CommandContext, args []string) {
	ctx.SendResponse(strings.Join(args, " "))
}

func (b *Bot) adminCommand(ctx *CommandContext, args []string) {
	ctx.SendResponse("Welcome admin.")
}

func (b *Bot) membersCommand(ctx *CommandContext, args []string) {
	// TODO(konkers): Support more than 1000 members.
	channel, err := ctx.Session.Channel(ctx.Message.ChannelID)
	if err != nil {
		ctx.SendResponse("error fetching channel: " + err.Error())
		return
	}

	members, err := ctx.Session.GuildMembers(channel.GuildID, "", 1000)
	if err != nil {
		ctx.SendResponse("error guild members: " + err.Error())
		return
	}

	data, _ := json.MarshalIndent(members, "", "  ")
	ctx.SendResponse(string(data))
}

//func creditsCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
//	if !isAdmin(m.Author) {
//		s.ChannelMessageSend(m.ChannelID, "Begone pretender!")
//		return
//	}
//
//	if len(args) != 2 {
//		s.ChannelMessageSend(m.ChannelID, "need two args.")
//		return
//	}
//
//	user, err := GetUser(db, args[0])
//	if err != nil {
//		s.ChannelMessageSend(m.ChannelID, "error getting user: "+err.Error())
//		return
//	}
//
//	delta, err := strconv.Atoi(args[1])
//	if err != nil {
//		s.ChannelMessageSend(m.ChannelID, "error converting number: "+err.Error())
//		return
//	}
//
//	user.Credits += delta
//	user.Update(db)
//}

var admins = map[string]bool{
	"337056040703229955": true, // konkers
}

var argsRe = regexp.MustCompile(`\s+`)

func (bot *Bot) Run() {

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + bot.config.Token)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		return
	}

	// Register ready as a callback for the ready events.
	dg.AddHandler(bot.ready)

	// Register messageCreate as a callback for the messageCreate events.
	dg.AddHandler(bot.messageCreate)

	// Register guildCreate as a callback for the guildCreate events.
	dg.AddHandler(bot.guildCreate)

	// Open the websocket and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening Discord session: ", err)
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Teletran1 is now running.  Press CTRL-C to exit.")
	fmt.Printf("https://discordapp.com/oauth2/authorize?client_id=%s&scope=bot&permissions=%d\n",
		bot.config.ClientId,
		discordgo.PermissionAllText|discordgo.PermissionAllVoice)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// This function will be called (due to AddHandler above) when the bot receives
// the "ready" event from Discord.
func (bot *Bot) ready(s *discordgo.Session, event *discordgo.Ready) {

	// Set the playing status.
	s.UpdateStatus(0, bot.config.CommandPrefix+"help")
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func (bot *Bot) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, bot.config.CommandPrefix) {
		args := argsRe.Split(strings.TrimPrefix(m.Content, bot.config.CommandPrefix), -1)
		fmt.Printf("%q\n", args)
		ctx := &CommandContext{
			Session: s,
			Message: m,
			Bot:     bot,
		}
		bot.commands.Exec(ctx, args)
	}
}

// This function will be called (due to AddHandler above) every time a new
// guild is joined.
func (bot *Bot) guildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
	if event.Guild.Unavailable {
		return
	}

	for _, channel := range event.Guild.Channels {
		if channel.ID == event.Guild.ID {
			_, _ = s.ChannelMessageSend(channel.ID, "Teletran1 here awaiting orders.")
			return
		}
	}
}
