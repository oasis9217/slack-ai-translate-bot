package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	openai "github.com/sashabaranov/go-openai"
	slacker "github.com/shomali11/slacker"
	slack "github.com/slack-go/slack"
)

func printCommandEvents(analyticsChannel <-chan *slacker.CommandEvent) {
	for event := range analyticsChannel {
		fmt.Println("[Command events]")
		fmt.Println(event.Timestamp)
		fmt.Println(event.Command)
		fmt.Println(event.Parameters)
		fmt.Println(event.Event)
		fmt.Println()
	}
}

func getAnswerFromChatGPT(aiClient *openai.Client, inputString string, answer chan string) {
	query := ""
	words := strings.Split(inputString, " ")

	if len(words) > 2 {
		query = fmt.Sprintf("영어 '%s'의 뜻이 뭐야?", inputString)
	} else {
		query = fmt.Sprintf("영어 단어 '%s'의 뜻이 뭐야? 그리고 비슷한 영단어도 3개 알려줘", inputString)
	}

	log.Print("Asking ", query)

	resp, err := aiClient.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: query,
				},
			},
		},
	)

	if err != nil {
		log.Fatal(err)
	}

	answer <- resp.Choices[0].Message.Content
}

func main() {
	godotenv.Load()

	bot := slacker.NewClient(os.Getenv("SLACK_BOT_TOKEN"), os.Getenv("SLACK_APP_TOKEN"))
	aiClient := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	aiReponseChannel := make(chan string)

	go printCommandEvents(bot.CommandEvents())

	bot.Command("translate <inputString>", &slacker.CommandDefinition{
		Description: "translate your word or sentence in Korean",
		Examples:    []string{"/translate word", "/translate sentence"},
		Handler: func(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
			inputString := request.Param("inputString")
			go getAnswerFromChatGPT(aiClient, inputString, aiReponseChannel)

			question := fmt.Sprintf("Translation for %s (`/translate %s`)", inputString, inputString)
			answer := <-aiReponseChannel

			attachments := []slack.Block{}
			attachments = append(attachments, slack.NewContextBlock(
				"1",
				slack.NewTextBlockObject("mrkdwn", question, false, false),
			))
			attachments = append(attachments, slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", answer, false, false),
				nil,
				nil,
			))

			response.Reply("", slacker.WithBlocks(attachments))
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := bot.Listen(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
