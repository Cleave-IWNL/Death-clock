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
			UserName:        &username,
			IsDeathAgeAsked: BoolPtr(false),
			IsBirthdayAsked: BoolPtr(false),
		}

		p.storage.InitUser(context.Background(), user)
	}

	user, err := p.storage.GetUserData(context.Background(), username)
	if err != nil {
		return e.Wrap("can't get user data: %s", err)
	}

	if *user.IsDeathAgeAsked && isNumber(text) {
		p.processAge(chatID, username, text)

		return nil
	}

	if *user.IsBirthdayAsked && isValidDate(text) {
		p.processBirthday(chatID, username, text)
		return nil
	}

	switch text {
	case LifeCalendarCmd:
		return p.sendHelp(chatID)
	case HelpCmd:
		return p.sendHelp(chatID)
	case StartCmd:
		return p.sendHello(chatID)
	case ShowTimeLeftCmd:
		return p.processTimeLeft(chatID, username)
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
		UserName:        &username,
		IsDeathAgeAsked: BoolPtr(false),
		IsBirthdayAsked: BoolPtr(true),
		DeathAge:        &age,
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

func (p *Processor) processTimeLeft(chatID int, username string) (err error) {
	defer func() { err = e.WrapIfErr("can't do command: processTimeLeft", err) }()

	user, err := p.storage.GetUserData(context.Background(), username)
	if err != nil {
		return err
	}

	if user.BirthsDay == nil {
		if err := p.tg.SendMessage(chatID, "Please complete the calculation first by clicking 'Start'"); err != nil {
			return err
		}

		return nil
	}

	deathDate := user.BirthsDay.AddDate(*user.DeathAge, 0, 0)

	now := time.Now()
	if deathDate.Before(now) {
		if err := p.tg.SendMessage(chatID, "Эээ... похоже, вы уже должны быть мертвы 🤔"); err != nil {
			return err
		}
	}

	diff := deathDate.Sub(now)

	days := int(diff.Hours() / 24)
	weeks := days / 7

	yearsLeft, monthsLeft := calendarDiff(now, deathDate)

	msg := fmt.Sprintf(
		"Дата смерти: *%s*\n\n"+
			"Осталось:\n"+
			"- %d лет\n"+
			"- %d месяцев\n"+
			"- %d недель\n"+
			"- %d дней",
		deathDate.Format("02.01.2006"),
		yearsLeft,
		yearsLeft*12+monthsLeft,
		weeks,
		days,
	)

	if err := p.tg.SendMessage(chatID, msg); err != nil {
		return err
	}

	return nil
}

func (p *Processor) processBirthday(chatID int, username string, text string) (err error) {
	defer func() { err = e.WrapIfErr("can't do command: processBirthday", err) }()

	birthday, err := time.Parse("02.01.2006", text)
	if err != nil {
		return fmt.Errorf("invalid birthday %q: %w", text, err)
	}

	user, err := p.storage.GetUserData(context.Background(), username)
	if err != nil {
		return err
	}

	if user.DeathAge == nil {
		return fmt.Errorf("death age not set for user %s", username)
	}

	deathDate := birthday.AddDate(*user.DeathAge, 0, 0)

	now := time.Now()
	if deathDate.Before(now) {
		if err := p.tg.SendMessage(chatID, "Эээ... похоже, вы уже должны быть мертвы 🤔"); err != nil {
			return err
		}
	}

	diff := deathDate.Sub(now)

	days := int(diff.Hours() / 24)
	weeks := days / 7

	yearsLeft, monthsLeft := calendarDiff(now, deathDate)

	user.BirthsDay = &birthday
	user.ExpectedDeathDate = &deathDate
	user.IsBirthdayAsked = BoolPtr(true)

	if err := p.storage.SaveUser(context.Background(), user); err != nil {
		return err
	}

	msg := fmt.Sprintf(
		"Дата смерти: *%s*\n\n"+
			"Осталось:\n"+
			"- %d лет\n"+
			"- %d месяцев\n"+
			"- %d недель\n"+
			"- %d дней",
		deathDate.Format("02.01.2006"),
		yearsLeft,
		yearsLeft*12+monthsLeft,
		weeks,
		days,
	)

	if err := p.tg.SendMessage(chatID, msg); err != nil {
		return err
	}

	return nil
}

func calendarDiff(start, end time.Time) (years int, months int) {
	years = end.Year() - start.Year()
	months = int(end.Month()) - int(start.Month())

	if end.Day() < start.Day() {
		months--
	}

	if months < 0 {
		years--
		months += 12
	}

	return
}

func (p *Processor) sendGettingDeathAge(chatID int, username string) (err error) {
	defer func() { err = e.WrapIfErr("can't do command: can't get death age", err) }()

	page := &storage.User{
		UserName:        &username,
		IsDeathAgeAsked: BoolPtr(true),
		IsBirthdayAsked: BoolPtr(false),
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

func BoolPtr(b bool) *bool { return &b }

func isValidDate(s string) bool {
	_, err := time.Parse("02.01.2006", s)
	return err == nil
}
