package telegram

import (
	"context"
	"errors"
	"log"
	"net/url"
	"strconv"
	"strings"

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

	if isAddCmd(text) {
		return p.savePage(chatID, text, username)
	}

	if isNumber(text) {
		return p.processAge(chatID, text, username)
	}

	switch text {
	case LifeCalendarCmd:
		return p.sendRandom(chatID, username)
	case HelpCmd:
		return p.sendHelp(chatID)
	case StartCmd:
		return p.sendHello(chatID)
	case ShowTimeLeftCmd:
		return p.sendHelp(chatID)
	case OpenNotebookCmd:
		return p.sendHello(chatID)
	case StartCalculateCmd:
		return p.sendGettingDeathAge(chatID, text, username)
	default:
		return p.tg.SendMessage(chatID, msgUnknownCommand, GetStaticKeyboard())
	}
}

func (p *Processor) processAge(chatID int, pageURL string, username string) (err error) {
	defer func() { err = e.WrapIfErr("can't do command: save page", err) }()

	page := &storage.Page{
		URL:      pageURL,
		UserName: username,
	}

	isExists, err := p.storage.IsExists(context.Background(), page)
	if err != nil {
		return err
	}
	if isExists {
		return p.tg.SendMessage(chatID, msgAlreadyExists)
	}

	if err := p.storage.Save(context.Background(), page); err != nil {
		return err
	}

	if err := p.tg.SendMessage(chatID, msgSaved); err != nil {
		return err
	}

	return nil
}

func (p *Processor) savePage(chatID int, pageURL string, username string) (err error) {
	defer func() { err = e.WrapIfErr("can't do command: save page", err) }()

	page := &storage.Page{
		URL:      pageURL,
		UserName: username,
	}

	isExists, err := p.storage.IsExists(context.Background(), page)
	if err != nil {
		return err
	}
	if isExists {
		return p.tg.SendMessage(chatID, msgAlreadyExists)
	}

	if err := p.storage.Save(context.Background(), page); err != nil {
		return err
	}

	if err := p.tg.SendMessage(chatID, msgSaved); err != nil {
		return err
	}

	return nil
}

func (p *Processor) sendRandom(chatID int, username string) (err error) {
	defer func() { err = e.WrapIfErr("can't do command: can't send random", err) }()

	page, err := p.storage.PickRandom(context.Background(), username)
	if err != nil && !errors.Is(err, storage.ErrNoSavedPages) {
		return err
	}
	if errors.Is(err, storage.ErrNoSavedPages) {
		return p.tg.SendMessage(chatID, msgNoSavedPages, GetStaticKeyboard())
	}

	if err := p.tg.SendMessage(chatID, page.URL, GetStaticKeyboard()); err != nil {
		return err
	}

	return p.storage.Remove(context.Background(), page)
}

func (p *Processor) sendGettingDeathAge(chatID int, pageURL string, username string) (err error) {
	defer func() { err = e.WrapIfErr("can't do command: can't get death age", err) }()

	page := &storage.Page{
		URL:             pageURL,
		UserName:        username,
		IsDeathAgeAsked: true,
	}

	if err := p.storage.Save(context.Background(), page); err != nil {
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

func isAddCmd(text string) bool {
	return isURL(text)
}

func isURL(text string) bool {
	u, err := url.Parse(text)

	return err == nil && u.Host != ""
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
