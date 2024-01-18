package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/beevik/timerqueue"
	"github.com/bwmarrin/discordgo"
	"github.com/sdomino/scribble"
)

// Bot parameters
var (
	GuildID        = flag.String("guild", "", "Test guild ID. If not passed - bot registers commands globally")
	BotToken       = flag.String("token", "", "Bot access token")
	RemoveCommands = flag.Bool("rmcmd", true, "Remove all commands after shutdowning or not")
)

var s *discordgo.Session
var settings *scribble.Driver

func init() { flag.Parse() }

func init() {
	var err error
	s, err = discordgo.New("Bot " + *BotToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}
}

var (
	integerOptionMinValue = 1.0

	dmPermission                   = false
	defaultMemberPermissions int64 = discordgo.PermissionManageServer

	commands = []*discordgo.ApplicationCommand{
		{
			Name:                     "settings",
			Description:              "Set Void Settings",
			DefaultMemberPermissions: &defaultMemberPermissions,
			DMPermission:             &dmPermission,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Name:        "active",
					Description: "Eat the messages?",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "seconds",
					Description: "Seconds",
					MinValue:    &integerOptionMinValue,
					MaxValue:    60,
					Required:    false,
				},
			},
		},
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"settings": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// Access options in the order provided by the user.
			options := i.ApplicationCommandData().Options

			// Or convert the slice into a map
			optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
			for _, opt := range options {
				optionMap[opt.Name] = opt
			}

			// Read a fish from the database (passing fish by reference)
			channelSettings := ChannelSettings{}
			if err := settings.Read("channel", i.ChannelID, &channelSettings); err != nil {
				fmt.Println("Error", err)
				channelSettings = NewChannelSettings()
			}

			if opt, ok := optionMap["active"]; ok {
				channelSettings.Active = opt.BoolValue()
			}

			if opt, ok := optionMap["seconds"]; ok {
				channelSettings.Seconds = opt.IntValue()
			}

			if err := settings.Write("channel", i.ChannelID, channelSettings); err != nil {
				fmt.Println("Error", err)
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				// Ignore type for now, they will be discussed in "responses"
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("Channel settings:\nActive: %v\nMessage lifespan: %v seconds", channelSettings.Active, channelSettings.Seconds),
				},
			})
		},
	}
)

func init() {
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		fmt.Printf("Recieved command: %s\n", i.ApplicationCommandData().Name)
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}

func main() {
	var err error

	ticker := time.NewTicker(500 * time.Millisecond)

	// Load Settings
	dir := "./settings"

	settings, err = scribble.New(dir, nil)
	if err != nil {
		log.Println("Error", err)
	}

	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})

	queue := timerqueue.New()
	s.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Read a fish from the database (passing fish by reference)
		channelSettings := ChannelSettings{}
		if err := settings.Read("channel", m.ChannelID, &channelSettings); err != nil {
			fmt.Println("Error", err)
			channelSettings = NewChannelSettings()
		}

		if channelSettings.Active {
			queue.Schedule(&MessasgeEvent{Session: s, Message: m.Message}, m.Timestamp.Add(time.Second*time.Duration(channelSettings.Seconds)))
		}
	})

	err = s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}

	log.Println("Adding commands...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, *GuildID, v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	defer s.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")

	for {
		select {
		case t := <-ticker.C:
			queue.Advance(t)
		case <-stop:
			if *RemoveCommands {

				log.Println("Removing commands...")
				for _, v := range registeredCommands {
					err := s.ApplicationCommandDelete(s.State.User.ID, *GuildID, v.ID)
					if err != nil {
						log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
					}
				}
			}
			log.Println("Gracefully shutting down.")
			return
		}
	}
}
