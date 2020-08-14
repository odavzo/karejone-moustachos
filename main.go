package main

import (
	"flag"
	"fmt"
	"math/rand"
	"moustachos/pkg/config"
	"os"
	"os/signal"

	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Variables used for command line parameters
var (
	dg         *discordgo.Session
	file       string
	playerList map[string]string //id username
)

func init() {
	flag.StringVar(&file, "f", "", "-f <config file>")
	flag.Parse()
}

func main() {

	if err, conf := config.GetConf(file); err != nil {
		fmt.Println(err.Error())
	} else {
		// Create a new Discord session using the provided bot token.
		dg, err = discordgo.New("Bot " + conf.Token)
		if err != nil {
			fmt.Println("Error creating Discord session,", err)
			return
		} else {
			fmt.Println("Bot connected")
		}
		playerList = make(map[string]string)
		// Register ready as a callback for the ready events.
		dg.AddHandler(ready)

		// Register messageCreate as a callback for the messageCreate events.
		dg.AddHandler(messageCreate)

		// Register guildCreate as a callback for the guildCreate events.
		dg.AddHandler(guildCreate)

		// We need information about guilds (which includes their channels),
		// messages and voice states.
		dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates)

		// Open a websocket connection to Discord and begin listening.
		err = dg.Open()
		if err != nil {
			fmt.Println("error opening connection,", err)
			return
		}
		initNoon()
		// Wait here until CTRL-C or other term signal is received.
		fmt.Println("Bot is now running.  Press CTRL-C to exit.")
		sc := make(chan os.Signal, 1)
		signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
		<-sc

		// Cleanly close down the Discord session.
		dg.Close()
	}
}

func initNoon() {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	t := time.Now()
	n := time.Date(t.Year(), t.Month(), t.Day(), 8, 0, r1.Intn(60), 0, t.Location())
	d := n.Sub(t)
	if d < 0 {
		n = n.Add(24 * time.Hour)
		d = n.Sub(t)
	}
	for {
		time.Sleep(d)
		d = 24*time.Hour + time.Duration((r1.Intn(60)-30))*time.Minute
		go sendMoustashos()
	}
}

func sendMoustashos() {
	dg.ChannelMessageSend("709689069017497681", ":man: :man_tone1: :man_tone2: :man_tone3: :man_tone4: :man_tone5: :man_tone4: :man_tone3: :man_tone2: :man_tone1: :man:")
}

// This function will be called (due to AddHandler above) when the bot receives
// the "ready" event from Discord.
func ready(s *discordgo.Session, event *discordgo.Ready) {

	// Set the playing status.
	s.UpdateStatus(0, "!moustachos")
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	// check if the message is "!moustachos"
	if strings.HasPrefix(m.Content, "!moustachos") {
		response := &discordgo.MessageEmbed{
			Color: 0xffcc00,
		}
		command := strings.Split(m.Content, " ")
		//now := time.Now()
		if len(command) > 1 {
			switch command[1] {
			case "bet":
				if len(command) > 2 {

					if _, exist := playerList[m.Author.ID]; exist {
						response.Title = "Le joueur existe déjà dans la liste"
					} else {
						if t, err := time.Parse("15h04", command[2]); err != nil {
							response.Title = "Tu sais pas écrire"
						} else {
							response.Title = "Le joueur " + m.Author.Username + " a été ajouté à la liste et parie sur " + strconv.Itoa(t.Hour()) + ":" + strconv.Itoa(t.Minute()) + " tout ça pour vérifier que l'heure est bien parser."
							playerList[m.Author.ID] = m.Author.Username
						}

					}
				}
			}
		}
		s.ChannelMessageSendEmbed(m.ChannelID, response)

		// Find the channel that the message came from.
		/*c, err := s.State.Channel(m.ChannelID)
		if err != nil {
			// Could not find channel.
			return
		}

		// Find the guild for that channel.
		g, err := s.State.Guild(c.GuildID)
		if err != nil {
			// Could not find guild.
			return
		}*/

		// Look for the message sender in that guild's current voice states.
		//for _, vs := range g.VoiceStates {
		//	if vs.UserID == m.Author.ID {
		//
		//
		//		return
		//	}
		//}
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
			_, _ = s.ChannelMessageSend(channel.ID, "Coucou")
			return
		}
	}
}
