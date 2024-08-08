package cleisthenes

type CMISMessage struct {
	CMIS map[Member]struct{}
}

type CMISSender interface {
	Send(cmis CMISMessage)
}

type CMISReceiver interface {
	Receive() <-chan CMISMessage
}

type CMISChannel struct {
	buffer chan CMISMessage
}

func NewCMISChannel() *CMISChannel {
	return &CMISChannel{
		buffer: make(chan CMISMessage, 1),
	}
}

func (c *CMISChannel) Send(cmis CMISMessage) {
	c.buffer <- cmis
}

func (c *CMISChannel) Receive() <-chan CMISMessage {
	return c.buffer
}
