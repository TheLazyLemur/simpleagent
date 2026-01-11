package tools

import (
	"encoding/json"
	"fmt"

	"simpleagent/claude"
)

// SkillInfo contains loaded skill data
type SkillInfo struct {
	Name         string
	AllowedTools []string
	Dir          string
	Content      string
}

// SkillLoader loads a skill by name - set from main
var SkillLoader func(name string) (*SkillInfo, error)

func init() {
	register(claude.Tool{
		Name:        "InvokeSkill",
		Description: "Load a skill by name. Use when task matches a skill description.",
		InputSchema: claude.InputSchema{
			Type: "object",
			Properties: map[string]claude.Property{
				"name": {Type: "string", Description: "Skill name to invoke"},
				"args": {Type: "string", Description: "Optional arguments"},
			},
			Required: []string{"name"},
		},
	}, invokeSkill)
}

type skillResult struct {
	name    string
	content string
}

func (r skillResult) String() string { return r.content }
func (r skillResult) Render() {
	fmt.Printf("\n%s %s\n", infoBadge.Render("skill"), dimStyle.Render(r.name))
}

func invokeSkill(input json.RawMessage) Result {
	var args struct {
		Name string `json:"name"`
		Args string `json:"args"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return newResult("InvokeSkill", Error(err.Error()))
	}

	if args.Name == "" {
		return newResult("InvokeSkill", Error("name required"))
	}

	if SkillLoader == nil {
		return newResult("InvokeSkill", Error("skill loader not configured"))
	}

	skill, err := SkillLoader(args.Name)
	if err != nil {
		return newResult("InvokeSkill", Error("skill not found: "+args.Name))
	}

	// Set tool restrictions if specified
	if len(skill.AllowedTools) > 0 {
		SetAllowedTools(skill.AllowedTools)
	}

	// Build output with skill content
	output := skill.Content
	if args.Args != "" {
		output = "Arguments: " + args.Args + "\n\n" + output
	}
	if skill.Dir != "" {
		output = "Skill directory: " + skill.Dir + "\n\n" + output
	}

	return skillResult{name: skill.Name, content: output}
}
