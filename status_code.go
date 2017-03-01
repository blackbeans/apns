package apns

const (
	DEVICE_TOKEN       = byte(1) //devicetoken
	PAY_LOAD           = byte(2) //payload
	NOTIFY_IDENTIYFIER = byte(3) //notify_identifier
	EXPIRATED_DATE     = byte(4) //expired date
	PRIORITY           = byte(5) // priority
)

const (
	CMD_SIMPLE_NOTIFY  = byte(0) //简单的notify
	CMD_ENHANCE_NOTIFY = byte(1) //扩充的notify
	CMD_POP            = byte(2) //正常的Notify命令
	CMD_RESP_ERR       = byte(8) //错误的响应头部
)

const (
	RESP_SUCC                 = byte(0)
	RESP_ERROR                = byte(1)
	RESP_MISS_TOKEN           = byte(2)
	RESP_MISS_TOPIC           = byte(3)
	RESP_MISS_PAYLOAD         = byte(4)
	RESP_INVALID_TOKEN_SIZE   = byte(5)
	RESP_INVALID_TOPIC_SIZE   = byte(6)
	RESP_INVALID_PAYLOAD_SIZE = byte(7)
	RESP_INVALID_TOKEN        = byte(8)
	RESP_SHUTDOWN             = byte(10)
	RESP_UNKNOW               = byte(255)
)
