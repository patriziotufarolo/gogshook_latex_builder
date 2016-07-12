package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/patriziotufarolo/golang_latex_builder"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

type Hook struct {
	Secret  string
	Event   string
	Id      string
	Payload []byte
}

type Configuration struct {
	Secret    string           `json:"secret"`
	SSLEnable bool             `json:"ssl_enable"`
	SSLKey    string           `json:"ssl_key"`
	SSLCrt    string           `json:"ssl_crt"`
	Port      int              `json:"port"`
	Address   string           `json:"address"`
	Git       GitConfiguration `json:"git"`
}

type GitConfiguration struct {
	ProjectName string `json:"project_name"`
	Workdir     string `json:"workdir"`
	Outdir      string `json:"outdir"`
}

type Response struct {
	Secret      string     `json:"secret"`
	Ref         string     `json:"ref"`
	Before      string     `json:"before"`
	After       string     `json:"after"`
	Compare_url string     `json:"compare_url"`
	Commits     []Commit   `json:"commits"`
	Repository  Repository `json:"repository"`
	Pusher      Author     `json:"pusher"`
	Sender      Sender     `json:"sender"`
}

type Commit struct {
	Id      string `json:"id"`
	Message string `json:"message"`
	Url     string `json:"url"`
	Author  Author `json:"author"`
}

type Author struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type Repository struct {
	Id          uint64 `json:"id"`
	Name        string `json:"name"`
	Url         string `json:"url"`
	SshUrl      string `json:"ssh_url"`
	CloneUrl    string `json:"clone_url"`
	Description string `json:"description"`
	Website     string `json:"website"`
	Watchers    int    `json:"watchers"`
	Owner       Author `json:"owner"`
	Private     bool   `json:"private"`
}

type Sender struct {
	Login     string `json:"login"`
	Id        int    `json:"id"`
	AvatarUrl string `json:"avatar_url"`
}

var configuration Configuration

func Parse(secret []byte, req *http.Request) (*Hook, error) {
	hook := Hook{}

	if !strings.EqualFold(req.Method, "POST") {
		return nil, errors.New("Unsupported method")
	}

	if hook.Event = req.Header.Get("X-Gogs-Event"); len(hook.Event) == 0 {
		return nil, errors.New("No event")
	}

	if hook.Id = req.Header.Get("X-Gogs-Delivery"); len(hook.Id) == 0 {
		return nil, errors.New("No event id")
	}

	body, err := ioutil.ReadAll(req.Body)

	if err != nil {
		return nil, err
	}

	hook.Payload = body
	return &hook, nil
}

func GitServer(w http.ResponseWriter, req *http.Request) {
	hook, err := Parse(nil, req)

	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(string(hook.Payload[:]))
	}

	var res Response
	if err := json.Unmarshal(hook.Payload, &res); err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Server", "GOGS Hook server")

	if res.Secret == configuration.Secret {
		err = golang_latex_builder.Build(configuration.Git.ProjectName, res.Repository.CloneUrl, res.Commits[0].Id, configuration.Git.Workdir, configuration.Git.Outdir)
		if err != nil {
			w.WriteHeader(500)
			log.Println(err)
		} else {
			w.WriteHeader(200)
		}
	} else {
		w.WriteHeader(403)
	}
}

func main() {
	configuration = Configuration{}

	//READ SETTINGS FROM JSON FILE
	file, err := os.Open("conf.json")
	if err != nil {
		log.Fatal("Unable to read configuration: ", err)
	}
	decoder := json.NewDecoder(file)

	err = decoder.Decode(&configuration)

	if err != nil {
		log.Fatal("Unable to parse configuration: ", err)
	}
	fmt.Println("Configuration loaded")
	if configuration.SSLEnable && (configuration.SSLKey == "" || configuration.SSLCrt == "") {
		log.Fatal("Please specify both ssl key and ssl crt")
	}

	http.HandleFunc("/hook", GitServer)

	fmt.Printf("Server listening at: %s\n", fmt.Sprintf("%s:%d", configuration.Address, configuration.Port))
	fmt.Printf("SSLEnabled: %t\n", configuration.SSLEnable)
	if configuration.SSLEnable {
		err = http.ListenAndServeTLS(fmt.Sprintf("%s:%d", configuration.Address, configuration.Port), configuration.SSLCrt, configuration.SSLKey, nil)
	} else {
		err = http.ListenAndServe(fmt.Sprintf("%s:%d", configuration.Address, configuration.Port), nil)
	}
	if err != nil {
		log.Fatal(err)
	}
}
