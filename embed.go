package gptease

import (
	"context"

	openai "github.com/sashabaranov/go-openai"
)

type Embedding []float32

// Dot computes the dot product of two embeddings.
//
// When the vectors are normalized, the dot product is the cosine similarity.
// This is typically the case unless you've generated your own or done some
// arithmetic on them.
func (e Embedding) Dot(other Embedding) float32 {
	var sum float32
	for i, x := range e {
		sum += x * other[i]
	}
	return sum
}

// Embed computes a vector embedding of a text string.
//
// Aside from the embedding vector, it returns the number of tokens found in
// the text. This can be useful to know how large the text is in the eyes of
// the AI, for example when using the embedding for Retrieval Augmented
// Generation (RAG).
func Embed(text string) (v Embedding, tokenCount int, err error) {
	client, err := DefaultClient()
	if err != nil {
		return nil, 0, err
	}
	resp, err := client.CreateEmbeddings(
		context.Background(),
		openai.EmbeddingRequest{
			Model: openai.AdaEmbeddingV2,
			Input: []string{text},
		},
	)
	if err != nil {
		return nil, 0, err
	}
	return Embedding(resp.Data[0].Embedding), resp.Usage.PromptTokens, nil
}
