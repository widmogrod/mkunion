package projection

import (
	log "github.com/sirupsen/logrus"
)

func Log(prefix string) Handler {
	return &LogHandler{
		prefix: prefix,
	}
}

type LogHandler struct {
	prefix string
}

func (l *LogHandler) Process(x Item, returning func(Item)) error {
	log.
		WithField("key", x.Key).
		WithField("item", ToStrItem(&x)).
		Infof("%s: Process \n", l.prefix)

	returning(x)
	return nil
}

func (l *LogHandler) Retract(x Item, returning func(Item)) error {
	log.
		WithField("key", x.Key).
		WithField("item", ToStrItem(&x)).
		Infof("%s: Retract \n", l.prefix)

	returning(x)
	return nil
}
