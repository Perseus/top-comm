package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"

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

func main() {
	err := godotenv.Load()

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

	color.HiBlue("Connected!")

	conn, err := net.DialTCP("tcp4", nil, tcpAddr)

	if err != nil {
		fmt.Printf("Error - %s \n", err)
		panic("Connection to GateServer failed")
	}

	conn.SetKeepAlive(true)
	defer conn.Close()

	if !authenticate(conn) {
		return
	}

}
