package main

import (
	"flag"
	"fmt"
	"math/rand"
	"moustachos/pkg/config"
	"moustachos/pkg/db"
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
	dg            *discordgo.Session
	file          string
	playerListBet map[string]time.Time //id time

)

func init() {
	flag.StringVar(&file, "f", "", "-f <config file>")
	flag.Parse()
}

func main() {

	if err, conf := config.GetConf(file); err != nil {
		fmt.Println(err.Error())
	} else {
		playerListBet = make(map[string]time.Time)
		// Create a new Discord session using the provided bot token.
		dg, err = discordgo.New("Bot " + conf.Token)
		db.Init()
		db.GetAllData(playerListBet)
		if err != nil {
			fmt.Println("Error creating Discord session,", err)
			return
		} else {
			fmt.Println("Bot connected")
		}

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
		go initNoon()

		// Wait here until CTRL-C or other term signal is received.
		fmt.Println("Bot is now running.  Press CTRL-C to exit.")
		sc := make(chan os.Signal, 1)
		signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
		<-sc
		response := &discordgo.MessageEmbed{
			Title: "Moustachos - Bingo Down",
		}
		dg.ChannelMessageSendEmbed("716988290355691621", response)

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

func center(s string, w int) string {
	return fmt.Sprintf("%[1]*s", -w, fmt.Sprintf("%[1]*s", (w+len(s))/2, s))
}

// This function will be called (due to AddHandler above) when the bot receives
// the "ready" event from Discord.
func ready(s *discordgo.Session, event *discordgo.Ready) {
	str := list()
	response := &discordgo.MessageEmbed{
		Title:       "Moustachos - Bingo Up",
		Description: str,
	}
	dg.ChannelMessageSendEmbed("716988290355691621", response)
	// Set the playing status.
	s.UpdateStatus(0, "!moustachos")
}

func list() (str string) {
	str = "```\n"
	str += fmt.Sprintln("|" + center("Username", 20) + "|" + center("Heure", 20) + "|")
	str += fmt.Sprintf("|--------------------|--------------------|\n")
	for key, _ := range playerListBet {
		u, _ := dg.User(key)
		str += fmt.Sprintln("|" + center(u.Username, 20) + "|" + center(playerListBet[key].Format("15h04"), 20) + "|")
	}
	str += "```\n"
	return str
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
			Title: "Moustachos - Bingo",
		}
		command := strings.Split(m.Content, " ")
		//now := time.Now()
		if len(command) > 1 {
			switch command[1] {
			case "bet":
				if len(command) > 2 {
					if t, err := time.Parse("15h04", command[2]); err != nil {
						response.Description = "Tu sais pas écrire"
					} else {
						if _, exist := playerListBet[m.Author.ID]; exist {
							response.Description = "Le joueur " + m.Author.Username + " a changé son parie pour " + strconv.Itoa(t.Hour()) + ":" + strconv.Itoa(t.Minute())
						} else {
							response.Description = "Le joueur " + m.Author.Username + " a parié sur " + strconv.Itoa(t.Hour()) + ":" + strconv.Itoa(t.Minute())
						}
						db.SaveData(m.Author.ID, t)
						playerListBet[m.Author.ID] = t
					}
				}

			case "list":
				response.Description = list()
			}

		}
		s.ChannelMessageSendEmbed(m.ChannelID, response)
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
