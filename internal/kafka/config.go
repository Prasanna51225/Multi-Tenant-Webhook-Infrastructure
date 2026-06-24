package kafka

import "fmt"

type Config struct {
	Brokers []string
	Topic   string
}

func (c Config) BrokerString() string {
	return fmt.Sprintf("%v", c.Brokers)
}
