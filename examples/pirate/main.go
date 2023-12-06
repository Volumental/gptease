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
