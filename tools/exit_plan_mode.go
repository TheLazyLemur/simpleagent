package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/glamour"
	"simpleagent/claude"
)

var planRenderer *glamour.TermRenderer

type ExitPlanModeInput struct {
	Plan string `json:"plan"`
}

type ExitPlanModeDecision struct {
	Decision string `json:"decision"`
	Note     string `json:"note,omitempty"`
}

func init() {
	planRenderer, _ = glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)

	register(claude.Tool{
		Name:        "ExitPlanMode",
		Description: "Present a plan for user approval and potentially exit plan mode. Pass your complete plan as the 'plan' parameter. User sees the plan and chooses: Accept (exit plan mode, proceed), Deny (stay in plan mode, revise), or Continue (stay in plan mode, keep exploring).",
		InputSchema: claude.InputSchema{
			Type: "object",
			Properties: map[string]claude.Property{
				"plan": {
					Type:        "string",
					Description: "The plan to present to the user for approval",
				},
			},
			Required: []string{"plan"},
		},
	}, executeExitPlanMode)
}

type exitPlanModeResult struct {
	plan     string
	decision string
	note     string
}

func (r exitPlanModeResult) String() string {
	data, _ := json.Marshal(ExitPlanModeDecision{Decision: r.decision, Note: r.note})
	return string(data)
}

func (r exitPlanModeResult) Render() {
	fmt.Println()
	fmt.Println(Plan("plan review"))
	fmt.Println()
	fmt.Print(renderMarkdown(r.plan))
	fmt.Println(Plan("options"))
	fmt.Printf("  %s %s %s\n", OptionNumber(1), Highlight("Accept"), Dim("- exit plan mode, proceed"))
	fmt.Printf("  %s %s %s\n", OptionNumber(2), Highlight("Deny"), Dim("- stay, agent revises"))
	fmt.Printf("  %s %s %s\n", OptionNumber(3), Highlight("Continue"), Dim("- stay, keep exploring"))
}

func executeExitPlanMode(input json.RawMessage) Result {
	var args ExitPlanModeInput
	if err := json.Unmarshal(input, &args); err != nil {
		return newResult("ExitPlanMode", Error(err.Error()))
	}

	// Display the plan and get user decision
	reader := bufio.NewReader(os.Stdin)
	decision, note := promptDecision(reader, args.Plan)

	return exitPlanModeResult{plan: args.Plan, decision: decision, note: note}
}

func renderMarkdown(text string) string {
	if planRenderer != nil {
		rendered, err := planRenderer.Render(text)
		if err == nil {
			return rendered
		}
	}
	return text + "\n"
}

func promptDecision(reader *bufio.Reader, plan string) (string, string) {
	fmt.Println()
	fmt.Println(Plan("plan review"))
	fmt.Println()
	fmt.Print(renderMarkdown(plan))
	fmt.Println(Plan("options"))
	fmt.Printf("  %s %s %s\n", OptionNumber(1), Highlight("Accept"), Dim("- exit plan mode, proceed"))
	fmt.Printf("  %s %s %s\n", OptionNumber(2), Highlight("Deny"), Dim("- stay, agent revises"))
	fmt.Printf("  %s %s %s\n", OptionNumber(3), Highlight("Continue"), Dim("- stay, keep exploring"))
	fmt.Print("\n" + Prompt())

	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)

	num, err := strconv.Atoi(text)
	if err != nil || num < 1 || num > 3 {
		return "Continue", "" // Default to continue on invalid input
	}

	switch num {
	case 1:
		return "Accept", ""
	case 2:
		return "Deny", promptNote(reader)
	case 3:
		return "Continue", promptNote(reader)
	default:
		return "Continue", ""
	}
}

func promptNote(reader *bufio.Reader) string {
	fmt.Print(Prompt() + Dim("note (optional): "))
	note, _ := reader.ReadString('\n')
	return strings.TrimSpace(note)
}
