package breaking

import (
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
)

// TestMain quiets the inferrer's info and warning logs, as the CLI does.
func TestMain(m *testing.M) {
	log.SetLevel(log.ErrorLevel)
	os.Exit(m.Run())
}
