package mqtt

import (
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/data/coerce"
	"github.com/project-flogo/core/data/metadata"
)

var MqttActivitymd = activity.ToMetadata(&Settings{}, &Input{}, &Output{})

func init() {
	_ = activity.Register(&MqttActivity{}, New)
}

func New(ctx activity.InitContext) (activity.Activity, error) {
	settings := &Settings{}
	err := metadata.MapToStruct(ctx.Settings(), settings, true)
	if err != nil {
		return nil, err
	}

	if !settings.SharedConnection {
		options := initClientOption(settings)
		if err != nil {
			//ctx.Logger().Errorf("Kafka parameters initialization got error: [%s]", err.Error())
			return nil, err
		}

		mqttClient := mqtt.NewClient(options)

		if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
			return nil, token.Error()
		}
		act := &MqttActivity{settings: settings, client: mqttClient}
		return act, nil
	}
	

	act := &MqttActivity{settings: settings}
	return act, nil
}

type MqttActivity struct {
	settings *Settings
	client   mqtt.Client
}

func (a *MqttActivity) Metadata() *activity.Metadata {
	return MqttActivitymd
}

func (a *MqttActivity) Eval(ctx activity.Context) (done bool, err error) {

	input := &Input{}

	err = ctx.GetInputObject(input)

	if err != nil {
		return true, err
	}
	if input.Connection != nil {
		ctx.Logger().Info("Using Shared Connection to publish..", input.Message)
		if token := input.Connection.GetConnection().(mqtt.Client).Publish(input.Topic, byte(input.Qos), true, input.Message); token.Wait() && token.Error() != nil {
			ctx.Logger().Info("Error in publishing..")
			return true, token.Error()
		}

	}else {
		if token := a.client.Publish(input.Topic, byte(input.Qos), true, input.Message); token.Wait() && token.Error() != nil {
			ctx.Logger().Info("Error in publishing..")
			return true, token.Error()
		}
		a.client.Disconnect(a.settings.Close)

	}
	
	ctx.Logger().Info("Message Published ..")

	return true, nil
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
