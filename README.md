GPTease - GPT made easy
=======================

This library is designed to make it as easy as possible to get started using ChatGPT and related OpenAI products in Go. For a more comprehensive library, you might want to consider something like [go-openai](https://github.com/sashabaranov/go-openai) instead (or use them side by side). This one is for the lazy coder who just wants to get going without learning all the details.

### Goals

* Make the most common use cases straightforward.
* Developer ergonomics.

### Non-goals

* Feature parity with OpenAI's API.

Getting started
---------------

Put your OpenAI API key in the environment variable named OPENAI_API_KEY like this:

```sh
export OPENAI_API_KEY=your_api_key_here
```

Or you may use a library such as [godotenv](https://github.com/joho/godotenv) to keep it in a file when you develop.

Then compile and run the following program:

```go
package main

import (
    "fmt"

    "github.com/Volumental/gptease"
)

func main() {
    var chat gptease.Chat
    chat.Instruction("Talk like a pirate. A cool pirate.")
    msg := chat.MustExchange("Tell me how to cook scrambled eggs.")
    fmt.Println(msg)
}
```

Notable features
----------------

You can easily let the AI call Go functions and use them as tools to generate a response. All that is required is for them to have a signature like `func(struct) (any, error)`.

It looks like this (full example [here](examples/gm/main.go)):

```go
type rollDieArgs struct { MaxValue int }

chat := gptease.Chat{
    Tools: []gptease.Tool{
        gptease.MakeTool(
            func(args rollDieArgs) (int, error) {
                rand.Seed(time.Now().UnixNano())
                return rand.Intn(args.MaxValue) + 1, nil
            },
            "rollDie",
            "Returns a random number between 1 and MaxValue (inclusive).",
        ),
    },
}

chat.Instruction("You are GM of a role playing game.")
msg := chat.MustExchange("I swing my sword against the goblin, for 1d10 damage.")
fmt.Println(msg)
```

You can provide more details on how to call the function by adding field tags to the argument struct. For example, you might have an argument struct like this:

```go
type rollDiceArgs struct {
    Num     int    `json:"num" desc:"The number of dies to roll."`
    Max     int    `json:"max" desc:"The highest value an individual die can show."`
    Type    string `json:"type,omitempty" desc:"Wheather to cheat." enum:"loaded,fair"`
}
```

It will use the `json` tag for field names, and to know whether a field is required. You may also add a `desc` tag to describe the meaning of the field, and an `enum` tag to indicate allowed values. This information will be sent to the AI as a JSON Schema to help it understand how to use the function.

Final words
-----------

Note that this is still under active development. Things might change that breaks your code when you upgrade to a newer version. Hopefully it'll be easy enough to adjust until our API has stabilised.
