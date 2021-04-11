package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/fatih/color"
	"github.com/joho/godotenv"
	config "github.com/perseus/top-comm/config"
	packet "github.com/perseus/top-comm/network"
)

var isAuthenticated = false

func authenticate(conn *net.TCPConn) bool {
	color.HiGreen("Authenticating...")

	authPacket := packet.CreateWritePacket()

	username := os.Getenv("comm_username")
	password := os.Getenv("comm_password")
	authPacket.SetCommand(8001)
	authPacket.WriteString(username)
	authPacket.WriteString(password)

	wpk := authPacket.BuildPacket()

	_, err := conn.Write(wpk)

	if err != nil {
		color.HiRed("Error while trying to authenticate - %s", err)
		return false
	}

	recvBuf := make([]byte, 2)

	for !isAuthenticated {
		_, err := conn.Read(recvBuf[:])

		if err != nil {
			fmt.Println(err)
		}

		packetLength := binary.BigEndian.Uint16(recvBuf)
		if packetLength > 2 {
			packetData := make([]byte, packetLength-2)
			_, err := conn.Read(packetData)

			if err != nil {
				fmt.Println(err)
			}

			rpk := packet.CreateReadPacket(packetData)

			if rpk.GetCommand() == config.AUTH_SUCCESS_PACKET {
				isAuthenticated = true

				color.HiGreen("Authentication succeeded!")
				break
			} else if rpk.GetCommand() == config.AUTH_FAIL_PACKET {
				failReason := rpk.ReadString()
				color.HiRed("Authentication failed - %s", failReason)
				return false
			}
		}
	}

	return true
}

func pollForRequests(sqsClient *sqs.Client, chn chan<- types.Message) {
	queueName := config.INPUT_QUEUE_NAME

	inputQueueUrl, err := sqsClient.GetQueueUrl(context.TODO(), &sqs.GetQueueUrlInput{
		QueueName: &queueName,
	})

	if err != nil {
		color.HiRed("Unable to fetch input queue URL %s", err)
	}

	color.Green("Listening on queue %s for messages...", queueName)

	for {
		messageOutputs, err := sqsClient.ReceiveMessage(context.TODO(), &sqs.ReceiveMessageInput{
			QueueUrl:            inputQueueUrl.QueueUrl,
			MaxNumberOfMessages: 10,
		})

		if err != nil {
			log.Fatal(err)
		}

		messages := messageOutputs.Messages

		for _, message := range messages {
			chn <- message
		}
	}
}

func setupAWS() *sqs.Client {
	color.Green("\n[AWS] Setting up AWS Environment")

	cfg, err := awsConfig.LoadDefaultConfig(context.TODO(), awsConfig.WithRegion("eu-west-2"))
	queueName := config.INPUT_QUEUE_NAME

	if err != nil {
		log.Fatal(err)
	}

	sqsClient := sqs.NewFromConfig(cfg)
	color.Green("[AWS] Created SQS Client")

	sqsClient.CreateQueue(context.TODO(), &sqs.CreateQueueInput{
		QueueName: &queueName,
	})

	color.Green("[AWS] Created SQS Queue %s", queueName)

	return sqsClient
}

func main() {
	err := godotenv.Load()
	var wg sync.WaitGroup

	if err != nil {
		log.Fatal("Error loading .env file")
		return
	}

	color.HiBlue("Connecting to GateServer...")

	tcpAddr, err := net.ResolveTCPAddr("tcp4", ":8000")

	if err != nil {
		color.HiRed("TCP Address could not be resolved")
		return
	}

	conn, err := net.DialTCP("tcp4", nil, tcpAddr)

	if err != nil {
		fmt.Printf("Error - %s \n", err)
		panic("Connection to GateServer failed")
	}

	color.HiBlue("Connected!")
	conn.SetKeepAlive(true)
	defer conn.Close()

	if !authenticate(conn) {
		time.Sleep(5 * time.Second)
		return
	}

	go func() {
		recvBuf := make([]byte, 2)

		for {
			_, err := conn.Read(recvBuf[:])

			if err != nil {
				fmt.Println(err)
			}

			packetLength := binary.BigEndian.Uint16(recvBuf)
			if packetLength > 2 {
				color.HiRed("Got packet from GateServer")
				packetData := make([]byte, packetLength-2)
				_, err := conn.Read(packetData)

				if err != nil {
					fmt.Println(err)
				}

				rpk := packet.CreateReadPacket(packetData)

				fmt.Println(rpk)
			}
		}
	}()
	sqsClient := setupAWS()

	if sqsClient == nil {
		color.Red("Unable to create an SQS Client")
		return
	}

	queueMessageChannel := make(chan types.Message, 50)
	tcpPacketChannel := make(chan packet.WPacket, 20)

	wg.Add(1)
	go func(sqsClient *sqs.Client, chn chan<- types.Message) {
		pollForRequests(sqsClient, chn)

		defer wg.Done()
	}(sqsClient, queueMessageChannel)

	wg.Add(1)
	go func() {
		handleMessages(queueMessageChannel, sqsClient, tcpPacketChannel)

		defer wg.Done()
	}()

	wg.Add(1)
	go func(chn chan packet.WPacket, conn *net.TCPConn) {
		writePacketHandler(chn, conn)
		defer wg.Done()
	}(tcpPacketChannel, conn)

	wg.Wait()
}
