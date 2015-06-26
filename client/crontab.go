package hooky

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/sebest/hooky/models"
	"github.com/sebest/hooky/restapi"
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

type TaskDefaults struct {
	// Queue is the name of the default Queue.
	Queue string `yaml:"queue"`

	// HTTPAuth is the HTTP authentication to use if any.
	HTTPAuth *HTTPAuth `yaml:"auth,omitempty"`

	// Retry
	Retry *Retry `yaml:"retry,omitempty"`

	// Active is the task active.
	Active *bool `yaml:"active,omitempty"`
}

type Task struct {
	// Queue is the name of the parent Queue.
	Queue *string `yaml:"queue,omitempty" json:"queue,omitempty"`

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
	Account      Account       `yaml:"account"`
	TaskDefaults *TaskDefaults `yaml:"tasks_defaults,omitempty"`
	Tasks        []*Task       `yaml:"tasks"`
	Application  string        `yaml:"application"`
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

func newReq(method string, url string, username string, password string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "Hooky")
	req.Header.Add("Content-Type", "application/json")
	if username != "" || password != "" {
		req.SetBasicAuth(username, password)
	}
	return req, nil
}

func SyncCrontab(baseURL string, crontab *Crontab) error {
	// Application
	app := crontab.Application
	if app == "" {
		app = "default"
	}

	client := &http.Client{}

	currentTasks := make(map[string]bool, 0)

	// Check that the application exists
	url := fmt.Sprintf("%s/accounts/%s/applications/%s", baseURL, crontab.Account.ID, app)
	req, err := newReq("GET", url, crontab.Account.ID, crontab.Account.Key, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		if resp.StatusCode == 404 {
			url := fmt.Sprintf("%s/accounts/%s/applications/%s", baseURL, crontab.Account.ID, app)
			req, err := newReq("PUT", url, crontab.Account.ID, crontab.Account.Key, nil)
			if err != nil {
				return err
			}
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			respBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				fmt.Println(string(respBody))
			}
		} else {
			fmt.Println(string(respBody))
		}
	}

	// Set all the tasks from the crontab
	for _, task := range crontab.Tasks {
		currentTasks[task.Name] = true
		payload, err := json.Marshal(task)
		if err != nil {
			return err
		}

		url := fmt.Sprintf("%s/accounts/%s/applications/%s/tasks/%s", baseURL, crontab.Account.ID, app, task.Name)
		req, err := newReq("PUT", url, crontab.Account.ID, crontab.Account.Key, bytes.NewBuffer(payload))
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if resp.StatusCode != 200 {
			fmt.Println(string(respBody))
		}
	}

	// Find tasks that have a schedule
	url = fmt.Sprintf("%s/accounts/%s/applications/%s/tasks?filters=schedule:true", baseURL, crontab.Account.ID, app)
	req, err = newReq("GET", url, crontab.Account.ID, crontab.Account.Key, nil)
	resp, err = client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	payload, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var tasks []*restapi.Task
	lr := &models.ListResult{
		List: &tasks,
	}
	if err := json.Unmarshal(payload, lr); err != nil {
		fmt.Println(err)
		return err
	}

	// Delete tasks that should not be there
	for _, task := range tasks {
		if _, ok := currentTasks[task.Name]; ok == false {
			url := fmt.Sprintf("%s/accounts/%s/applications/%s/tasks/%s", baseURL, crontab.Account.ID, app, task.Name)
			req, err := newReq("DELETE", url, crontab.Account.ID, crontab.Account.Key, nil)
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
		}
	}

	return nil
}
