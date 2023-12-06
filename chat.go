package gptease

import (
	"context"
	"errors"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

const DEFAULT_CHAT_MODEL = openai.GPT4TurboPreview

var (
	ErrContentFilter      = errors.New("response omitted due to content filter")
	ErrNotFinished        = errors.New("response generation not finished")
	ErrTokenLimit         = errors.New("token limit reached")
	ErrUnexpectedResponse = errors.New("unexpected response from OpenAI API")
)

type Dialogue []openai.ChatCompletionMessage

// ChatTweaks contains parameters that can be changed to alter the behavior of
// the AI, such as how random the responses should be. If parameters are not
// set, default values will be used by the API.
type ChatTweaks struct {
	// Temperature sets the corresponding parameter in the API call to OpenAI,
	// documented as follows:
	//
	// What sampling temperature to use, between 0 and 2. Higher values like
	// 0.8 will make the output more random, while lower values like 0.2 will
	// make it more focused and deterministic.
	//
	// We generally recommend altering this or top_p but not both.
	Temperature float32

	// TopP sets the corresponding parameter in the API call to OpenAI,
	// documented as follows:
	//
	// An alternative to sampling with temperature, called nucleus sampling,
	// where the model considers the results of the tokens with top_p
	// probability mass. So 0.1 means only the tokens comprising the top 10%
	// probability mass are considered.
	//
	// We generally recommend altering this or temperature but not both.
	TopP float32
}

// Chat is a wrapper around the OpenAI API that makes it easier to have a
// conversation with an AI. It keeps track of the dialogue and the parameters
// to use for the API calls.
type Chat struct {
	// Dialogue contains the messages exchanged between the user and the AI
	// thus far. It can be modified directly, but often it's more convenient
	// to use methods like UserSaid and AssistantSaid. The Dialogue will be
	// automatically updated when calling functions like Exchange or Talk.
	Dialogue Dialogue

	// Model is the name of the model to use for the chat. If empty, the
	// DEFAULT_MODEL will be used.
	Model string

	// Tweaks contains the parameters to use for the chat. If empty, the
	// default parameters will be used.
	Tweaks ChatTweaks

	// Tools contains the definitions of any functions that the AI may make
	// callbacks to. These can easily be created directly from Go functions
	// using the MakeTool function.
	Tools []Tool

	c *openai.Client
}

func (c *Chat) client() (client *openai.Client, err error) {
	if c.c != nil {
		return c.c, nil
	}
	return DefaultClient()
}

func (c *Chat) model() string {
	if c.Model != "" {
		return c.Model
	}
	return DEFAULT_CHAT_MODEL
}

// Talk asks the AI to generate a response to the dialogue so far. It returns
// the response or an error. The response is automatically added to the
// dialogue.
//
// If the chat has tools available for the AI to invoke, Talk will handle such
// invocations automatically, making multiple API calls as needed.
func (c *Chat) Talk() (response string, err error) {
	var tools = make([]openai.Tool, len(c.Tools))
	for i, t := range c.Tools {
		tools[i] = t.openaiTool()
	}
	for {
		client, err := c.client()
		if err != nil {
			return "", err
		}
		resp, err := client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model:       c.model(),
				Messages:    c.Dialogue,
				Temperature: c.Tweaks.Temperature,
				TopP:        c.Tweaks.TopP,
				Tools:       tools,
			},
		)
		if err != nil {
			return "", err
		}
		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("%w: OpenAI API returned no choices", ErrUnexpectedResponse)
		}
		switch resp.Choices[0].FinishReason {
		case openai.FinishReasonFunctionCall:
			return "", fmt.Errorf("%w: deprecated function call returned by API", ErrUnexpectedResponse)
		case openai.FinishReasonToolCalls:
			var calls = resp.Choices[0].Message.ToolCalls
			if len(calls) == 0 {
				return "", fmt.Errorf("%w: no calls provided", ErrUnexpectedResponse)
			}
			c.Dialogue = append(c.Dialogue, resp.Choices[0].Message)
			for _, call := range calls {
				var toolErr error
				var out string
				if call.Type != "function" {
					toolErr = fmt.Errorf("error: unknown tool call type %s", call.Type)
				} else {
					var found bool
					for _, t := range c.Tools {
						if t.Name == call.Function.Name {
							out, toolErr = t.Handler(call.Function.Arguments)
							found = true
							break
						}
					}
					if !found {
						toolErr = fmt.Errorf("error: no tool found with name %s", call.Function.Name)
					}
				}
				var content string
				switch {
				case toolErr != nil:
					content = toolErr.Error()
				default:
					content = out
				}
				c.Dialogue = append(c.Dialogue, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    content,
					ToolCallID: call.ID,
				})
			}
			// Invoke the AI once again, now with the tool outputs added
			// to the dialogue.
			continue
		case openai.FinishReasonContentFilter:
			return "", ErrContentFilter
		case openai.FinishReasonNull:
			return "", ErrNotFinished

			// On "stop" or "length", we continue to return the response.
		}

		response = resp.Choices[0].Message.Content
		// Add the response from the AI to the dialogue.
		c.Dialogue = append(c.Dialogue, resp.Choices[0].Message)
		return response, nil
	}
}

// Exchange adds a message from the user to the dialogue and asks the AI to
// generate a response. If there was an error, the dialogue is not modified.
func (c *Chat) Exchange(content string) (response string, err error) {
	if content == "" {
		return "", fmt.Errorf("empty content")
	}
	var dlen = len(c.Dialogue)
	// Add the user's message to the dialogue.
	c.UserSaid(content)
	if resp, err := c.Talk(); err != nil {
		// Reset the dialogue to how it was before the call to Exchange.
		c.Dialogue = c.Dialogue[:dlen]
		return "", err
	} else {
		return resp, nil
	}
}

// MustExchange is like Exchange, but panics if there was an error. It is not
// recommended for normal use, but can be convenient for quick hacks when
// testing things out and running your program manually from command line.
func (c *Chat) MustExchange(content string) string {
	var resp, err = c.Exchange(content)
	if err != nil {
		panic(fmt.Sprintf("Chat error: %v\n", err))
	}
	return resp
}

// AssistantSaid adds a message to the dialogue as if said by the AI.
func (c *Chat) AssistantSaid(msg string) {
	c.Dialogue = append(c.Dialogue, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: msg,
	})
}

// UserSaid adds a message to the dialogue said by the user.
func (c *Chat) UserSaid(msg string) {
	c.Dialogue = append(c.Dialogue, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: msg,
	})
}

// ExampleExchange is a convenience function that adds a message from the user
// and a response from the AI to the dialogue. It can be used to guide the AI
// to respond in a certain way.
func (c *Chat) ExampleExchange(input, response string) {
	c.UserSaid(input)
	c.AssistantSaid(response)
}

// Instruction adds a message to the dialogue as from the "system". This is
// typically used to give the AI instructions, for example what role it should
// emulate.
func (c *Chat) Instruction(txt string) {
	c.Dialogue = append(c.Dialogue, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: txt,
	})
}
