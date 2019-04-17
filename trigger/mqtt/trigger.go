package mqtt

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/project-flogo/core/data/coerce"
	"github.com/project-flogo/core/data/metadata"
	"github.com/project-flogo/core/support/log"
	"github.com/project-flogo/core/trigger"
)

var triggerMd = trigger.NewMetadata(&Settings{}, &HandlerSettings{}, &Output{})

func init() {
	_ = trigger.Register(&MqttTrigger{}, &Factory{})
}

// MqttTrigger is simple MQTT trigger
type MqttTrigger struct {
	handlers []clientHandler
	settings *Settings
	logger   log.Logger
}
type clientHandler struct {
	client mqtt.Client
	topic  string
	qos    int
}
type Factory struct {
}

func (*Factory) Metadata() *trigger.Metadata {
	return triggerMd
}

// New implements trigger.Factory.New
func (*Factory) New(config *trigger.Config) (trigger.Trigger, error) {
	s := &Settings{}

	err := metadata.MapToStruct(config.Settings, s, true)
	if err != nil {
		return nil, err
	}

	return &MqttTrigger{settings: s, logger: ctx.Logger()}, nil
}

// Initialize implements trigger.Initializable.Initialize
func (t *MqttTrigger) Initialize(ctx trigger.InitContext) error {

	for _, handler := range ctx.GetHandlers() {
		options := initClientOption(t.settings)

		s := &HandlerSettings{}
		err := metadata.MapToStruct(handler.Settings(), s, true)
		if err != nil {
			return err
		}
		options.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
			topic := msg.Topic()

			payload := string(msg.Payload())
			t.RunHandler(handler, payload)

		})

		//Creating new client for each handler because each client struct expects one publish handler
		client := mqtt.NewClient(options)

		if token := client.Connect(); token.Wait() && token.Error() != nil {
			return token.Error()
		}
		t.handlers = append(t.handlers, clientHandler{client: client, topic: s.Topic, qos: s.Qos})
	}

	return nil
}

func initClientOption(settings *Settings) *mqtt.ClientOptions {

	opts := mqtt.NewClientOptions()
	opts.AddBroker(settings.Broker)
	opts.SetClientID(settings.Id)
	opts.SetUsername(settings.User)
	opts.SetPassword(settings.Password)
	b, err := coerce.ToBool(settings.Cleansess)
	if err != nil {
		//log.Error("Error converting \"cleansess\" to a boolean ", err.Error())
		return nil
	}
	opts.SetCleanSession(b)
	if storeType := settings.Store; storeType != ":memory:" {
		opts.SetStore(mqtt.NewFileStore(settings.Store))
	}
	return opts
}

// Start implements trigger.Trigger.Start
func (t *MqttTrigger) Start() error {

	for _, handler := range t.handlers {

		if token := handler.client.Subscribe(handler.topic, byte(handler.qos), nil); token.Wait() && token.Error() != nil {
			t.logger.Errorf("Error subscribing to topic %s: %s", handler.topic, token.Error())
			return token.Error()
		} else {
			t.logger.Debugf("Subscribed to topic: %s, will trigger handler: %s", handler.topic, handler)
		}
	}

	return nil
}

// Stop implements ext.Trigger.Stop
func (t *MqttTrigger) Stop() error {
	//unsubscribe from topic
	for _, handler := range t.handlers {
		t.logger.Debug("Unsubscribing from topic: ", handler.topic)
		if token := handler.client.Unsubscribe(handler.topic); token.Wait() && token.Error() != nil {
			t.logger.Errorf("Error unsubscribing from topic %s: %s", handler.topic, token.Error())

		}
		handler.client.Disconnect(250)
	}

	return nil
}

// RunHandler runs the handler and associated action
func (t *MqttTrigger) RunHandler(handler trigger.Handler, payload string) {

	trgData := make(map[string]interface{})
	trgData["message"] = payload

	results, err := handler.Handle(context.Background(), trgData)

	if err != nil {
		t.logger.Error("Error starting action: ", err.Error())
	}

	t.logger.Debugf("Ran Handler: [%s]", handler)

	var replyData interface{}

	if len(results) != 0 {
		dataAttr, ok := results["data"]
		if ok {
			replyData = dataAttr.Value()
		}
	}

	if replyData != nil {
		dataJson, err := json.Marshal(replyData)
		if err != nil {
			t.logger.Error(err)
		} else {
			replyTo := handler.GetStringSetting("topic")
			if replyTo != "" {
				t.publishMessage(replyTo, string(dataJson))
			}
		}
	}
}

func (t *MqttTrigger) publishMessage(topic string, message string) {

	t.logger.Debug("ReplyTo topic: ", topic)
	t.logger.Debug("Publishing message: ", message)

	qos, err := strconv.Atoi(t.config.GetSetting("qos"))
	if err != nil {
		t.logger.Error("Error converting \"qos\" to an integer ", err.Error())
		return
	}
	if len(topic) == 0 {
		t.logger.Warn("Invalid empty topic to publish to")
		return
	}
	token := t.client.Publish(topic, byte(qos), false, message)
	sent := token.WaitTimeout(5000 * time.Millisecond)
	if !sent {
		// Timeout occurred
		log.Errorf("Timeout occurred while trying to publish to topic '%s'", topic)
		return
	}
}
