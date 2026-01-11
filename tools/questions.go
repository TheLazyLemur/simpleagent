package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"simpleagent/claude"
)

type Option struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

type Question struct {
	Question    string   `json:"question"`
	Header      string   `json:"header"`
	Options     []Option `json:"options"`
	MultiSelect bool     `json:"multiSelect"`
}

var pendingQuestions []Question
var questionAnswers map[string]string

func init() {
	register(claude.Tool{
		Name:        "AskUserQuestion",
		Description: "Ask the user questions to gather preferences, clarify ambiguous instructions, or get decisions on implementation choices.",
		InputSchema: claude.InputSchema{
			Type: "object",
			Properties: map[string]claude.Property{
				"questions": {
					Type:        "array",
					Description: "Questions to ask (1-4)",
					Items: &claude.Property{
						Type: "object",
						Properties: map[string]claude.Property{
							"question":    {Type: "string", Description: "The question to ask"},
							"header":      {Type: "string", Description: "Short label (max 12 chars)"},
							"multiSelect": {Type: "boolean", Description: "Allow multiple selections"},
							"options": {
								Type:        "array",
								Description: "Available choices (2-4 options)",
								Items: &claude.Property{
									Type: "object",
									Properties: map[string]claude.Property{
										"label":       {Type: "string", Description: "Option display text (1-5 words)"},
										"description": {Type: "string", Description: "Explanation of option"},
									},
								},
							},
						},
					},
				},
			},
			Required: []string{"questions"},
		},
	}, executeQuestion)
}

type questionResult struct {
	questions []Question
	answers   map[string]string
}

func (r questionResult) String() string {
	if r.answers != nil {
		data, _ := json.Marshal(r.answers)
		return string(data)
	}
	return `{"status":"awaiting_input"}`
}

func (r questionResult) Render() {
	renderQuestions(r.questions)
}

func executeQuestion(input json.RawMessage) Result {
	var args struct {
		Questions []Question `json:"questions"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return newResult("AskUserQuestion", Error(err.Error()))
	}
	pendingQuestions = args.Questions
	questionAnswers = make(map[string]string)

	// Collect answers interactively
	reader := bufio.NewReader(os.Stdin)
	for _, q := range args.Questions {
		answer := promptQuestion(reader, q)
		questionAnswers[q.Header] = answer
	}

	pendingQuestions = nil
	return questionResult{questions: args.Questions, answers: questionAnswers}
}

func renderQuestions(questions []Question) {
	fmt.Println()
	for _, q := range questions {
		fmt.Printf("  %s %s\n", Status(q.Header), q.Question)
		for i, opt := range q.Options {
			fmt.Printf("    %s %s %s\n", OptionNumber(i+1), Highlight(opt.Label), Dim("- "+opt.Description))
		}
	}
}

func promptQuestion(reader *bufio.Reader, q Question) string {
	fmt.Printf("\n%s %s\n", Status(q.Header), q.Question)
	for i, opt := range q.Options {
		fmt.Printf("  %s %s %s\n", OptionNumber(i+1), Highlight(opt.Label), Dim("- "+opt.Description))
	}
	fmt.Printf("  %s %s\n", OptionNumber(len(q.Options)+1), Dim("Other (custom answer)"))

	if q.MultiSelect {
		fmt.Print(Prompt() + Dim("numbers, comma-separated: "))
	} else {
		fmt.Print(Prompt())
	}

	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)

	if q.MultiSelect {
		return parseMultiSelect(text, q.Options)
	}
	return parseSingleSelect(text, q.Options, reader)
}

func parseSingleSelect(text string, options []Option, reader *bufio.Reader) string {
	num, err := strconv.Atoi(text)
	if err != nil || num < 1 || num > len(options)+1 {
		return text // treat as custom input
	}
	if num == len(options)+1 {
		fmt.Print(Prompt() + Dim("custom: "))
		custom, _ := reader.ReadString('\n')
		return strings.TrimSpace(custom)
	}
	return options[num-1].Label
}

func parseMultiSelect(text string, options []Option) string {
	parts := strings.Split(text, ",")
	var selected []string
	for _, p := range parts {
		num, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil || num < 1 || num > len(options) {
			continue
		}
		selected = append(selected, options[num-1].Label)
	}
	return strings.Join(selected, ", ")
}

// GetPendingQuestions returns questions awaiting answers
func GetPendingQuestions() []Question {
	return pendingQuestions
}
