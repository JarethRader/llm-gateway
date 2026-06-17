package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/jarethrader/llm-gateway/gateway-service/test/fakevllm"
)

func main() {
	addr := flag.String("addr", ":11434", "")
	ttft := flag.Duration("ttft", 150*time.Millisecond, "")
	inter := flag.Duration("inter", 25*time.Millisecond, "")
	tokens := flag.Int("tokens", 40, "")
	model := flag.String("model", "m", "")
	flag.Parse()

	f := fakevllm.New(fakevllm.Behavior{
		Addr:       addr,
		TTFT:       *ttft,
		InterToken: *inter,
		Tokens:     *tokens,
		Model:      *model,
	})
	fmt.Printf("fake vllm on %s", f.URL())
	select {}
}
