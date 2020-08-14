package main

import (
	"flag"
	"fmt"
	"math/rand"
	"moustachos/pkg/config"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Variables used for command line parameters
var (
	dg   *discordgo.Session
	file string
)

func init() {
	flag.StringVar(&file, "f", "", "-f <config file>")
	flag.Parse()
}

func main() {

	if err, conf := config.GetConf(file); err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println(conf.Token)
		// Create a new Discord session using the provided bot token.
		dg, err = discordgo.New("Bot " + conf.Token)
		if err != nil {
			fmt.Println("error creating Discord session,", err)
			return
		}

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
