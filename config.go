package phraseGen

var (
	// Version of application
	Version string

	// Revision of application
	Revision string

	// Environment of application
	Environment string
)

// Config of the file
type Config struct {
	ErrbitHost string
	ErrbitID   int
	ErrbitKey  string
}
