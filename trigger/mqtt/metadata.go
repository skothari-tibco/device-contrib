package mqtt

type Settings struct {
	Broker   string `md:"broker"`
	Id       string `md:"id"`
	User     string `md:"user"`
	Password string `md:"password"`
	Store    string `md:"store"`

	Cleansess bool `md:"cleansess"`
}

type HandlerSettings struct {
	Topic string `md:"topic"`
	Qos   int    `md:"qos"`
}

type Output struct {
	Message string `md:"message"`
}

type Reply struct {
	Data interface{} `md:"data"`
}
