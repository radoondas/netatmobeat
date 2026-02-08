package beater

import "context"

// TestAuth validates authentication by attempting InitializeTokenState.
// Creates its own context for HTTP requests.
// Used by the "test auth" CLI command. Returns nil on success.
func (bt *Netatmobeat) TestAuth() error {
	bt.ctx, bt.cancel = context.WithCancel(context.Background())
	defer bt.cancel()
	return bt.InitializeTokenState()
}
