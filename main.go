package main

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/viz-cx/viz-news-bot/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var collection *mongo.Collection
var ctx = context.TODO()
var telegramUser int64

var wait = 0

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("panic occurred:", err)
			wait = wait + 1
			time.Sleep(time.Duration(wait) * time.Second)
			main()
		}
	}()

	telegramUser, _ = strconv.ParseInt(os.Getenv("TELEGRAM_RECEIVER_ID"), 10, 64)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO")))
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

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

	lastBlock, err := models.GetLastBlock()
	if err != nil {
		log.Println(err)
	}
	if lastBlock == 0 {
		props, err := cls.API.GetDynamicGlobalProperties()
		if err != nil {
			log.Println(err)
		}
		lastBlock = props.LastIrreversibleBlockNum
	}

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
						channel, err := parseChannel(bot, op.Memo)
						if err != nil {
							continue
						}
						if channel != nil {
							_, err := models.GetTelegramChannel(channel.ID)
							if err == nil { // already exists
								continue
							}
							err = models.SaveTelegramChannel(
								models.TelegramChannel{
									ID:          primitive.NewObjectID(),
									ChannelID:   channel.ID,
									Title:       channel.Title,
									UserName:    channel.UserName,
									Description: channel.Description,
									CreatedAt:   time.Now(),
									BlockID:     block.Number,
								})
							if err != nil {
								log.Println(err)
							}
							err = sendMessageWithChannel(bot, channel)
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

func parseChannel(bot *tgbotapi.BotAPI, str string) (*tgbotapi.Chat, error) {
	var re = regexp.MustCompile(`(?m)channel:(@[a-z0-9_]+):`)
	var arr = re.FindStringSubmatch(str)
	if len(arr) == 0 {
		return nil, errors.New("Not a channel")
	}
	c, err := bot.GetChat(tgbotapi.ChatConfig{SuperGroupUsername: arr[1]})
	if err != nil {
		return nil, err
	}
	if c.Type != "channel" {
		return nil, errors.New(fmt.Sprintf("Not a channel: @%s", c.UserName))
	}
	return &c, nil
}

func sendMessageWithChannel(bot *tgbotapi.BotAPI, c *tgbotapi.Chat) error {
	var description = ""
	if strings.TrimSpace(c.Description) != "" {
		description = "\n\n *** \n\n" + c.Description + "\n\n ***"
	}
	text := randomEmoji() + " Новый #канал с ботом: \n\n" + c.Title + " — @" + c.UserName + description
	msg := tgbotapi.NewMessage(telegramUser, text)
	_, err := bot.Send(msg)
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
