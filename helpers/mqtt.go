package helpers

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/altereitay/FinalProjectBackend/db"
	"github.com/altereitay/FinalProjectBackend/helpers"
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
	token := client.Publish(topic, 0, true, payload)
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

func HandleSimplifiedArticles(client mqtt.Client, msg mqtt.Message) {
	var payload struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		log.Printf("bad JSON on %q: %v", msg.Topic(), err)
	}

	simplified, err := helpers.ReadTxt(payload.Name)
	if err != nil {
		log.Printf("readTxt error: %v", err)
	}

	if err := db.AddSimplifiedVersion(payload.ID, simplified); err != nil {
		log.Printf("db update error: %v", err)
	}
}
