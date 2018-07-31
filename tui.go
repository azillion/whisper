package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/azillion/whisper/internal/getconfig"
	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
	"github.com/marcusolsson/tui-go"
	"github.com/sirupsen/logrus"
)

type post struct {
	username string
	message  string
	time     string
}

func (p *post) messageToPost(message *discordgo.Message) error {
	s := tuiDS.State
	logrus.Debugf("%v\n", s)
	p.username = message.Author.String()
	content, err := message.ContentWithMoreMentionsReplaced(tuiDS)
	if err != nil {
		return err
	}
	p.message = content
	parsedTime, err := message.Timestamp.Parse()
	if err != nil {
		return err
	}
	p.time = parsedTime.Format("15:04")
	return nil
}

// Channel Channel being displayed by TUI
var (
	channel *discordgo.Channel
	posts   []post
	tuiDS   *discordgo.Session
)

const tuiHelp = `TUI for sending and receiving Discord DMs`

func (cmd *tuiCommand) Name() string      { return "tui" }
func (cmd *tuiCommand) Args() string      { return "[OPTIONS]" }
func (cmd *tuiCommand) ShortHelp() string { return tuiHelp }
func (cmd *tuiCommand) LongHelp() string  { return tuiHelp }
func (cmd *tuiCommand) Hidden() bool      { return false }

func (cmd *tuiCommand) Register(fs *flag.FlagSet) {}

type tuiCommand struct{}

func (cmd *tuiCommand) Run(ctx context.Context, args []string) error {
	StartTUI()
	return nil
}

// StartTUI Start the TUI display
func StartTUI() {
	// sidebar := tui.NewVBox(
	// 	tui.NewLabel("CHANNELS"),
	// 	tui.NewLabel("general"),
	// 	tui.NewLabel("random"),
	// 	tui.NewLabel(""),
	// 	tui.NewLabel("DIRECT MESSAGES"),
	// 	tui.NewLabel("slackbot"),
	// 	tui.NewSpacer(),
	// )
	// sidebar.SetBorder(true)
	tuiDS, err := createDiscordSession(getconfig.AuthConfig{})
	if err != nil {
		logrus.Debugf("Session Failed \n%v\nexiting.", spew.Sdump(tuiDS))
		err = fmt.Errorf("You may need to login from a browser first or check your credentials\n%v", err)
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	// Get a list of DM channels
	channels, err := tuiDS.UserChannels()
	logrus.Debugf("Retrieved Channels\n%v\n", spew.Sdump(channels))
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	// Display available channels
	fmt.Println("Available Private Channels:")
	for i, chann := range channels {
		logrus.Debugf("Channel %d\n%v\n", i, spew.Sdump(chann))

		// Switch of supported channel types
		switch chanType := chann.Type; chanType {
		case 1: // Direct Messages
			var flatRecipients string
			recipients := chann.Recipients
			logrus.Debugf("Channel %d recipients\n%v\n", i, spew.Sdump(recipients))
			for _, recipient := range recipients {
				flatRecipients = fmt.Sprintf("%s %s", flatRecipients, recipient.Username)
			}
			fmt.Printf("\t%d) DM to %s\n", i, strings.TrimSpace(flatRecipients))
		default:
			fmt.Println("No available channels")
			os.Exit(0)
		}
	}

	// Get channel selection
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Select a channel to switch to: ")
	channelSelS, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	channelSel, err := strconv.Atoi(strings.TrimSpace(channelSelS))
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	// Set Channel to selected channel
	channel = channels[channelSel]
	logrus.Debugf("Channel selected %d\n%v\n", channelSel, spew.Sdump(channel))

	// Get Channel messages
	messages, err := tuiDS.ChannelMessages(channel.ID, 100, "", "", "")
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	logrus.Debugf("Channel messages\n%v\n", spew.Sdump(messages))
	// spew.Dump(messages[0])

	p := post{}
	message := messages[0]
	p.username = message.Author.String()
	content, err := message.ContentWithMoreMentionsReplaced(tuiDS)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	p.message = content
	parsedTime, err := message.Timestamp.Parse()
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	p.time = parsedTime.Format("15:04")

	posts = append(posts, p)
	spew.Dump(posts)
	os.Exit(0)
	// Convert messages into tui messages
	// tuiMessages, err := convertToTUIMessages(messages)
	// if err != nil {
	// 	logrus.Debugf("%v\n", err)
	// 	os.Exit(1)
	// }

	history := tui.NewVBox()

	for _, m := range posts {
		history.Append(tui.NewHBox(
			tui.NewLabel(m.time),
			tui.NewPadder(1, 0, tui.NewLabel(fmt.Sprintf("<%s>", m.username))),
			tui.NewLabel(m.message),
			tui.NewSpacer(),
		))
	}

	historyScroll := tui.NewScrollArea(history)
	historyScroll.SetAutoscrollToBottom(true)

	historyBox := tui.NewVBox(historyScroll)
	historyBox.SetBorder(true)

	input := tui.NewEntry()
	input.SetFocused(true)
	input.SetSizePolicy(tui.Expanding, tui.Maximum)

	inputBox := tui.NewHBox(input)
	inputBox.SetBorder(true)
	inputBox.SetSizePolicy(tui.Expanding, tui.Maximum)

	chat := tui.NewVBox(historyBox, inputBox)
	chat.SetSizePolicy(tui.Expanding, tui.Expanding)

	input.OnSubmit(func(e *tui.Entry) {
		history.Append(tui.NewHBox(
			tui.NewLabel(time.Now().Format("15:04")),
			tui.NewPadder(1, 0, tui.NewLabel(fmt.Sprintf("<%s>", "john"))),
			tui.NewLabel(e.Text()),
			tui.NewSpacer(),
		))
		input.SetText("")
	})

	root := tui.NewHBox(chat)

	ui, err := tui.New(root)
	if err != nil {
		logrus.Debugf("%v\n", err)
		os.Exit(1)
	}

	ui.SetKeybinding("Esc", func() { ui.Quit() })

	if err := ui.Run(); err != nil {
		logrus.Debugf("%v\n", err)
		os.Exit(1)
	}
}

// func convertToTUIMessages(messages *[]discordgo.Message) ([]post, error) {
// 	var posts
// }
