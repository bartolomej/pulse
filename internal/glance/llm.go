package glance

import (
	"context"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/outputparser"
)

type LLM struct {
	model llms.Model
}

func NewLLM() (*LLM, error) {
	model, err := openai.New(
		openai.WithModel("gpt-4o-mini"),
	)
	if err != nil {
		return nil, err
	}
	return &LLM{model: model}, nil
}

type feedMatch struct {
	ID        string `json:"id"`
	Score     int    `json:"score"`
	Highlight string `json:"highlight"`
}

type completionResponse struct {
	Matches []feedMatch `json:"matches"`
}

// filterFeed returns the IDs of feed entries that match the query
func (llm *LLM) filterFeed(ctx context.Context, feed []feedEntry, query string) ([]feedMatch, error) {
	sb := strings.Builder{}

	sb.WriteString(`
You are an activity feed personalization assistant, 
that helps the user find and focus on the most relevant content.

You are given a list of feed entries with id, title, and description fields - given the natural language query,
you should return the list of feed entry IDs alongside the score and highlight (explanation for why its a good match) that best match the query.
`)
	sb.WriteString(fmt.Sprintf("filter query: %s\n", query))

	for _, entry := range feed {
		sb.WriteString(fmt.Sprintf("id: %s\n", entry.ID))
		sb.WriteString(fmt.Sprintf("title: %s\n", entry.Title))
		sb.WriteString(fmt.Sprintf("description: %s\n", entry.Description))
		sb.WriteString("\n")
	}

	return llm.structuredComplete(ctx, sb.String())
}

func (llm *LLM) structuredComplete(ctx context.Context, prompt string) ([]feedMatch, error) {
	parser, err := outputparser.NewDefined(completionResponse{})
	if err != nil {
		return nil, fmt.Errorf("creating parser: %w", err)
	}

	schema := parser.GetFormatInstructions()
	decoratedPrompt := fmt.Sprintf("%s\n\n%s", prompt, schema)

	fmt.Printf("decoratedPrompt: %s\n", decoratedPrompt)

	out, err := llms.GenerateFromSinglePrompt(
		ctx,
		llm.model,
		decoratedPrompt,
	)
	if err != nil {
		return nil, fmt.Errorf("generating completion: %w", err)
	}

	fmt.Printf("out: %s\n", out)

	response, err := parser.Parse(out)
	if err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return response.Matches, nil
}
