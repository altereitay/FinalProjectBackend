package helpers

import (
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var client mqtt.Client

func InitMQTT() error {
	broker := "mqtt://localhost:1883"
	clientID := "backend-client"
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.SetKeepAlive(2 * time.Second)
	opts.SetPingTimeout(1 * time.Second)

	opts.OnConnect = func(c mqtt.Client) {
		log.Printf("Connected to mqtt broker: %s", broker)
	}

	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		log.Printf("Connection lost: %v", err)
	}

	client = mqtt.NewClient(opts)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func Publish(topic string, payload string) error {
	if client == nil || !client.IsConnected() {
		return fmt.Errorf("MQTT client not connected")
	}
	token := client.Publish(topic, 0, false, payload)
	token.Wait()
	return token.Error()
}

func Subscribe(topic string, handler mqtt.MessageHandler) error {
	if client == nil || !client.IsConnected() {
		return fmt.Errorf("MQTT client not connected")
	}
	token := client.Subscribe(topic, 0, handler)
	token.Wait()
	return token.Error()
}
