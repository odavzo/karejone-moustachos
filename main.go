package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand"
	"moustachos/config"
	"moustachos/db"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

type time_moustachos struct {
	next_period_msg time.Duration
	next_time_msg   time.Time
	next_time_min   time.Time
	next_time_max   time.Time
}

type role_moustachos struct {
	mdj *discordgo.Role
	idj *discordgo.Role
}

type classement_item_t struct {
	player_id string
	delta     uint64
}

type classement_t []classement_item_t

// Variables used for command line parameters
var (
	dg            *discordgo.Session
	file          string
	playerListBet map[string]time.Time //id time
	rd            *rand.Rand
	conf          config.Config
	// Init struct
	tm = &time_moustachos{
		next_period_msg: 0,
		next_time_msg:   time.Now(),
		next_time_min:   time.Now(),
		next_time_max:   time.Now(),
	}
	rm = &role_moustachos{
		mdj: nil,
		idj: nil,
	}
)

func (p classement_t) Len() int {
	return len(p)
}

func (p classement_t) Less(i, j int) bool {
	return p[i].delta < p[j].delta
}

func (p classement_t) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func center(s string, w int) string {
	return fmt.Sprintf("%[1]*s", -w, fmt.Sprintf("%[1]*s", (w+len(s))/2, s))
}

func init() {
	flag.StringVar(&file, "f", "", "-f <config file>")
	flag.Parse()
	rd = rand.New(rand.NewSource(time.Now().UnixNano()))
	t_now := time.Now()
	tm.next_time_msg = time.Date(t_now.Year(), t_now.Month(), t_now.Day()+1, 8, rd.Intn(60), 0, 0, t_now.Location())
	tm.next_time_min = time.Date(t_now.Year(), t_now.Month(), t_now.Day()+1, 8, 0, 0, 0, t_now.Location())
	tm.next_time_max = time.Date(t_now.Year(), t_now.Month(), t_now.Day()+1, 8, 59, 0, 0, t_now.Location())
	tm.next_period_msg = tm.next_time_msg.Sub(t_now)
}

func main() {
	var err error
	if err, conf = config.GetConf(file); err != nil {
		fmt.Println(err.Error())
	} else {
		playerListBet = make(map[string]time.Time)
		// Create a new Discord session using the provided bot token.
		dg, err = discordgo.New("Bot " + conf.Token)
		if err = db.Init(); err != nil {
			panic(err)
		}
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
		if rm.mdj, err = dg.GuildRoleCreate(conf.MoustachosGuildId); err != nil {
			panic(err)
		} else {
			rm.mdj.Name = "Moustachos du jour"
			rm.mdj.Color = 0x32a89e
			rm.mdj.Hoist = true
			dg.GuildRoleEdit(conf.MoustachosGuildId, rm.mdj.ID, rm.mdj.Name, rm.mdj.Color, rm.mdj.Hoist, rm.mdj.Permissions, rm.mdj.Mentionable)
		}
		if rm.idj, err = dg.GuildRoleCreate(conf.MoustachosGuildId); err != nil {
			panic(err)
		} else {
			rm.idj.Name = "Imberbe du jour"
			rm.idj.Color = 0xc7ae6f
			dg.GuildRoleEdit(conf.MoustachosGuildId, rm.idj.ID, rm.idj.Name, rm.idj.Color, rm.idj.Hoist, rm.idj.Permissions, rm.idj.Mentionable)
		}
		go printNextMessageEstimation()
		go goMoustachos()

		// Wait here until CTRL-C or other term signal is received.
		fmt.Println("Bot is now running.  Press CTRL-C to exit.")
		sc := make(chan os.Signal, 1)
		signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
		<-sc
		dg.ChannelMessageSendEmbed(conf.MoustachosChannelId, &discordgo.MessageEmbed{
			Title: "Moustachos - Bingo Down",
		})
		dg.GuildRoleDelete(conf.MoustachosGuildId, rm.mdj.ID)
		dg.GuildRoleDelete(conf.MoustachosGuildId, rm.idj.ID)

		// Cleanly close down the Discord session.
		dg.Close()
	}
}

func printNextMessageEstimation() {
	response := &discordgo.MessageEmbed{
		Title: "Moustachos",
	}
	response.Description = fmt.Sprintf("TEST DE MERDE\nDemain, prochain message entre %s et %s\n",
		tm.next_time_min.Format("15h04"), tm.next_time_max.Format("15h04"))

	dg.ChannelMessageSendEmbed(conf.MoustachosChannelId, response)
}

