package main

import (
	"io"
	"net/http"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/mitchellh/mapstructure"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v2"
)

type ReloadFunc func(file string) (Versions, error)

type Subscriber struct {
	url        string
	accessKey  string
	secretKey  string
	reload     ReloadFunc
	answerFile string
	client     *http.Client
}

func NewSubscriber(url, accessKey, secretKey, answerFile string, reload ReloadFunc) *Subscriber {
	return &Subscriber{
		url:        url,
		accessKey:  accessKey,
		secretKey:  secretKey,
		reload:     reload,
		answerFile: answerFile,
		client:     &http.Client{},
	}
}

func (s *Subscriber) Subscribe() error {
	handlers := map[string]revents.EventHandler{
		"ping":          s.noOp,
		"config.update": s.configUpdate,
	}

	router, err := revents.NewEventRouter("", 0, s.url, s.accessKey, s.secretKey, nil, handlers, "", 3, revents.DefaultPingConfig)
	if err != nil {
		return err
	}

	go func() {
		logrus.Fatalf("Exiting subscriber: %v", router.StartWithoutCreate(nil))
	}()

	return nil
}

func (s *Subscriber) noOp(event *revents.Event, c *client.RancherClient) error {
	return nil
}

func (s *Subscriber) configUpdate(event *revents.Event, c *client.RancherClient) error {
	update := ConfigUpdateData{}
	if err := mapstructure.Decode(event.Data, &update); err != nil {
		return err
	}

	found := false
	for _, item := range update.Items {
		if found = item.Name == "metadata-answers"; found {
			if err := s.downloadAndReload(); err != nil {
				return err
			}
			break
		}
	}

	_, err := c.Publish.Create(&client.Publish{
		Name:        event.ReplyTo,
		PreviousIds: []string{event.ID},
	})
	return err
}

func (s *Subscriber) downloadAndReload() error {
	url := s.url + "/configcontent/metadata-answers"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(s.accessKey, s.secretKey)
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	tempFile := s.answerFile + ".temp"
	out, err := os.Create(tempFile)
	if err != nil {
		return err
	}
	defer func() {
		out.Close()
		os.Remove(tempFile)
	}()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	versions, err := s.reload(tempFile)
	if err != nil {
		return err
	}

	os.Rename(tempFile, s.answerFile)
	def, ok := versions["latest"]["default"].(map[string]interface{})
	if ok {
		version, _ := def["version"].(string)
		logrus.Infof("Applied %s", url+"?version="+version)
		req, err := http.NewRequest("PUT", url+"?version="+version, nil)
		if err != nil {
			return err
		}

		req.SetBasicAuth(s.accessKey, s.secretKey)
		resp, err := s.client.Do(req)
		if err != nil {
			return err
		}
		if resp.Body != nil {
			resp.Body.Close()
		}
	}

	return err
}

type ConfigUpdateData struct {
	ConfigUrl string
	Items     []ConfigUpdateItem
}

type ConfigUpdateItem struct {
	Name string
}
