package main

import (
	"github.com/eclipse/paho.mqtt.golang"
	"gopkg.in/go-playground/webhooks.v5/github"
	"log"
	"net/http"
	"os"
)

const ClientID = "GhWebhookToMQTT"

var (
	MQTTHost     = os.Getenv("MQTT_HOST")
	MQTTUserName = os.Getenv("MQTT_USERNAME")
	MQTTPassword = os.Getenv("MQTT_PASSWORD")
	GithubSecret = os.Getenv("GITHUB_SECRET")

	AcceptableEvents = []github.Event{
		github.PushEvent,
	}
)

func main() {
	// init mqtt client
	mqttOpt := mqtt.NewClientOptions()
	mqttOpt.AddBroker(MQTTHost)
	mqttOpt.SetUsername(MQTTUserName)
	mqttOpt.SetPassword(MQTTPassword)
	mqttOpt.SetClientID(ClientID)

	client := mqtt.NewClient(mqttOpt)
	defer client.Disconnect(250)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	// init webhook
	hook, err := github.New(github.Options.Secret(GithubSecret))
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/hook", func(rw http.ResponseWriter, req *http.Request) {
		payload, err := hook.Parse(req, AcceptableEvents...)
		if err != nil {
			if err == github.ErrEventNotFound {
				rw.WriteHeader(http.StatusNoContent)
				return
			}
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		log.Println("Webhook received")
		var (
			topic   = "/GhWebhook/"
			message = ""
		)

		switch v := payload.(type) {
		case github.PushPayload:
			topic += v.Repository.FullName + "/push"
			message = "ref:" + v.Ref
		}

		token := client.Publish(topic, 1, false, message)
		if token.Wait() && token.Error() != nil {
			log.Printf("ERROR: %v", token.Error().Error())
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.Println("a MQTT Message published")

		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte("OK"))
		return
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
