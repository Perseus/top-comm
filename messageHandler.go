package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/bwmarrin/discordgo"
	"github.com/fatih/color"
	"github.com/mitchellh/mapstructure"
	"github.com/perseus/top-comm/config"
	packet "github.com/perseus/top-comm/network"
)

type QueueMessage struct {
	ActionName string
	Payload    interface{}
}

func handleMessages(chn chan types.Message, sqsClient *sqs.Client, tcpPacketChn chan packet.WPacket) {
	queueName := config.INPUT_QUEUE_NAME
	queueUrl, err := sqsClient.GetQueueUrl(context.TODO(), &sqs.GetQueueUrlInput{
		QueueName: &queueName,
	})

	if err != nil {
		color.HiRed("Unable to get QueueURL")
		return
	}

	for val := range chn {
		var result QueueMessage
		parsedString, _ := strconv.Unquote(*val.Body)

		json.Unmarshal([]byte(parsedString), &result)

		if action, ok := config.SupportedActions[result.ActionName]; ok {
			color.HiGreen("[Message] Processing action - %s", result.ActionName)
			go func() {
				handleAction(action.GetPacketId(), result.Payload, tcpPacketChn)

				sqsClient.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
					ReceiptHandle: val.ReceiptHandle,
					QueueUrl:      queueUrl.QueueUrl,
				})

				color.HiGreen("[Message] Action processed - %s", result.ActionName)
			}()
		}

	}
}

func handleAction(actionId uint16, payload interface{}, chn chan packet.WPacket) {
	var wpk = packet.CreateWritePacket()
	wpk.SetCommand(int(actionId))

	switch actionId {
	case 8010: // accept player into guild
		var data config.AcceptPlayerInGuildPayload
		mapstructure.Decode(payload, &data)

		wpk.WriteShort(uint16(data.AccepterCharId))
		wpk.WriteShort(uint16(data.ApplierCharId))
		wpk.WriteShort(uint16(data.GuildId))
	case 8011:
		var data config.RejectPlayerFromGuildPayload
		mapstructure.Decode(payload, &data)

		wpk.WriteShort(uint16(data.RejecterCharId))
		wpk.WriteShort(uint16(data.ApplierCharId))
		wpk.WriteShort(uint16(data.GuildId))
	}

	chn <- wpk
}

func writePacketHandler(chn chan packet.WPacket, conn *net.TCPConn) {
	for val := range chn {
		packetInBytes := val.BuildPacket()
		conn.Write(packetInBytes)
	}
}

func handleCommandsFromGate(rpk packet.RPacket) {
	dg, dgErr := discordgo.New()

	if dgErr != nil {
		fmt.Println("DG err", dgErr)
	}

	if rpk.GetCommand() == 1514 {
		charName := rpk.ReadString()
		chatChannel := rpk.ReadString()
		chatContent := rpk.ReadString()
		webhookId := os.Getenv("playerChatWebhookId")
		webhookToken := os.Getenv("playerChatWebhookToken")

		dg.WebhookExecute(webhookId, webhookToken, false, &discordgo.WebhookParams{
			Content:  chatContent,
			Username: "[" + chatChannel + "] " + charName,
		})

		fmt.Println(charName, chatChannel, chatContent, len(chatContent))
	}
}
