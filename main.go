package main

import (
	"html"
	"log"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/VIZ-Blockchain/viz-go-lib"
	"github.com/VIZ-Blockchain/viz-go-lib/operations"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	_ "github.com/joho/godotenv/autoload"
)

func main() {

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	err = start(bot)
	if err != nil {
		log.Panic(err)
	}
}

func start(bot *tgbotapi.BotAPI) error {
	cls, _ := viz.NewClient(os.Getenv("VIZ_NODE"))
	defer cls.Close()

	config, err := cls.API.GetConfig()
	if err != nil {
		return err
	}

	props, err := cls.API.GetDynamicGlobalProperties()
	if err != nil {
		return err
	}
	lastBlock := props.LastIrreversibleBlockNum

	log.Printf("---> Entering the block processing loop (last block = %v)\n", lastBlock)
	for {
		props, err := cls.API.GetDynamicGlobalProperties()
		if err != nil {
			return err
		}

		for props.LastIrreversibleBlockNum-lastBlock > 0 {
			block, err := cls.API.GetBlock(lastBlock)
			if err != nil {
				return err
			}
			log.Printf("Received block %v", block.Number)

			for _, tx := range block.Transactions {
				for _, operation := range tx.Operations {
					switch op := operation.Data().(type) {
					case *operations.AwardOperation:
						channel := getChannel(op.Memo)
						if channel != "" {
							err = saveChannel(bot, channel)
							if err != nil {
								log.Println(err)
							}
						}
						// case *operations.UnknownOperation:
						// 	log.Printf("Unkowned operation receivded: %v+\n", op)
					}
				}
			}

			lastBlock++
		}

		time.Sleep(time.Duration(config.BlockInterval) * time.Second)
	}
}

func getChannel(str string) string {
	var re = regexp.MustCompile(`(?m)channel:(@[a-z0-9_]+):`)
	var arr = re.FindStringSubmatch(str)
	if len(arr) > 1 {
		return arr[1]
	}
	return ""
}

func saveChannel(bot *tgbotapi.BotAPI, channel string) error {

	c, err := bot.GetChat(tgbotapi.ChatConfig{SuperGroupUsername: channel})
	if err != nil {
		return err
	}

	var description = ""
	if strings.TrimSpace(c.Description) != "" {
		description = "\n\n *** \n\n" + c.Description + "\n\n ***"
	}
	text := randomEmoji() + " Новый #канал с ботом: \n\n" + c.Title + " — @" + c.UserName + description
	msg := tgbotapi.NewMessageToChannel(os.Getenv("TELEGRAM_CHANNEL"), text)
	_, err = bot.Send(msg)
	return err
}

func randomEmoji() string {
	rand.Seed(time.Now().UnixNano())
	// http://apps.timwhitlock.info/emoji/tables/unicode
	emoji := [][]int{
		// Emoticons icons
		{128513, 128591},
		// Transport and map symbols
		{128640, 128704},
	}
	r := emoji[rand.Int()%len(emoji)]
	min := r[0]
	max := r[1]
	n := rand.Intn(max-min+1) + min
	return html.UnescapeString("&#" + strconv.Itoa(n) + ";")
}
