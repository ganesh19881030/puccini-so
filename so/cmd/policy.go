package cmd

import (
	"encoding/json"
	logger "log"
	"math/rand"
	"os"
	"time"

	gopher_and_rabbit "github.com/masnun/gopher-and-rabbit"
	"github.com/streadway/amqp"
)

/*
	NOTE:

	This is a 'special' case implementation of policy engine.
	Its purpose is to demonstrate the entire event notification generation
	and policy execution process at runtime using a simple RabbitMQ
	based Producer/Consumer implementation.

	This is NOT a general purpose implementation of policy engine
	which will be required to handle a real life scenario.
*/
var stopProducer chan bool
var stopConsumer chan bool

// ExecutePolicy ...
func ExecutePolicy(policy *PolicyDefinition) interface{} {
	PolicyStartStepName := policy.Name + ".start"
	triggerSteps := make([]PolicyStepDefinition, 0)

	// Find trigger Steps from policy 'start' step
	for _, step := range policy.Steps {
		if step.Name != PolicyStartStepName {
			continue
		}

		for _, trigger := range step.Triggers {
			for _, step := range policy.Steps {
				if trigger == step.Name {
					triggerSteps = append(triggerSteps, *step)
				}
			}
		}
	}

	// start producer and consumer
	go startProducer()
	go startConsumer(triggerSteps)
	return nil
}

// DeletePolicy ... this actually stops producer and consumer execution
// which involves stopping of threads and deleting RabbitMQ resources
func DeletePolicy(policy *PolicyDefinition) interface{} {
	PolicyStopStepName := policy.Name + ".stop"

	for _, step := range policy.Steps {
		if step.Name != PolicyStopStepName {
			continue
		}

		if stopProducer != nil {
			close(stopProducer)
			logger.Printf("Producer stopped, PID: %d", os.Getpid())
		}

		if stopConsumer != nil {
			close(stopConsumer)
			logger.Printf("Consumer stopped, PID: %d", os.Getpid())
		}
	}
	return nil
}

// producer simulates real life generation of events by simply generating
// events periodically and sending them to RabbitMQ queue
func startProducer() {
	producerConnection, producerErr := amqp.Dial(gopher_and_rabbit.Config.AMQPConnectionURL)
	handleError(producerErr, "Can't connect to AMQP")
	defer producerConnection.Close()

	amqpChannel, err := producerConnection.Channel()
	handleError(err, "Can't create a amqpChannel")

	defer amqpChannel.Close()

	queue, err := amqpChannel.QueueDeclare("firewallDemo", true, false, false, false, nil)
	handleError(err, "Could not declare `firewallDemo` queue")
	defer amqpChannel.QueueDelete("firewallDemo", false, false, false)

	rand.Seed(time.Now().UnixNano())

	stopProducer = make(chan bool)

	go func() {
		logger.Printf("Producer ready to send EventType and Traffic_Volume, PID: %d", os.Getpid())

		// generate an event notification every 10 seconds and send it to Queue
		for {
			time.Sleep(10 * time.Second)

			select {
			case <-stopProducer:
				x := <-stopProducer
				logger.Printf("Producer stopped->channel value: %t", x)
				return
			default:
				trafficVolume := rand.Intn(999)
				var eventType string
				if trafficVolume >= 200 && trafficVolume <= 250 {
					eventType = "Traffic Volume is between 200 and 250"
				} else if trafficVolume >= 850 && trafficVolume <= 900 {
					eventType = "Traffic Volume is between 850 and 900"
				} else {
					eventType = "cci.interfaces.TrafficMonitor.traffic_volume_notification"
				}

				eventJSON, err := json.Marshal(&eventMessage{eventType, trafficVolume})
				logger.Printf("Producer->Send msg: %s", string(eventJSON))

				err = amqpChannel.Publish("", queue.Name, false, false, amqp.Publishing{
					ContentType: "application/json",
					Body:        []byte(eventJSON),
				})

				if err != nil {
					logger.Printf("Error publishing message on Queue firewallDemo: %s", err)
				}
			}
		}
	}()

	<-stopProducer
}

// consumer listens to notifications received on RabbitMQ queue and calls trigger functions
func startConsumer(triggerSteps []PolicyStepDefinition) {
	consumerConnection, consumerErr := amqp.Dial(gopher_and_rabbit.Config.AMQPConnectionURL)
	handleError(consumerErr, "Consumer-> Can't connect to AMQP")
	defer consumerConnection.Close()

	amqpChannel, err := consumerConnection.Channel()
	handleError(err, "Consumer-> Can't create a amqpChannel")

	defer amqpChannel.Close()

	queue, err := amqpChannel.QueueDeclare("firewallDemo", true, false, false, false, nil)
	handleError(err, "Consumer-> Could not declare `firewallDemo` queue")
	defer amqpChannel.QueueDelete("firewallDemo", false, false, false)

	err = amqpChannel.Qos(1, 0, false)
	handleError(err, "Consumer-> Could not configure QoS")

	messageChannel, err := amqpChannel.Consume(
		queue.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	handleError(err, "Consumer-> Could not register consumer")

	stopConsumer = make(chan bool)

	go func() {
		logger.Printf("Consumer ready, PID: %d", os.Getpid())

		select {
		case <-stopConsumer:
			x := <-stopConsumer
			logger.Printf("Consumer stopped->channel value: %t", x)
			return
		default:
			for d := range messageChannel {
				logger.Printf("Consumer-> Received a message: %s", d.Body)

				data := eventMessage{}
				err := json.Unmarshal(d.Body, &data)

				if err != nil {
					logger.Printf("Consumer-> Error decoding JSON: %s", err)
				}

				logger.Printf("Consumer->Received event :%s,volume : %d", data.EventType, data.TrafficVolume)

				if err := d.Ack(true); err != nil {
					logger.Printf("Error acknowledging message : %s", err)
				} else {
					logger.Printf("Consumer-> Acknowledged message")
				}

				for _, triggerStep := range triggerSteps {
					if triggerStep.EventType == data.EventType {
						logger.Printf("Consumer->Event matched with trigger, Call trigger:%s", triggerStep.Name)
						executeTrigger(data.TrafficVolume, triggerStep)
					} else {
						logger.Printf("Consumer->Event not matched with trigger :%s", triggerStep.Name)
					}
				}
			}

		}
	}()

	<-stopConsumer
}

func executeTrigger(operand1 int, triggerStep PolicyStepDefinition) {
	conditionObjects := triggerStep.Conditions
	for _, conditionObject := range conditionObjects {
		for _, condition := range *conditionObject {
			for operator, operand2 := range condition {
				if operator == "greater_than" && operand1 > int(operand2.(float64)) {
					logger.Printf("Trigger %s -> Received traffic volume is : %d so Packet_rate updated to 300", triggerStep.Name, operand1)
				} else if operator == "less_than" && operand1 < int(operand2.(float64)) {
					logger.Printf("Trigger %s -> Received traffic volume is : %d so Packet_rate updated to 800", triggerStep.Name, operand1)
				} else if operator == "equal" && operand1 == int(operand2.(float64)) {
					logger.Printf("Trigger %s -> Received traffic volume is : %d so don't need to update packet_rate", triggerStep.Name, operand1)
				} else {
					logger.Printf("Trigger %s -> Received traffic volume is : %d but condition not matched", triggerStep.Name, operand1)
				}
			}
		}
	}
}

type eventMessage struct {
	EventType     string
	TrafficVolume int
}

func handleError(err error, msg string) {
	if err != nil {
		logger.Printf("%s: %s", msg, err)
	}
}
