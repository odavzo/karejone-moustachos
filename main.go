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

type list_moustachos struct {
	list_player_next_day map[string]time.Time
	list_player_curr_day map[string]time.Time
}

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

const moustachos_str_emote = ":man: :man_tone1: :man_tone2: :man_tone3:" +
	":man_tone4: :man_tone5: :man_tone4: :man_tone3: :man_tone2: :man_tone1:" +
	":man: :man: :man_tone1: :man_tone2:"

type classement_item_t struct {
	player_id string
	delta     uint64
}

type classement_t []classement_item_t

// Variables used for command line parameters
var (
	dg   *discordgo.Session
	file string
	rd   *rand.Rand
	conf config.Config
	// Init struct
	lm = &list_moustachos{
		list_player_next_day: make(map[string]time.Time),
		list_player_curr_day: make(map[string]time.Time),
	}
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
	tm.next_time_msg = time.Date(t_now.Year(), t_now.Month(), t_now.Day()+1, 8, rd.Intn(60), 1, 0, t_now.Location())
	tm.next_time_min = time.Date(t_now.Year(), t_now.Month(), t_now.Day()+1, 8, 0, 0, 0, t_now.Location())
	tm.next_time_max = time.Date(t_now.Year(), t_now.Month(), t_now.Day()+1, 8, 59, 0, 0, t_now.Location())
	tm.next_period_msg = tm.next_time_msg.Sub(t_now)
}

func main() {
	var err error
	if err, conf = config.GetConf(file); err != nil {
		fmt.Println(err.Error())
	} else {

		// Create a new Discord session using the provided bot token.
		dg, err = discordgo.New("Bot " + conf.Token)
		if err = db.Init(); err != nil {
			panic(err)
		}
		db.GetAllData(lm.list_player_next_day)
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
		go manageVoting()

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
	response.Description = fmt.Sprintf("Demain, prochain message\n")
	if tm.next_time_min.After(tm.next_time_max) {
		response.Description += "entre " + tm.next_time_min.Format("15h04") + " et " + "23h59 et entre 00h00 et " + tm.next_time_max.Format("15h04")
	} else {
		response.Description += "entre " + tm.next_time_min.Format("15h04") + " et " + tm.next_time_max.Format("15h04")
	}

	dg.ChannelMessageSendEmbed(conf.MoustachosChannelId, response)
}

func manageVoting() {
	for {
		t_yesterday := time.Date(tm.next_time_msg.Year(), tm.next_time_msg.Month(), tm.next_time_msg.Day(), 0, 0, 1, 0, tm.next_time_msg.Location())
		time.Until(t_yesterday)
		for k, v := range lm.list_player_next_day {
			lm.list_player_curr_day[k] = v
			delete(lm.list_player_next_day, k)
		}
	}

}

func goMoustachos() {
	for {
		fmt.Println("Next Moustachos Message in " + tm.next_period_msg.String())
		time.Sleep(tm.next_period_msg)
		t_now := time.Now()
		delta_min := rd.Intn(60) - 30
		t_previous := tm.next_time_msg
		tm.next_period_msg = 24*time.Hour + time.Duration(delta_min)*time.Minute
		tm.next_time_msg = t_now.Add(tm.next_period_msg)
		tm.next_time_min = t_previous.Add(-30 * time.Minute)
		tm.next_time_max = t_previous.Add(29 * time.Minute)
		if tm.next_time_min.Day() != t_now.Day() {
			tm.next_time_min.Add(24 * time.Hour)
		}
		if tm.next_time_max.Day() != t_now.Day() {
			tm.next_time_min.Add(-24 * time.Hour)
		}
		fmt.Println(tm.next_time_min)
		fmt.Println(tm.next_time_max)
		response := &discordgo.MessageEmbed{
			Title: "Moustachos",
		}
		if len(lm.list_player_curr_day) > 2 {
			classement := make(classement_t, len(lm.list_player_curr_day))
			i := 0
			for key, value := range lm.list_player_curr_day {
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
			response.Description += moustachos_str_emote + "\n"
			response.Description += fmt.Sprintf("```" +
				"┌───────────────────────────────────┐\n" +
				"│" + center("Classement du jour", 35) + "│\n" +
				"├───┬───────────────┬───────────────┤\n" +
				"│" + center("#", 3) + "│" + center("Personnage", 15) + "│" + center("Δ (en min)", 15) + "│\n" +
				"├───┼───────────────┼───────────────┤\n")
			for i, value := range classement {
				u, _ := dg.User(value.player_id)
				response.Description += fmt.Sprintln("│" + center(strconv.FormatInt(int64(i), 10), 3) + "│" +
					center(u.Username, 15) + "│" + center(strconv.FormatUint(value.delta, 10), 15) + "│")
			}
			response.Description += fmt.Sprintf("└───┴───────────────┴───────────────┘\n")
			response.Description += "```\n"
			response.Description += moustachos_str_emote + "\n\n"

			dg.GuildMemberRoleAdd(conf.MoustachosGuildId, classement[0].player_id, rm.mdj.ID)
			dg.GuildMemberRoleAdd(conf.MoustachosGuildId, classement[len(classement)-1].player_id, rm.idj.ID)
			response.Description += center("Le <@&"+rm.mdj.ID+"> est <@"+classement[0].player_id+">\n", 36) +
				"L' <@&" + rm.idj.ID + "> est <@" + classement[len(classement)-1].player_id + ">\n\n"
			response.Description += moustachos_str_emote
		} else {
			response.Description += moustachos_str_emote + "\n\n"
			response.Description += "Pas assez de joueur ont joué hier, il faut au moins deux joueurs !!\n\n"
			response.Description += moustachos_str_emote
		}
		dg.ChannelMessageSendEmbed(conf.MoustachosChannelId, response)
		go printNextMessageEstimation()
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
	for key, _ := range lm.list_player_next_day {
		u, _ := dg.User(key)
		str += fmt.Sprintln("│" + center(u.Username, 19) + "│" + center(lm.list_player_next_day[key].Format("15h04"), 15) + "│")
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
			if len(command) > 1 {
				switch command[1] {
				case "bet":
					if len(command) > 2 {
						if t, err := time.Parse("15h04", command[2]); err != nil {
							response.Description = "Tu sais pas écrire"
						} else {
							if _, exist := lm.list_player_next_day[m.Author.ID]; exist {
								response.Description = "Le joueur " + m.Author.Username + " a changé son parie pour " + strconv.Itoa(t.Hour()) + ":" + strconv.Itoa(t.Minute())
							} else {
								response.Description = "Le joueur " + m.Author.Username + " a parié sur " + strconv.Itoa(t.Hour()) + ":" + strconv.Itoa(t.Minute())
							}
							db.SaveData(m.Author.ID, t)
							lm.list_player_next_day[m.Author.ID] = t
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
