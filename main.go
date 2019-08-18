package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
	//	"github.com/webmakersteve/myamtech-bot/queue"
	"github.com/webmakersteve/myamtech-bot/plusthyme"
	"github.com/webmakersteve/myamtech-bot/simulationcraft"
	"strings"
)

// Variables used for command line parameters
var (
	Token             string
	Environment       string
	Simcraft          string
	BnetClientID      string
	BnetClientSecret  string
	dungeoneerRole    string
	loadedDungeoneers bool
)

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.StringVar(&Environment, "e", "production", "Running environment")
	flag.StringVar(&Simcraft, "s", "", "Simcraft path directory")
	flag.StringVar(&BnetClientID, "b", "", "Bnet access id")
	flag.StringVar(&BnetClientSecret, "k", "", "Bnet access token")
	flag.Parse()

	// Overwrite token with an environment variable if it is set
	environmentToken := os.Getenv("BOT_TOKEN")
	if environmentToken != "" {
		Token = environmentToken
	}

	if Simcraft == "" {
		Simcraft = "/usr/bin"
	}

	environmentBnetClientID := os.Getenv("BNET_ACCESS_KEY_ID")
	environmentBnetClientSecret := os.Getenv("BNET_SECRET_ACCESS_KEY")
	if environmentBnetClientID != "" {
		BnetClientID = environmentBnetClientID
	}

	if environmentBnetClientSecret != "" {
		BnetClientSecret = environmentBnetClientSecret
	}

	if Environment == "production" {
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		// The TextFormatter is default, you don't actually have to do this.
		log.SetFormatter(&log.TextFormatter{})
	}

	dungeoneerRole = "612390052911251644"
}

func main() {
	if Token == "" {
		log.Error("No token provided")
		return
	}

	err := simulationcraft.Initialize(Simcraft, BnetClientID, BnetClientSecret)

	if err != nil {
		log.Error("Error initializing simcraft store:" + err.Error())
		return
	}

	discord, err := discordgo.New("Bot " + Token)

	if err != nil {
		log.Error("Error creating discord bot", err)
		return
	}

	discord.AddHandler(messageCreate)

	err = discord.Open()
	if err != nil {
		log.Error("Error creating connection", err)
		return
	}

	defer discord.Close()

	/*
		createdQueue, err := queue.ListenForMessages(discord)
		if err != nil {
			log.Fatal(err)
			return
		}

		defer createdQueue.Close()
	*/
	// Wait here until CTRL-C or other term signal is received.
	log.Info("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

func botIsMentioned(mentions []*discordgo.User, userID string) bool {
	for _, mention := range mentions {
		if mention.ID == userID {
			return true
		}
	}
	return false
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if !loadedDungeoneers {
		loadedDungeoneers = true
		members, err := s.GuildMembers(m.GuildID, "1", 1000)
		if err != nil {
			log.Error(err)
			return
		}
		for _, member := range members {
			for _, role := range member.Roles {
				if role == dungeoneerRole {
					log.Info("Found member " + member.User.Username + " who is a dungeoneer")
					plusthyme.UpdateRegistration(member.User.ID, true)
				}
			}
		}
	}
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	author := m.Author.ID
	if author == s.State.User.ID {
		return
	}

	if strings.Contains(strings.ToLower(m.Content), "takis") {
		emoji := discordgo.Emoji{
			ID:   "601169762092843010",
			Name: "takis",
		}
		err := s.MessageReactionAdd(m.ChannelID, m.ID, emoji.APIName())
		if err != nil {
			log.Error(err)
		}
	}

	var content string
	if botIsMentioned(m.Mentions, s.State.User.ID) {
		content = strings.TrimSpace(strings.Replace(m.ContentWithMentionsReplaced(), "@"+s.State.User.Username, "", -1))
		log.Info("Bot was mentioned in message \"" + content + "\"")
	} else {
		return
	}

	// If the message is "ping" reply with "Pong!"
	if content == "ping" {
		s.ChannelMessageSend(m.ChannelID, "Pong!")
		return
	}

	// If the message is "pong" reply with "Ping!"
	if content == "pong" {
		s.ChannelMessageSend(m.ChannelID, "Ping!")
		return
	}

	if content == "plusthyme" {
		registeredUsers := plusthyme.GetAllRegistered()
		if len(registeredUsers) == 0 {
			s.ChannelMessageSend(m.ChannelID, "No one is registered for plus :(")
			return
		}

		var otherUsersRegistered = false
		for _, user := range registeredUsers {
			if user != m.Author.ID {
				otherUsersRegistered = true
				channel, err := s.UserChannelCreate(user)
				if err != nil {
					log.Error(err)
				} else {
					s.ChannelMessageSend(channel.ID, m.Author.Username+" has requested your presence for PLUSTHYME.")
				}
			}
		}

		if !otherUsersRegistered {
			s.ChannelMessageSend(m.ChannelID, "You're the only one registered for plusthyme :(")
			return
		}
		s.ChannelMessageSend(m.ChannelID, "I have rallied the masses, <@"+m.Author.ID+">.")
		return
	}

	if strings.HasPrefix(content, "register") {
		plusthyme.UpdateRegistration(m.Author.ID, true)
		s.GuildMemberRoleAdd(m.GuildID, m.Author.ID, dungeoneerRole)
		s.ChannelMessageSend(m.ChannelID, "<@"+m.Author.ID+"> OK. I registered you for plusthyme.")
		return
	}

	if strings.HasPrefix(content, "unregister") {
		plusthyme.UpdateRegistration(m.Author.ID, true)
		s.GuildMemberRoleRemove(m.GuildID, m.Author.ID, dungeoneerRole)
		s.ChannelMessageSend(m.ChannelID, "<@"+m.Author.ID+"> OK. I unregistered you for plusthyme.")
		return
	}

	if strings.HasPrefix(content, "simulate ") {
		parts := strings.Split(content, " ")
		simulationcraft.Simulate("wyrmrest-accord", parts[1], s, m)
	}
}
