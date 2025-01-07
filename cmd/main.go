package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	rabbitmqIndividualQueueMessages = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rabbitmq_individual_queue_messages",
			Help: "Total number of messages in a queue",
		},
		[]string{"host", "vhost", "name"},
	)
	rabbitmqIndividualQueueMessagesReady = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rabbitmq_individual_queue_messages_ready",
			Help: "Number of ready messages in a queue",
		},
		[]string{"host", "vhost", "name"},
	)
	rabbitmqIndividualQueueMessagesUnacknowledged = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rabbitmq_individual_queue_messages_unacknowledged",
			Help: "Number of unacknowledged messages in a queue",
		},
		[]string{"host", "vhost", "name"},
	)
)

func init() {
	prometheus.MustRegister(rabbitmqIndividualQueueMessages)
	prometheus.MustRegister(rabbitmqIndividualQueueMessagesReady)
	prometheus.MustRegister(rabbitmqIndividualQueueMessagesUnacknowledged)
}

func getRabbitMQMetrics(host string) error {
	url := fmt.Sprintf("http://%s/api/queues", host)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.SetBasicAuth(os.Getenv("RABBITMQ_USER"), os.Getenv("RABBITMQ_PASSWORD"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var queues []struct {
		Vhost                  string `json:"vhost"`
		Name                   string `json:"name"`
		Messages               int    `json:"messages"`
		MessagesReady          int    `json:"messages_ready"`
		MessagesUnacknowledged int    `json:"messages_unacknowledged"`
	}

	err = json.NewDecoder(resp.Body).Decode(&queues)
	if err != nil {
		return err
	}

	for _, q := range queues {
		rabbitmqIndividualQueueMessages.WithLabelValues(host, q.Vhost, q.Name).Set(float64(q.Messages))
		rabbitmqIndividualQueueMessagesReady.WithLabelValues(host, q.Vhost, q.Name).Set(float64(q.MessagesReady))
		rabbitmqIndividualQueueMessagesUnacknowledged.WithLabelValues(host, q.Vhost, q.Name).Set(float64(q.MessagesUnacknowledged))
	}

	return nil
}

func main() {
	host := os.Getenv("RABBITMQ_HOST")
	if host == "" {
		log.Fatal("RABBITMQ_HOST environment variable not set")
	}

	go func() {
		for {
			err := getRabbitMQMetrics(host)
			if err != nil {
				log.Printf("Error fetching RabbitMQ metrics: %v", err)
			}
			// Fetch metrics every minute
			<-time.After(time.Minute)
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	log.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
