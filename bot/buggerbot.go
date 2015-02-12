package bot

import (
	"log"
	"sync"
	"time"

	"github.com/andybons/hipchat"
)

type BuggerBotProtocol struct {
	Message    string
	Recipients []string
}

type buggerBot struct {
	Buggers    []func() (ch chan BuggerBotProtocol, nextPoll int)
	ChatClient hipchat.Client
}

func BuggerBot(client hipchat.Client) buggerBot {
	return buggerBot{[]func() (ch chan BuggerBotProtocol, nextPoll int){}, client}
}
func (b *buggerBot) Register(fn func() (ch chan BuggerBotProtocol, nextPoll int)) {
	b.Buggers = append(b.Buggers, fn)
	log.Println(b.Buggers)
}

func (b *buggerBot) Run() {
	var wg sync.WaitGroup
	log.Println(b.Buggers)
	for _, bugger := range b.Buggers {
		wg.Add(1)
		log.Println("adding bugger")
		go func(fn func() (ch chan BuggerBotProtocol, nextPoll int)) {
			for {
				bbps, nextPoll := fn()
				for bbp := range bbps {
					err := b.sendMessage(bbp.Message, bbp.Recipients)
					if err != nil {

						log.Println(err)
					}
				}
				time.Sleep(time.Second * time.Duration(nextPoll))
			}
		}(bugger)
	}
	wg.Wait()

}

func (b *buggerBot) sendMessage(message string, recipients []string) error {

	for _, recipient := range recipients {
		req := hipchat.MessageRequest{
			RoomId:        recipient,
			From:          "BuggerBot",
			Message:       message,
			Color:         hipchat.ColorPurple,
			MessageFormat: hipchat.FormatText,
			Notify:        true,
		}
		if err := b.ChatClient.PostMessage(req); err != nil {
			return err
		}
	}
	return nil
}
