package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/Volumental/gptease"
)

type rollDieArgs struct{ MaxValue int }

func RollDie(args rollDieArgs) (int, error) {
	rand.Seed(time.Now().UnixNano())
	v := rand.Intn(args.MaxValue) + 1
	fmt.Printf("[Rolled a %v]\n", v)
	return v, nil
}

func main() {
	chat := gptease.Chat{
		Tools: []gptease.Tool{
			gptease.MakeTool(
				RollDie,
				"rollDie",
				"Returns a random number between 1 and MaxValue (inclusive).",
			),
		},
	}

	chat.Instruction("You are GM of a role playing game.")
	msg := chat.MustExchange("I swing my sword against the goblin, for 1d10 damage.")
	fmt.Println(msg)
}
