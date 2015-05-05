package hooky

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"gopkg.in/yaml.v2"
)

type Retry struct {
	// Attempts is the current number of attempts we did.
	Attempts *int `yaml:"attempts,omitempty" json:"attempts,omitempty"`
	// MaxAttempts is the maximum number of attempts we will try.
	MaxAttempts *int `yaml:"max_attempts,omitempty" json:"maxAttempts,omitempty"`
	// Factor is factor to increase the duration between each attempts.
	Factor *float64 `yaml:"factor,omitempty" json:"factor,omitempty"`
	// Min is the minimum duration between each attempts in seconds.
	Min *int `yaml:"min,omitempty" json:"min,omitempty"`
	// Max is the maximum duration between each attempts in seconds.
	Max *int `yaml:"max,omitempty" json:"max,omitempty"`
}

type HTTPAuth struct {
	Username *string `yaml:"username,omitempty" json:"username,omitempty"`
	Password *string `yaml:"password,omitempty" json:"password,omitempty"`
}

type Account struct {
	// ID is the ID of the Account creating the Tasks.
	ID string `yaml:"id"`

	// Key is the key of the Account creating the Tasks.
	Key string `yaml:"key"`
}

type TasksDefaults struct {
	// Application is the name of the default Application.
	Application string `yaml:"application"`

	// Queue is the name of the default Queue.
	Queue string `yaml:"queue"`

	// HTTPAuth is the HTTP authentication to use if any.
	HTTPAuth *HTTPAuth `yaml:"auth,omitempty"`

	// Retry
	Retry *Retry `yaml:"retry,omitempty"`

	// Active is the task active.
	Active *bool `yaml:"active,omitempty"`
}

type Tasks struct {
	// Application is the name of the parent Application.
	Application *string `yaml:"application,omitempty" json:"-"`

	// Queue is the name of the parent Queue.
	Queue *string `yaml:"queue,omitempty" json:"-"`

	// Name is the task's name.
	Name string `yaml:"name" json:"-"`

	// URL is the URL that the worker with requests.
	URL string `yaml:"url" json:"url"`

	// HTTPAuth is the HTTP authentication to use if any.
	HTTPAuth *HTTPAuth `yaml:"auth,omitempty" json:"auth,omitempty"`

	// Method is the HTTP method that will be used to execute the request.
	Method *string `yaml:"method,omitempty" json:"method,omitempty"`

	// Headers are the HTTP headers that will be used schedule executing the request.
	Headers *map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`

	// Payload is arbitrary data that will be POSTed on the URL.
	Payload *string `yaml:"payload,omitempty" json:"payload,omitempty"`

	// Schedule is a cron specification describing the recurrency.
	Schedule string `yaml:"schedule" json:"schedule"`

	// Active is the task active.
	Active *bool `yaml:"active,omitempty" json:"active,omitempty"`

	// Retry is the retry strategy parameters in case of errors.
	Retry *Retry `yaml:"retry,omitempty" json:"retry,omitempty"`
}

type Crontab struct {
	Account       Account        `yaml:"account"`
	TasksDefaults *TasksDefaults `yaml:"tasks_defaults,omitempty"`
	Tasks         []*Tasks       `yaml:"tasks"`
}

func NewCrontabFromFile(path string) (*Crontab, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	crontab := &Crontab{}

	err = yaml.Unmarshal(content, crontab)
	if err != nil {
		return nil, err
	}
	return crontab, nil
}

func SyncCrontab(baseURL string, crontab *Crontab) error {
	for _, task := range crontab.Tasks {
		payload, err := json.Marshal(task)
		if err != nil {
			return err
		}
		url := fmt.Sprintf("%s/accounts/%s/applications/%s/queues/%s/tasks/%s", baseURL, crontab.Account.ID, "default", "default", task.Name)
		req, err := http.NewRequest("PUT", url, bytes.NewBuffer(payload))
		if err != nil {
			return err
		}
		req.Header.Add("User-Agent", "Hooky")
		req.Header.Add("Content-Type", "application/json")
		req.SetBasicAuth(crontab.Account.ID, crontab.Account.Key)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
	}
	return nil
}
