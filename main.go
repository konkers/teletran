package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/asdine/storm"
	"github.com/bwmarrin/discordgo"
)

var MainCommandSet = NewCommandSet()
var commands = make(map[string]Command)
var db *storm.DB

func isAdmin(user *discordgo.User) bool {
	admin, ok := admins[user.ID]
	return ok && admin
}

func echoCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	s.ChannelMessageSend(m.ChannelID, strings.Join(args, " "))
}

func whoamiCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	var userId string
	if len(args) == 0 {
		userId = m.Author.ID
	} else {
		userId = args[0]
	}

	user, err := GetUser(db, userId)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "error getting user: "+err.Error())

	} else {
		b, _ := json.MarshalIndent(user, "", "  ")
		s.ChannelMessageSend(m.ChannelID, string(b))
	}
}

func adminCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	s.ChannelMessageSend(m.ChannelID, "Welcome admin.")
}

func membersCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// TODO(konkers): Support more than 1000 members.
	channel, err := s.Channel(m.ChannelID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "error fetching channel: "+err.Error())
		return
	}

	members, err := s.GuildMembers(channel.GuildID, "", 1000)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "error guild members: "+err.Error())
		return
	}

	b, _ := json.MarshalIndent(members, "", "  ")
	s.ChannelMessageSend(m.ChannelID, string(b))
}

func creditsCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if !isAdmin(m.Author) {
		s.ChannelMessageSend(m.ChannelID, "Begone pretender!")
		return
	}

	if len(args) != 2 {
		s.ChannelMessageSend(m.ChannelID, "need two args.")
		return
	}

	user, err := GetUser(db, args[0])
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "error getting user: "+err.Error())
		return
	}

	delta, err := strconv.Atoi(args[1])
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "error converting number: "+err.Error())
		return
	}

	user.Credits += delta
	user.Update(db)
}

func init() {
	flag.StringVar(&token, "t", "", "Bot Token")
	flag.Parse()

	MainCommandSet.AddCommand("echo", "echo echo echo echo", echoCommand, false)
	MainCommandSet.AddCommand("whoami", "displays user information.", whoamiCommand, false)
	MainCommandSet.AddCommand("admin", "Asserts admin status.", adminCommand, true)
	MainCommandSet.AddCommand("members", "Display server members.", membersCommand, false)
	MainCommandSet.AddCommand("credits", "Adjust users' credits.", creditsCommand, true)
}

var admins = map[string]bool{
	"337056040703229955": true, // konkers
}

var token string
var buffer = make([][]byte, 0)
var commandPrefix = "!"
var argsRe = regexp.MustCompile(`\s+`)

func main() {

	if token == "" {
		fmt.Println("No token provided. Please run: airhorn -t <bot token>")
		return
	}

	// Load the sound file.
	err := loadSound()
	if err != nil {
		fmt.Println("Error loading sound: ", err)
		fmt.Println("Please copy $GOPATH/src/github.com/bwmarrin/examples/airhorn/airhorn.dca to this directory.")
		return
	}

	db, err = storm.Open("bot.db")
	if err != nil {
		fmt.Println("Can't open db: ", err)
		return
	}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		return
	}

	// Register ready as a callback for the ready events.
	dg.AddHandler(ready)

	// Register messageCreate as a callback for the messageCreate events.
	dg.AddHandler(messageCreate)

	// Register guildCreate as a callback for the guildCreate events.
	dg.AddHandler(guildCreate)

	// Open the websocket and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening Discord session: ", err)
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Airhorn is now running.  Press CTRL-C to exit.")
	fmt.Printf("https://discordapp.com/oauth2/authorize?client_id=%d&scope=bot,&permissions=%d",
		363395462608191500,
		discordgo.PermissionAllText|discordgo.PermissionAllVoice)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// This function will be called (due to AddHandler above) when the bot receives
// the "ready" event from Discord.
func ready(s *discordgo.Session, event *discordgo.Ready) {

	// Set the playing status.
	s.UpdateStatus(0, "!airhorn")
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Find the channel that the message came from.
	c, err := s.State.Channel(m.ChannelID)
	if err != nil {
		// Could not find channel.
		return
	}

	// Find the guild for that channel.
	g, err := s.State.Guild(c.GuildID)
	if err != nil {
		// Could not find guild.
		return
	}

	// check if the message is "!airhorn"
	if strings.HasPrefix(m.Content, "!airhorn") {

		// Look for the message sender in that guild's current voice states.
		for _, vs := range g.VoiceStates {
			if vs.UserID == m.Author.ID {
				err = playSound(s, g.ID, vs.ChannelID)
				if err != nil {
					fmt.Println("Error playing sound:", err)
				}

				return
			}
		}
	} else if strings.HasPrefix(m.Content, commandPrefix) {
		args := argsRe.Split(strings.TrimPrefix(m.Content, commandPrefix), -1)
		fmt.Printf("%q\n", args)
		MainCommandSet.Exec(s, m, args)
	}
}

// This function will be called (due to AddHandler above) every time a new
// guild is joined.
func guildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {

	if event.Guild.Unavailable {
		return
	}

	for _, channel := range event.Guild.Channels {
		if channel.ID == event.Guild.ID {
			_, _ = s.ChannelMessageSend(channel.ID, "Airhorn is ready! Type !airhorn while in a voice channel to play a sound.")
			return
		}
	}
}

// loadSound attempts to load an encoded sound file from disk.
func loadSound() error {

	file, err := os.Open("airhorn.dca")
	if err != nil {
		fmt.Println("Error opening dca file :", err)
		return err
	}

	var opuslen int16

	for {
		// Read opus frame length from dca file.
		err = binary.Read(file, binary.LittleEndian, &opuslen)

		// If this is the end of the file, just return.
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			err := file.Close()
			if err != nil {
				return err
			}
			return nil
		}

		if err != nil {
			fmt.Println("Error reading from dca file :", err)
			return err
		}

		// Read encoded pcm from dca file.
		InBuf := make([]byte, opuslen)
		err = binary.Read(file, binary.LittleEndian, &InBuf)

		// Should not be any end of file errors
		if err != nil {
			fmt.Println("Error reading from dca file :", err)
			return err
		}

		// Append encoded pcm data to the buffer.
		buffer = append(buffer, InBuf)
	}
}

// playSound plays the current buffer to the provided channel.
func playSound(s *discordgo.Session, guildID, channelID string) (err error) {

	// Join the provided voice channel.
	vc, err := s.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return err
	}

	// Sleep for a specified amount of time before playing the sound
	time.Sleep(250 * time.Millisecond)

	// Start speaking.
	vc.Speaking(true)

	// Send the buffer data.
	for _, buff := range buffer {
		vc.OpusSend <- buff
	}

	// Stop speaking
	vc.Speaking(false)

	// Sleep for a specificed amount of time before ending.
	time.Sleep(250 * time.Millisecond)

	// Disconnect from the provided voice channel.
	vc.Disconnect()

	return nil
}
