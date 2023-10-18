package listener

import (
	"go-gin-gorm-example/infrastructure/config"
	"go-gin-gorm-example/module/article"
	"go-gin-gorm-example/utils"

	"github.com/gookit/event"
)

type Listener struct {
	articleHttp article.InterfaceHttp
}

// NewListener should be call from main.go
// and accept struct handler from boot make handler
// because need the http interface should be called first
// to do action, so we can just call from handler -> service -> repository
func NewListener(articleHttp article.InterfaceHttp) Listener {
	return Listener{
		articleHttp: articleHttp,
	}
}

// ListenForShutdownEvent listen on the shutdown event
// look utils/ShutDownEvent constant.
func (l *Listener) ListenForShutdownEvent() {
	event.On(utils.ShutDownEvent, event.ListenerFunc(func(e event.Event) error {
		// TriggerShutdown wrapping action for the shutdown event
		l.TriggerShutdown()
		return nil
	}))
}

// TriggerShutdown sends a signal to the repository and performs shutdown actions.
func (l *Listener) TriggerShutdown() {
	//need to call save in memory data to json file
	if !config.Conf.Postgres.EnablePostgres {
		l.articleHttp.SaveToFile()
	}
}

// TriggerStartUp sends a signal to the repository and performs start up actions.
// this call should be not initiated on event because we can just call it on the main.go
func (l *Listener) TriggerStartUp() {
	if !config.Conf.Postgres.EnablePostgres {
		l.articleHttp.LoadFromFile()
	}
}
