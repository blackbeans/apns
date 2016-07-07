package protocol

const (
	CODE_SERVER_SUCC = 0
	// Error code list
	CODE_THROWABLE      = 300
	CODE_INITIALIZETION = 301
	CODE_SERIALIZATION  = 302
	CODE_REMOTING       = 303
	CODE_TIMEOUT        = 304

	CODE_NO_URI_FOUND          = 401
	CODE_INVALID_TARGET_URI    = 402
	CODE_INITIALIZATION_CLIENT = 3010
	CODE_SERIALIZATION_CLIENT  = 3020
	CODE_REMOTING_CLIENT       = 3030
	CODE_TIMEOUT_CLIENT        = 3040
	CODE_USER_CANCELLED        = 305

	CODE_SERVICE_NOT_FOUND     = 501
	CODE_METHOD_NOT_FOUND      = 502
	CODE_INVOCATION_TARGET     = 503
	CODE_THREAD_POOL_IS_FULL   = 504
	CODE_ASYNC_SUBMIT          = 505
	CODE_IP_NOT_ALLOWED        = 506
	CODE_INITIALIZATION_SERVER = 3011
	CODE_SERIALIZATION_SERVER  = 3021
	CODE_REMOTING_SERVER       = 3031
	CODE_TIMEOUT_SERVER        = 3041

	//error message list
	MSG_MESSAGE        = "Error code [%d], error message [%s]."
	MSG_INITIALIZATION = "Initialization exception: %s"
	MSG_REMOTING       = "Remoting exception: %s"
	MSG_SERIALIZATION  = "Serialization exception: %s"
	MSG_TIMEOUT        = "Timeout exception: %s"
	MSG_UNKNOWN        = "Unknown exception."

	MSG_NO_URI_FOUND        = "No uri found: %s"
	MSG_INVALID_TARGET_URI  = "Invalid target uri: %s"
	MSG_PARAMS_NOT_MATCHED  = "params number is not matched! %d/%d"
	MSG_SERVICE_NOT_FOUND   = "Service not found: %s."
	MSG_METHOD_NOT_FOUND    = "Method not found: %s."
	MSG_INVOCATION_TARGET   = "Invocation target exception: (%s)"
	MSG_THREAD_POOL_IS_FULL = "Threadpool is full: %s"
)
