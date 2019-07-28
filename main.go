package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
	"github.com/webmakersteve/myamtech-bot/queue"
	"github.com/webmakersteve/myamtech-bot/simulationcraft"
	"strings"
)

// Variables used for command line parameters
var (
	Token            string
	Environment      string
	Simcraft         string
	BnetClientID     string
	BnetClientSecret string
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

	createdQueue, err := queue.ListenForMessages(discord)
	if err != nil {
		log.Fatal(err)
		return
	}

	defer createdQueue.Close()

	// Wait here until CTRL-C or other term signal is received.
	log.Info("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}
	// If the message is "ping" reply with "Pong!"
	if m.Content == "ping" {
		s.ChannelMessageSend(m.ChannelID, "Pong!")
	}

	// If the message is "pong" reply with "Ping!"
	if m.Content == "pong" {
		s.ChannelMessageSend(m.ChannelID, "Ping!")
	}

	if strings.HasPrefix(m.Content, "simulate ") {
		trimmed := strings.TrimSpace(m.Content)
		parts := strings.Split(trimmed, " ")
		simulationcraft.Simulate("wyrmrest-accord", parts[1], s, m)
	}
}