func goMoustachos() {
	for {
		fmt.Println("Next Moustachos Message in " + tm.next_period_msg.String())
		time.Sleep(tm.next_period_msg)
		tm.next_period_msg = 24*time.Hour + time.Duration((rd.Intn(60)-30))*time.Minute
		t_now := time.Now()
		response := &discordgo.MessageEmbed{
			Title: "Moustachos",
		}
		classement := make(classement_t, len(playerListBet))
		i := 0
		for key, value := range playerListBet {
			t_player := time.Date(t_now.Year(), t_now.Month(), t_now.Day(), value.Hour(), value.Minute(), 0, 0, t_now.Location())
			d_delta_player := t_now.Sub(t_player)
			uint64_delta_player_min := uint64(math.Abs(d_delta_player.Minutes()))
			e := classement_item_t{
				player_id: key,
				delta:     uint64_delta_player_min,
			}
			classement[i] = e
			i++
		}
		// Tri
		sort.Sort(classement)
		response.Description += ":man: :man_tone1: :man_tone2: :man_tone3: :man_tone4: :man_tone5: :man_tone4: :man_tone3: :man_tone2: :man_tone1: :man: :man: :man_tone1: :man_tone2:\n"
		response.Description += fmt.Sprintf("```" +
			"┌───────────────────────────────────┐\n" +
			"│" + center("Classement du jour", 35) + "│\n" +
			"├───┬───────────────┬───────────────┤\n" +
			"│" + center("#", 3) + "│" + center("Personnage", 15) + "│" + center("Δ (en min)", 15) + "│\n" +
			"├───┼───────────────┼───────────────┤\n")
		for i, value := range classement {
			u, _ := dg.User(value.player_id)
			response.Description += fmt.Sprintln("│" + center(strconv.FormatInt(int64(i), 10), 3) + "│" + center(u.Username, 15) + "│" + center(strconv.FormatUint(value.delta, 10), 15) + "│")
		}
		response.Description += fmt.Sprintf("└───┴───────────────┴───────────────┘\n")
		response.Description += "```"
		response.Description += "\n:man: :man_tone1: :man_tone2: :man_tone3: :man_tone4: :man_tone5: :man_tone4: :man_tone3: :man_tone2: :man_tone1: :man: :man: :man_tone1: :man_tone2:\n\n"

		dg.GuildMemberRoleAdd(conf.MoustachosGuildId, classement[0].player_id, rm.mdj.ID)
		dg.GuildMemberRoleAdd(conf.MoustachosGuildId, classement[len(classement)-1].player_id, rm.idj.ID)
		//dg.ChannelMessageSendEmbed(conf.MoustachosChannelId, response)
		response.Description += center("Le <@&"+rm.mdj.ID+"> est <@"+classement[0].player_id+">\n", 36) +
			"Le <@&" + rm.idj.ID + "> est <@" + classement[len(classement)-1].player_id + ">\n\n"
		response.Description += ":man: :man_tone1: :man_tone2: :man_tone3: :man_tone4: :man_tone5: :man_tone4: :man_tone3: :man_tone2: :man_tone1: :man: :man: :man_tone1: :man_tone2:"
		dg.ChannelMessageSendEmbed(conf.MoustachosChannelId, response)
	}
}

// This function will be called (due to AddHandler above) when the bot receives
// the "ready" event from Discord.
func ready(s *discordgo.Session, event *discordgo.Ready) {
	str := list()
	response := &discordgo.MessageEmbed{
		Title:       "Moustachos - Bingo Up",
		Description: str,
	}
	dg.ChannelMessageSendEmbed(conf.MoustachosChannelId, response)
	// Set the playing status.
	s.UpdateStatus(0, "!moustachos")
}

func list() (str string) {
	str += fmt.Sprintf("```\n" +
		"┌───────────────────────────────────┐\n" +
		"│" + center("Dernière valeur", 35) + "│\n" +
		"├───────────────────┬───────────────┤\n" +
		"│" + center("Username", 19) + "│" + center("Heure", 15) + "│\n")
	for key, _ := range playerListBet {
		u, _ := dg.User(key)
		str += fmt.Sprintln("│" + center(u.Username, 19) + "│" + center(playerListBet[key].Format("15h04"), 15) + "│")
	}
	str += fmt.Sprintf("└───────────────────┴───────────────┘\n")
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
	if m.ChannelID == conf.MoustachosChannelId {
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
