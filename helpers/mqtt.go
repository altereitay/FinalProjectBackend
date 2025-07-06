package helpers

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/altereitay/FinalProjectBackend/db"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var client mqtt.Client

const SIMPLIFY_TOPIC = "articles/simplified"
const TERMS_TOPIC = "articles/terms"

type SimplifiedJSON struct {
	Hash   string `json:"hash"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type TermsJSON struct {
	Hash   string          `json:"hash"`
	Terms  []db.SingleTerm `json:"terms"`
	Status string          `json:"status`
}

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

func Publish(topic string, payload []byte) error {
	if client == nil || !client.IsConnected() {
		return fmt.Errorf("MQTT client not connected")
	}
	token := client.Publish(topic, 2, true, payload)
	token.Wait()
	return token.Error()
}

func Subscribe(topic string, handler mqtt.MessageHandler) error {
	if client == nil || !client.IsConnected() {
		return fmt.Errorf("MQTT client not connected")
	}
	token := client.Subscribe(topic, 2, handler)
	token.Wait()
	return token.Error()
}

func ClearQueue(topic string) error {
	return Publish(topic, nil)
}

func HandleSimplifiedArticles(client mqtt.Client, msg mqtt.Message) {
	var payload SimplifiedJSON

	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		log.Printf("bad JSON on %q: %v", msg.Topic(), err)
	}

	if payload.Status != "done" {
		return
	}

	if err := ClearQueue(SIMPLIFY_TOPIC); err != nil {
		log.Printf("ClearQueue error: %v", err)
		return
	}

	simplified, err := ReadTxt(payload.Name)
	if err != nil {
		log.Printf("readTxt error: %v", err)
		return
	}

	if err := db.AddSimplifiedVersion(payload.Hash, simplified); err != nil {
		log.Printf("db update error: %v", err)
		return
	}
}

func HandleTerms(client mqtt.Client, msg mqtt.Message) {
	var payload TermsJSON

	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		log.Printf("bad JSON on %q: %v", msg.Topic(), err)
	}

	if payload.Status != "done" {
		return
	}

	if err := ClearQueue(TERMS_TOPIC); err != nil {
		log.Printf("ClearQueue error: %v", err)
		return
	}

	if err := db.AddTerms(payload.Hash, payload.Terms); err != nil {
		log.Printf("db update error: %v", err)
	}
}
