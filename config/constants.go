package config

type SupportedAction struct {
	name     string
	packetId uint16
}

func (action SupportedAction) GetPacketId() uint16 {
	return action.packetId
}

var (
	SupportedActionNames = []SupportedAction{
		{
			name:     "AcceptPlayerInGuild",
			packetId: 8010,
		},
		{
			name:     "RejectPlayerFromGuild",
			packetId: 8011,
		},
	}
)

func GetSupportedActions() map[string]SupportedAction {
	supportedActions := make(map[string]SupportedAction)

	for _, val := range SupportedActionNames {
		supportedActions[val.name] = val
	}

	return supportedActions
}

const (
	AUTH_SUCCESS_PACKET = 8501
	AUTH_FAIL_PACKET    = 8502
	INPUT_QUEUE_NAME    = "comm-module-input-q"
)

var (
	SupportedActions = GetSupportedActions()
)

type AcceptPlayerInGuildPayload struct {
	AccepterCharId int
	ApplierCharId  int
	GuildId        int
}

type RejectPlayerFromGuildPayload struct {
	RejecterCharId int
	ApplierCharId  int
	GuildId        int
}
