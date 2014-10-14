package entry

const (
	DEVICE_TOKEN       = 1 //devicetoken
	PAY_LOAD           = 2 //payload
	NOTIFY_IDENTIYFIER = 3 //notify_identifier
	EXPIRATED_DATE     = 4 //expired date
	PRIORITY           = 5 // priority
)

const (
	CMD_POP      = 2 //正常的Notify命令
	CMD_RESP_ERR = 8 //错误的响应头部
)

const (
	RESP_SUCC                 = 0
	RESP_ERROR                = 1
	RESP_MISS_TOKEN           = 2
	RESP_MISS_TOPIC           = 3
	RESP_MISS_PAYLOAD         = 4
	RESP_INVALID_TOKEN_SIZE   = 5
	RESP_INVALID_TOPIC_SIZE   = 6
	RESP_INVALID_PAYLOAD_SIZE = 7
	RESP_INVALID_TOKEN        = 8
	RESP_SHUTDOWN             = 10
	RESP_NONE                 = 255
)
