package user

import (
	"encoding/json"

	"github.com/asdine/storm"
	"github.com/konkers/teletran"
)

type User struct {
	ID      string `json:"id",storm:"id"`
	Credits int    `json:"credits"`
}

type UserModule struct {
	bot *teletran.Bot
	db  storm.Node
}

func NewUserModule(bot *teletran.Bot) *UserModule {
	userModule := &UserModule{
		bot: bot,
		db:  bot.GetDbBucket("users"),
	}

	bot.AddCommand("whoami", "Look up user data.", userModule.whoamiCommand)

	return userModule
}

func (um *UserModule) whoamiCommand(ctx *teletran.CommandContext, args []string) {
	var userId string
	var err error
	if len(args) == 0 {
		userId = ctx.Message.Author.ID
	} else {
		userId, err = ctx.LookupUser(args[0])
		if err != nil {
			ctx.SendResponse(err.Error())
			return
		}
	}

	user, err := um.GetUser(userId)
	if err != nil {
		ctx.SendResponse("error getting user: " + err.Error())

	} else {
		b, _ := json.MarshalIndent(user, "", "  ")
		ctx.SendResponse(string(b))
	}
}

func (um *UserModule) GetAuthor(ctx *teletran.CommandContext) (*User, error) {
	return um.GetUser(ctx.Message.Author.ID)
}

func (um *UserModule) GetUser(ID string) (*User, error) {
	var user User
	err := um.db.One("ID", ID, &user)
	if err == storm.ErrNotFound {
		user.ID = ID
		user.Credits = 0
		um.db.Save(&user)
	} else if err != nil {
		return nil, err
	}

	return &user, nil
}

func (um *UserModule) Sync(user *User) error {
	return um.db.Update(user)
}
