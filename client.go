package gptease

import (
	"fmt"
	"os"
	"sync"

	openai "github.com/sashabaranov/go-openai"
)

var defaultClient *openai.Client
var defaultClientOnce sync.Once

// SetDefaultClient sets the default OpenAI API client.
//
// You usually don't need to call this function.
func SetDefaultClient(client *openai.Client) {
	defaultClient = client
}

// DefaultClient returns the default OpenAI API client. It will use the
// OPENAI_API_KEY environment variable to get your API key.
func DefaultClient() (client *openai.Client, err error) {
	defaultClientOnce.Do(func() {
		if apikey := os.Getenv("OPENAI_API_KEY"); apikey == "" {
			err = fmt.Errorf("OPENAI_API_KEY environment variable not set")
			return
		} else {
			defaultClient = openai.NewClient(apikey)
		}
	})
	return defaultClient, err
}
