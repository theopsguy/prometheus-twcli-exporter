package shell

// Shell An interface for running shell commands in the OS
type Shell interface {
	Execute(binary string, args ...string) ([]byte, error)
}
