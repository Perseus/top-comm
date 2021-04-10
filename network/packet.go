package packet

import (
	"encoding/binary"
	"unsafe"
)

type WPacket struct {
	cmd    int
	header uint32
	size   uint16
	offset uint8
	packet []uint8
}

func (wpk WPacket) GetMaxLength() int {
	return 1024
}

func (wpk *WPacket) SetCommand(cmd int) {
	wpk.cmd = cmd
}

/*
	WriteString appends all the data required to send a string in a packet.

	This data includes (in order)

	1. a uint16 = len(string) + 1 ( the 1 is for the terminating null character at the end of the string)

	2. the entire content of the string in the form of bytes

	After this, we increment the total size of the packet by len(string) + 1 (size of the terminating null character)
*/
func (wpk *WPacket) WriteString(str string) bool {
	stringLength := uint16(len(str) + 1)

	if !wpk.WriteShort(stringLength) {
		return false
	}

	stringInBytes := []byte(str)
	stringInBytes = append(stringInBytes, 0)

	wpk.packet = append(wpk.packet, stringInBytes...)
	wpk.size += stringLength
	return true
}

func (wpk *WPacket) WriteShort(num uint16) bool {
	// if adding the number would make the packet size over our limit, we ignore it
	if unsafe.Sizeof(num)+uintptr(wpk.size) > uintptr(wpk.GetMaxLength()) {
		return false
	}

	data := make([]uint8, 2)
	binary.BigEndian.PutUint16(data, num)

	wpk.packet = append(wpk.packet, data...)
	wpk.size += uint16(unsafe.Sizeof(num))

	return true
}

func (wpk *WPacket) BuildPacket() []uint8 {
	wpk.header = wpk.GetDefaultHeader()

	sizeInBytes := make([]uint8, 2)
	headerInBytes := make([]uint8, 4)
	cmdInBytes := make([]uint8, 2)

	binary.BigEndian.PutUint16(sizeInBytes, wpk.size)
	binary.BigEndian.PutUint32(headerInBytes, wpk.header)
	binary.BigEndian.PutUint16(cmdInBytes, uint16(wpk.cmd))

	packetStart := append(sizeInBytes, headerInBytes...)
	packetStart = append(packetStart, cmdInBytes...)
	wpk.packet = append(packetStart, wpk.packet...)

	return wpk.packet
}

func (wpk WPacket) GetDefaultHeader() uint32 {
	return 2147483648
}

func (wpk WPacket) GetCurrSize() uint16 {
	return wpk.size
}

func CreateWritePacket() WPacket {
	return WPacket{
		size:   8,
		cmd:    0,
		header: 2147483648,
		offset: 0,
		packet: []uint8{},
	}
}

type RPacket struct {
	cmd                                            uint16
	offset, rpos, size, revrpos, tickcount, header uint32
	packet                                         []uint8
}

func CreateReadPacket(data []uint8) RPacket {
	header := binary.BigEndian.Uint32(data[0:4])
	cmd := binary.BigEndian.Uint16(data[4:6])

	// we initialize the offset as 6 to account for the header and cmd being read already
	rpk := RPacket{
		packet: data,
		header: header,
		cmd:    cmd,
		offset: 6,
	}

	return rpk
}

func (rpk RPacket) GetPacket() []uint8 {
	return rpk.packet
}

func (rpk RPacket) GetCommand() uint16 {
	return rpk.cmd
}

func (rpk RPacket) GetRemainingDataLength() int {
	return len(rpk.packet) - int(rpk.offset)
}

func (rpk *RPacket) ReadShort() uint16 {
	var result uint16 = 0
	remainingLength := rpk.GetRemainingDataLength()

	if remainingLength >= 2 {
		result = binary.BigEndian.Uint16(rpk.packet[rpk.offset : rpk.offset+2])
		rpk.offset += 2
	}

	return result
}

func (rpk *RPacket) ReadLong() uint32 {
	var result uint32 = 0
	remainingLength := rpk.GetRemainingDataLength()

	if remainingLength >= 4 {
		result = binary.BigEndian.Uint32(rpk.packet[rpk.offset : rpk.offset+4])
		rpk.offset += 4
	}

	return result
}

func (rpk *RPacket) ReadString() string {
	var result string = ""
	// read the string length
	var stringLength = rpk.ReadShort()

	// if its 0, just return an empty string
	if stringLength == 0 {
		return result
	}

	remainingLength := rpk.GetRemainingDataLength()
	// read the data from the offset to offset + stringLength
	// convert it to a string and then return
	if remainingLength >= int(stringLength) {
		result = string(rpk.packet[rpk.offset : rpk.offset+uint32(stringLength)])
		rpk.offset += uint32(stringLength)
	}

	return result
}
