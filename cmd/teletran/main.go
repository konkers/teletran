package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/konkers/teletran"
	"github.com/konkers/teletran/store"
	"github.com/konkers/teletran/user"
)

var configFilename string
var storeConfigFilename string

func init() {
	flag.StringVar(&configFilename, "config", "teletran.json", "Config file.")
	flag.StringVar(&storeConfigFilename, "store-config", "store.json", "Store config file.")
}

func main() {
	flag.Parse()

	b, err := ioutil.ReadFile(configFilename)
	if err != nil {
		fmt.Printf("Error reading from %s: %s\n", configFilename, err.Error())
		return
	}

	var config teletran.Config
	err = json.Unmarshal(b, &config)
	if err != nil {
		fmt.Printf("Error decoding config: %s\n", err.Error())
		return
	}

	b, err = ioutil.ReadFile(storeConfigFilename)
	if err != nil {
		fmt.Printf("Error reading from %s: %s\n", configFilename, err.Error())
		return
	}

	var storeConfig store.Config
	err = json.Unmarshal(b, &storeConfig)
	if err != nil {
		fmt.Printf("Error decoding config: %s\n", err.Error())
		return
	}

	bot, err := teletran.NewBot(&config)
	if err != nil {
		fmt.Printf("Error creating bot: %s\n", err.Error())
		return
	}

	_ = user.NewUserModule(bot)
	_ = store.NewStoreModule(bot, &storeConfig)

	bot.Run()
}
