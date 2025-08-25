package telegram

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"death-clock/lib/e"
	"death-clock/storage"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	LifeCalendarCmd   = "📅 Life calendar"
	HelpCmd           = "/help"
	StartCmd          = "/start"
	StartCalculateCmd = "👋 Start"
	OpenNotebookCmd   = "📖 Open my notebook"
	ShowTimeLeftCmd   = "🕘 How much time do i have left?"
)

func GetStaticKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(StartCalculateCmd),
			tgbotapi.NewKeyboardButton(ShowTimeLeftCmd),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(OpenNotebookCmd),
			tgbotapi.NewKeyboardButton(LifeCalendarCmd),
		),
	)
}

func (p *Processor) doCmd(text string, chatID int, username string) error {
	text = strings.TrimSpace(text)

	log.Printf("got new command '%s' from '%s", text, username)

	userExists, err := p.storage.IsUserExists(context.Background(), username)
	if err != nil {
		return e.Wrap("can't check if user exists in db: %s", err)
	}

	if !userExists {
		user := &storage.User{
			UserName:        username,
			IsDeathAgeAsked: false,
			IsBirthdayAsked: false,
		}

		p.storage.InitUser(context.Background(), user)
	}

	user, err := p.storage.GetUserData(context.Background(), username)
	if err != nil {
		return e.Wrap("can't get user data: %s", err)
	}

	if user.IsDeathAgeAsked && isNumber(text) {
		p.processAge(chatID, username, text)
	}

	if user.IsBirthdayAsked && isNumber(text) {
		p.processBirthday(chatID, username, text)
	}

	switch text {
	case LifeCalendarCmd:
		return p.sendHelp(chatID)
	case HelpCmd:
		return p.sendHelp(chatID)
	case StartCmd:
		return p.sendHello(chatID)
	case ShowTimeLeftCmd:
		return p.sendHelp(chatID)
	case OpenNotebookCmd:
		return p.sendHello(chatID)
	case StartCalculateCmd:
		return p.sendGettingDeathAge(chatID, username)
	default:
		return p.tg.SendMessage(chatID, msgUnknownCommand, GetStaticKeyboard())
	}
}

func (p *Processor) processAge(chatID int, username string, text string) (err error) {
	defer func() { err = e.WrapIfErr("can't do command: processAge", err) }()

	age, err := strconv.Atoi(text)
	if err != nil {
		return fmt.Errorf("invalid death age %q: %w", text, err)
	}

	page := &storage.User{
		UserName:        username,
		IsDeathAgeAsked: false,
		IsBirthdayAsked: true,
		DeathAge:        age,
	}

	if err := p.storage.SaveUser(context.Background(), page); err != nil {
		return err
	}

	var markup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("60"),
			tgbotapi.NewKeyboardButton("70"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("80"),
			tgbotapi.NewKeyboardButton("90"),
		),
	)

	if err := p.tg.SendMessage(chatID, "Please write down your birthday. For example 14.09.2002", markup); err != nil {
		return err
	}

	return nil
}

func (p *Processor) processBirthday(chatID int, username string, text string) (err error) {
	defer func() { err = e.WrapIfErr("can't do command: processBirthday", err) }()

	// Парсим дату в формате dd.mm.yyyy
	birthday, err := time.Parse("02.01.2006", text)
	if err != nil {
		return fmt.Errorf("invalid birthday %q: %w", text, err)
	}

	page := &storage.User{
		UserName:        username,
		IsDeathAgeAsked: false,
		IsBirthdayAsked: true,
		BirthsDay:       birthday,
	}

	if err := p.storage.SaveUser(context.Background(), page); err != nil {
		return err
	}

	if err := p.tg.SendMessage(chatID, fmt.Sprintf("Saved, your death date is: %s", birthday.Format("02.01.2006"))); err != nil {
		return err
	}

	return nil
}

func (p *Processor) sendGettingDeathAge(chatID int, username string) (err error) {
	defer func() { err = e.WrapIfErr("can't do command: can't get death age", err) }()

	page := &storage.User{
		UserName:        username,
		IsDeathAgeAsked: true,
		IsBirthdayAsked: false,
	}

	if err := p.storage.SaveUser(context.Background(), page); err != nil {
		return err
	}

	var markup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("60"),
			tgbotapi.NewKeyboardButton("70"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("80"),
			tgbotapi.NewKeyboardButton("90"),
		),
	)

	if err := p.tg.SendMessage(chatID, "Please select your expected lifespan", markup); err != nil {
		return err
	}

	return nil
}

func (p *Processor) sendHelp(chatID int) error {
	return p.tg.SendMessage(chatID, msgHelp, GetStaticKeyboard())
}

func (p *Processor) sendHello(chatID int) error {
	return p.tg.SendMessage(chatID, msgHello, GetStaticKeyboard())
}

func isNumber(text string) bool {
	if _, err := strconv.ParseFloat(text, 64); err == nil {
		return true
	}

	if _, err := strconv.Atoi(text); err == nil {
		return true
	}

	return false
}
