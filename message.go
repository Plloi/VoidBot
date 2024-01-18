package main

import (
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

type MessasgeEvent struct {
	Message *discordgo.Message
	Session *discordgo.Session
}

func (m MessasgeEvent) OnTimer(t time.Time) {
	messages, err := m.Session.ChannelMessages(m.Message.ChannelID, 100, "", "", "")
	if err != nil {
		log.Printf("failed to featch messages: %v", err)
	}

	for _, message := range messages {
		err := m.Session.ChannelMessageDelete(message.ChannelID, message.ID)
		if err != nil {
			log.Print(err)
		}
	}
}
