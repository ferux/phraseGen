package phrasegen

var (
	// Version of application
	Version string

	// Revision of application
	Revision string

	// Environment of application
	Environment string

	// LogLevel of the applications.
	LogLevel string

	// Config is a settings for the application
	Config Configuration

	// Port is a port for listening to gRPC service
	Port uint16
)

// Configuration of the file
type Configuration struct {
	ErrbitHost string
	ErrbitID   int64
	ErrbitKey  string
}

/*
	"github.com/namsral/flag"
*/
