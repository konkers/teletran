package store

import (
	"fmt"
	"strconv"

	"github.com/asdine/storm"
	"github.com/konkers/teletran"
)

type Item struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Cost   int    `json:"cost"`
	Unique bool   `json:"unique"`
}

type Inventory struct {
	Pk     int    `storm:"id,increment"`
	Owner  string `storm:"index"`
	ItemId string `storm:"index"`
}

type Config struct {
	Items []*Item `json:"items"`
}

type StoreModule struct {
	config        *Config
	bot           *teletran.Bot
	db            storm.Node
	commands      *teletran.CommandSet
	unitsCommands *teletran.CommandSet

	itemCache map[string]*Item
}

type balance struct {
	UserId  string `storm:"id"`
	Balance int
}

func NewStoreModule(bot *teletran.Bot, config *Config) *StoreModule {
	sm := &StoreModule{
		config:        config,
		bot:           bot,
		db:            bot.GetDbBucket("store"),
		commands:      teletran.NewCommandSet(),
		unitsCommands: teletran.NewCommandSet(),
		itemCache:     make(map[string]*Item),
	}

	for _, item := range config.Items {
		sm.itemCache[item.Id] = item
	}

	sm.commands.AddCommand("items", "View items for sale.", sm.itemsCommand, false)
	sm.commands.AddCommand("buy", "Buy and item.", sm.buyCommand, false)
	sm.commands.AddCommand("units", "Display you're units balance.", sm.unitsCommand, false)

	sm.unitsCommands.AddCommand("add", "Add units to user.", sm.unitsAddCommand, true)

	bot.AddCommand("store", "Interact with the store.", sm.storeCommand)
	bot.AddCommand("inventory", "Displays which items you own.", sm.inventoryCommand)

	return sm
}

func (sm *StoreModule) getBalance(userId string) (*balance, error) {
	db := sm.db.From("balance")
	var balance balance

	err := db.One("UserId", userId, &balance)

	if err == storm.ErrNotFound {
		balance.UserId = userId
		balance.Balance = 0
		db.Save(&balance)
	} else if err != nil {
		return nil, err
	}

	return &balance, nil
}

func (sm *StoreModule) getInventory(userId string) ([]Inventory, error) {
	var inventory []Inventory
	// err := sm.db.From("inventory").Find("Owner", userId, &inventory)
	err := sm.db.From("inventory").All(&inventory)
	if err == storm.ErrNotFound {
		return []Inventory{}, nil
	} else if err != nil {
		return nil, err
	}
	return inventory, nil
}

func (sm *StoreModule) updateBalance(balance *balance) error {
	db := sm.db.From("balance")
	return db.Update(balance)
}

func (sm *StoreModule) storeCommand(ctx *teletran.CommandContext, args []string) {
	sm.commands.Exec(ctx, args)
}

func (sm *StoreModule) itemsCommand(ctx *teletran.CommandContext, args []string) {
	response := "```Store inventory:\n"
	for i, item := range sm.config.Items {
		response += fmt.Sprintf(" [%d] %s (cost %d)\n", i, item.Name, item.Cost)
	}
	response += "```"
	ctx.SendResponse(response)
}

func (sm *StoreModule) buyCommand(ctx *teletran.CommandContext, args []string) {

	if len(args) != 1 {
		ctx.SendResponse("usage: buy <item number>")
		return
	}

	itemNumber, err := strconv.Atoi(args[0])
	if err != nil {
		ctx.SendResponse(fmt.Sprintf("%s is not a number.", args[0]))
		return
	}

	if itemNumber < 0 || itemNumber >= len(sm.config.Items) {
		ctx.SendResponse(fmt.Sprintf("Item %d out of range.", itemNumber))
	}

	balance, err := sm.getBalance(ctx.Message.Author.ID)
	if err != nil {
		ctx.SendResponse(fmt.Sprintf("Error retrieving balance: %s", err.Error()))
		return
	}

	item := sm.config.Items[itemNumber]

	if item.Cost > balance.Balance {
		ctx.SendResponse("You don't have enough units.")
		return
	}

	balance.Balance -= item.Cost

	inv := &Inventory{
		Owner:  ctx.Message.Author.ID,
		ItemId: item.Id,
	}

	err = sm.db.From("inventory").Save(inv)
	if err != nil {
		ctx.SendResponse(fmt.Sprintf("Error purchasing %s: %s",
			item.Name, err.Error()))
		return
	}
	sm.updateBalance(balance)

	ctx.SendResponse(fmt.Sprintf("%s purchased.", item.Name))
}

func (sm *StoreModule) unitsCommand(ctx *teletran.CommandContext, args []string) {
	if len(args) == 0 {
		balance, err := sm.getBalance(ctx.Message.Author.ID)
		if err != nil {
			ctx.SendResponse(fmt.Sprintf("Error retrieving balance: %s", err.Error()))
			return
		}
		ctx.SendResponse(fmt.Sprintf("Your balance is %d.", balance.Balance))
	} else {
		sm.unitsCommands.Exec(ctx, args)
	}
}

func (sm *StoreModule) unitsAddCommand(ctx *teletran.CommandContext, args []string) {
	if len(args) != 2 {
		ctx.SendResponse("usage: units add <user_id> <delta>.")
		return
	}

	delta, err := strconv.Atoi(args[1])
	if err != nil {
		ctx.SendResponse(fmt.Sprintf("%s is not a number.", args[1]))
		return
	}

	userId, err := ctx.LookupUser(args[0])
	if err != nil {
		ctx.SendResponse(err.Error())
		return
	}

	balance, err := sm.getBalance(userId)
	if err != nil {
		ctx.SendResponse(fmt.Sprintf("Can't get balance for %s.", args[0]))
		return
	}

	balance.Balance += delta

	sm.updateBalance(balance)
	ctx.SendResponse(fmt.Sprintf("%s's new balance: %d", args[0], balance.Balance))
}

func (sm *StoreModule) inventoryCommand(ctx *teletran.CommandContext, args []string) {
	inventory, err := sm.getInventory(ctx.Message.Author.ID)
	if err != nil {
		ctx.SendResponse(fmt.Sprintf("Can't fetch inventory: %s", err.Error()))
		return
	}

	if len(inventory) == 0 {
		ctx.SendResponse("Your inventory is empty.")
		return
	}

	message := "Your inventory:\n"
	for _, line := range inventory {
		item, _ := sm.itemCache[line.ItemId]
		message += fmt.Sprintf(" * %s\n", item.Name)
	}
	ctx.SendResponse(message)
}
