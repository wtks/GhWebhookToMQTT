package main

import (
	"encoding/json"
	"github.com/eclipse/paho.mqtt.golang"
	"gopkg.in/go-playground/webhooks.v5/github"
	"log"
	"net/http"
	"os"
)

var (
	MQTTClientID = os.Getenv("MQTT_CLIENT_ID")
	MQTTHost     = os.Getenv("MQTT_HOST")
	MQTTUserName = os.Getenv("MQTT_USERNAME")
	MQTTPassword = os.Getenv("MQTT_PASSWORD")
	GithubSecret = os.Getenv("GITHUB_SECRET")
	TopicPrefix  = os.Getenv("TOPIC_PREFIX")
	Port         = os.Getenv("HTTP_PORT")

	AcceptableEvents = []github.Event{
		github.PushEvent,
		github.ReleaseEvent,
	}
)

func main() {
	if len(TopicPrefix) == 0 {
		TopicPrefix = "/GhWebhook/"
	}
	if len(MQTTClientID) == 0 {
		MQTTClientID = "GhWebhookToMQTT"
	}
	if len(Port) == 0 {
		Port = "8080"
	}

	// init mqtt client
	mqttOpt := mqtt.NewClientOptions()
	mqttOpt.AddBroker(MQTTHost)
	mqttOpt.SetUsername(MQTTUserName)
	mqttOpt.SetPassword(MQTTPassword)
	mqttOpt.SetClientID(MQTTClientID)

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

		log.Println("Webhook was received")
		var (
			topic   = TopicPrefix
			message = ""
		)

		switch v := payload.(type) {
		case github.PushPayload:
			topic += v.Repository.FullName + "/push"
			message = convertToJson(map[string]interface{}{
				"ref": v.Ref,
			})
		case github.ReleasePayload:
			topic += v.Repository.FullName + "/release"
			message = convertToJson(map[string]interface{}{
				"id":        v.Release.ID,
				"tag":       v.Release.TagName,
				"draft":     v.Release.Draft,
				"hasAssets": len(v.Release.Assets) > 0,
			})
		}

		token := client.Publish(topic, 1, false, message)
		if token.Wait() && token.Error() != nil {
			log.Printf("ERROR: %v", token.Error().Error())
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.Println("a Message was published:", topic)

		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte("OK"))
		return
	})
	log.Fatal(http.ListenAndServe(":"+Port, nil))
}

func convertToJson(m map[string]interface{}) string {
	b, _ := json.Marshal(m)
	return string(b)
}
